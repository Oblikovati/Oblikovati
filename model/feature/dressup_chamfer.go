// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Dress-up features — the CHAMFER definition (M48 #2233 split of dressup.go). The edge chamfer feature
// (distance/distance-and-angle/two-distance modes, concave strategy) and its Recompute. The chamfer
// cut geometry itself lives in chamfer.go; the adder collection in dressup.go.

// ChamferType aliases the public chamfer-mode discriminator (ADR-0018).
type ChamferType = types.ChamferType

// ChamferConcaveStrategy aliases the public concave-edge strategy discriminator (ADR-0018).
type ChamferConcaveStrategy = types.ChamferConcaveStrategy

// ChamferDefinition bevels selected edges. Type (M20-F03) selects the setback mode: equal
// Distance on both faces (the default / zero value), two independent distances (Distance +
// Distance2, asymmetric), or a Distance plus the chamfer-face Angle. FlatCorners blends a
// vertex where three selected edges meet into a flat triangular face — only for the
// equal-distance mode; an asymmetric chamfer leaves the corner planes to meet at a point.
// ConcaveStrategy applies only to CONCAVE (internal) edges: fill the inside corner with material
// (outward, the zero-value default) or cut a recessed relief groove (inward).
type ChamferDefinition struct {
	EdgeKeys        [][]byte
	Distance        func() float64
	Distance2       func() float64 // twoDistances: setback on the second face
	Angle           func() float64 // distanceAndAngle: chamfer-face angle (radians)
	Type            ChamferType    // zero value ⇒ equal-distance
	FlatCorners     bool
	ConcaveStrategy ChamferConcaveStrategy  // zero value ⇒ outward (fill the inside corner)
	GeomEdges       []topo.GeometricEdgeRef // externally-authored edges by geometric descriptor (see FilletDefinition.GeomEdges)
	// EdgeAnchors maps an EdgeKeys entry to its mint-time midpoint for the geometric recovery
	// tier (ADR-0043 P6b); see FilletDefinition.EdgeAnchors.
	EdgeAnchors map[string]math.Point3
	// ReferenceFace is the face Distance is measured on for the asymmetric modes (#1888). Empty
	// leaves the assignment to the edge's own face order, which is a topology artefact — on
	// mirrored geometry that can put the larger setback on the wrong face. See orderedSetbacks.
	ReferenceFace []byte
	// PartialStart and PartialLength bevel only a SPAN of each edge, measured from its start vertex
	// (#1888). PartialLength 0 ⇒ the whole edge. See wedgeSpan.
	PartialStart  func() float64
	PartialLength func() float64
}

// ChamferFeature bevels selected edges (equal-distance, two-distance, or distance-and-angle).
type ChamferFeature struct {
	def      *ChamferDefinition
	featName string
}

func (c *ChamferFeature) Definition() *ChamferDefinition { return c.def }
func (c *ChamferFeature) Kind() string                   { return "chamfer" }

// Recompute bevels each selected (convex) edge by cutting a wedge tool along it via the
// boolean; the two setbacks come from the mode (see chamferSetbacks). Flat-corner blending
// applies to every mode — the blend is built from the per-face setbacks, so an asymmetric
// (two-distance / distance-angle) corner blends just like a symmetric one. See chamfer.go.
func (c *ChamferFeature) Recompute(in Input) (Output, error) {
	d1, d2, err := chamferSetbacks(c.def)
	if err != nil {
		return Output{}, err
	}
	keys, err := bindGeomEdges(in, c.def.EdgeKeys, c.def.GeomEdges, "chamfer")
	if err != nil {
		return Output{}, err
	}
	return chamferEdges(in, keys, d1, d2, featOr(c.featName, "chamfer"), c.def.FlatCorners,
		c.def.ConcaveStrategy, c.def.runOf(), c.def.EdgeAnchors)
}
