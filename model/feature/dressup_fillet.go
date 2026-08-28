// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Dress-up features — the FILLET definition (M48 #2233 split of dressup.go). The edge/face fillet
// feature: its edge sets (constant or variable radius with intermediate points), corner/cross-section
// modes, definition and the feature wrapper whose Recompute rebinds the picked edges and defers to the
// kernel blend. The DressUpFeatures adder collection stays in dressup.go.

// FilletEdgeSet is one edge set of a fillet definition (the reference's
// FilletConstantRadiusEdgeSet / FilletVariableRadiusEdgeSet): a constant Radius over the
// whole set, or — when Radius is nil — a variable StartRadius→EndRadius over a single edge
// (radius runs linearly from the edge's start vertex to its end vertex). RadiusPoints are
// optional intermediate (position, radius) stops along that variable edge (#695), each at a
// fraction 0<T<1 of the edge, strictly increasing in T — the reference's FilletRadius set points.
type FilletEdgeSet struct {
	EdgeKeys     [][]byte
	Radius       func() float64
	StartRadius  func() float64
	EndRadius    func() float64
	RadiusPoints []FilletRadiusPoint
}

// FilletRadiusPoint is one intermediate radius stop on a variable fillet edge: T is the fraction
// along the edge (start vertex = 0, end vertex = 1), Radius the rolling-ball radius there.
type FilletRadiusPoint struct {
	T      float64
	Radius func() float64
}

// variable reports whether the set carries a start→end radius instead of a constant one.
func (s FilletEdgeSet) variable() bool { return s.Radius == nil }

// FilletCornerType aliases the public corner-treatment discriminator (ADR-0018).
type FilletCornerType = types.FilletCornerType

// FilletCrossSection aliases the public blend cross-section shape (M36-F08, ADR-0018): arc (G1,
// default), G2 (curvature-continuous), or conic (rho-controlled).
type FilletCrossSection = types.FilletCrossSection

// Fillet cross-section shapes (aliases of the canonical api/types values).
const (
	FilletArc   = types.FilletSectionArc
	FilletG2    = types.FilletSectionG2
	FilletConic = types.FilletSectionConic
)

// FilletDefinition rounds selected edges. EdgeKeys+Radius is the original single
// constant-radius form; EdgeSets (when non-empty) takes precedence and carries any mix of
// constant and variable sets (#323). CornerType selects how a vertex where two filleted edges
// meet (third edge sharp) is treated — miter crease (default/zero), or a full round.
type FilletDefinition struct {
	EdgeKeys        [][]byte
	Radius          func() float64
	EdgeSets        []FilletEdgeSet
	CornerType      FilletCornerType
	ConcaveStrategy types.FilletConcaveStrategy // concave edges: outward fill (zero/default) or inward recess
	// CrossSection selects the blend cross-section shape (M36-F08): the default arc (G1), a
	// curvature-continuous G2, or a rho-controlled conic. Rho sets a conic's fullness (0<ρ<1,
	// 0.5=parabola). Non-arc sections build via the swept ruling band (no analytic cylinder).
	CrossSection FilletCrossSection
	Rho          float64
	// GeomEdges are edges selected by a serialized GEOMETRIC descriptor rather than an
	// Oblikovati lineage key — the path an external author (the NX exporter, M8/ADR-0040)
	// uses, since it cannot synthesize lineage. They bind to the running body's edges at
	// recompute (see bindGeomEdges) and fold into the edge list; absent for a normal
	// Oblikovati-authored fillet.
	GeomEdges []topo.GeometricEdgeRef
	// EdgeAnchors maps an EdgeKeys entry (raw reference key, as a string) to the edge's
	// midpoint captured when the user picked it. It feeds the GEOMETRIC recovery tier
	// (ADR-0043 P6b): when a lost key's parent has several surviving siblings, the anchor
	// disambiguates by nearness. Absent for an older recipe or an edit-mode retained key —
	// such a reference degrades to exact/ancestral recovery only.
	EdgeAnchors map[string]math.Point3
}

// FilletType reports the definition's discriminator: always an edge fillet for now (the
// reference's face and full-round fillets are follow-ups tracked on #323).
func (d *FilletDefinition) FilletType() types.FilletType { return types.EdgeFillet }

// FilletFeature is an edge fillet over one or more constant/variable radius edge sets.
type FilletFeature struct{ def *FilletDefinition }

// Definition returns the fillet recipe.
func (f *FilletFeature) Definition() *FilletDefinition { return f.def }

// Kind implements [Feature].
func (f *FilletFeature) Kind() string { return "fillet" }

// Recompute rounds the picked convex edges on the running body with a real rolling-ball
// blend (cylinder faces; planar ruling strips for variable sets). See fillet.go.
func (f *FilletFeature) Recompute(in Input) (Output, error) {
	prof := blendProfile{cross: f.def.CrossSection, rho: f.def.Rho}
	if len(f.def.EdgeSets) > 0 {
		return filletBodySets(in, f.def.EdgeSets, f.def.CornerType, f.def.ConcaveStrategy, prof, "fillet")
	}
	keys, err := bindGeomEdges(in, f.def.EdgeKeys, f.def.GeomEdges, "fillet")
	if err != nil {
		return Output{}, err
	}
	return filletBody(in, keys, callOrZero(f.def.Radius), f.def.CornerType, f.def.ConcaveStrategy, prof, "fillet", f.def.EdgeAnchors)
}
