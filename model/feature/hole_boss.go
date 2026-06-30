// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Hole and boss features place parametric cylindrical cuts/bosses on a picked placement
// face (held by reference key, re-resolved each recompute). A hole drills at an explicit
// Center (externally-authored holes) or the face centroid, along the inward normal with an
// EXACT cylinder wall (K1b): a Through All hole on a
// planar slab via brep.CutCylindricalHole, a blind hole via brep.CutBlindCylindricalHole
// (flat bottom, or a conical drill point when PointAngle is set). A counterbore adds a flat
// recess + shoulder; a countersink a true cone recess (exact-only). Unsupported drilled/
// counterbore shapes fall back to the faceted boolean. A boss is the join-side mirror of the
// drilled hole: the same tool cylinder grown OUT of the face and unioned (#327). A lost
// placement face → Sick.

// HoleTapInfo carries thread data for a tapped hole, consumed by hole tables (M14).
type HoleTapInfo struct {
	Tapped      bool
	Designation string
}

// HoleType is the hole's bottom/profile style.
type HoleType uint8

const (
	DrilledHole HoleType = iota
	CounterboreHole
	CountersinkHole
)

// HoleDefinition is the recipe for a hole: a placement face, diameter and depth,
// type, and optional tap data.
type HoleDefinition struct {
	PlacementFaceKey []byte
	GeomFace         *topo.GeometricFaceRef // externally-authored placement face by geometric descriptor (ADR-0040)
	// Center is an explicit drill point in model space (e.g. an exporter reading the host's
	// real hole position). Nil means drill at the placement face centroid (interactive default).
	Center          *math.Point3
	Diameter        func() float64
	Depth           func() float64
	ThroughAll      bool           // drill all the way through (depth ignored)
	CounterDiameter func() float64 // recess/sink diameter (Counterbore + Countersink)
	CounterDepth    func() float64 // counterbore recess depth (CounterboreHole)
	CounterAngle    func() float64 // countersink included angle, radians (CountersinkHole)
	PointAngle      func() float64 // drilled blind-hole point: included angle, radians (0 = flat)
	Type            HoleType
	Tap             HoleTapInfo
}

// HoleFeature drills a hole into the running solid.
type HoleFeature struct {
	def      *HoleDefinition
	featName string
	tool     *topo.Body // the drill cylinder of the last recompute, for pattern replication
}

// Definition returns the hole recipe.
func (h *HoleFeature) Definition() *HoleDefinition { return h.def }

// Kind implements [Feature].
func (h *HoleFeature) Kind() string { return "hole" }

// Operation reports that a hole removes material, so a pattern of a hole re-cuts the bore at
// each occurrence (one body with N holes) instead of copying the whole solid. Implements
// [OperationalFeature].
func (h *HoleFeature) Operation() ops.PartFeatureOperation { return ops.Cut }

// ToolBody returns the drill cylinder the last recompute cut, so a pattern replicates a clean
// bore at each occurrence rather than diffing the running solid (which degenerates once the
// bore makes a face curved). For a counterbore/countersink the tool is the main bore cylinder —
// the dominant cut — so a pattern of one reproduces its bores. Implements [ToolFeature].
func (h *HoleFeature) ToolBody() *topo.Body { return h.tool }

// Recompute resolves the placement face, then drills at the explicit center (or the face
// centroid) along the inward normal (see drill), subtracting the hole from the running body.
func (h *HoleFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, ok := resolveHoleFace(body, h.def)
	if !ok {
		return Output{}, fmt.Errorf("hole: placement face reference lost")
	}
	r, depth := callOrZero(h.def.Diameter)/2, callOrZero(h.def.Depth)
	if r <= 0 {
		return Output{}, fmt.Errorf("hole: diameter %g must be > 0", 2*r)
	}
	if !h.def.ThroughAll && depth <= 0 {
		return Output{}, fmt.Errorf("hole: depth %g must be > 0 (or set ThroughAll)", depth)
	}
	into, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0).Scale(-1))
	if err != nil {
		return Output{}, fmt.Errorf("hole: placement face has no normal")
	}
	center := holeCenter(h.def, face)
	h.tool = h.buildTool(body, center, into, r, depth)
	res, err := h.drill(body, center, into, r, depth)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res)}, nil
}

// holeCenter is the drill start point: an explicit center (projected onto the placement face's
// plane so the bore starts exactly at the surface), or the face centroid when none is set.
func holeCenter(def *HoleDefinition, face *topo.Face) math.Point3 {
	if def.Center == nil {
		return centroidOf(faceVertexPoints(face))
	}
	return projectOntoFacePlane(*def.Center, face)
}

// projectOntoFacePlane drops a model-space point onto a planar face's plane along its normal.
func projectOntoFacePlane(p math.Point3, face *topo.Face) math.Point3 {
	n, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0))
	if err != nil {
		return p
	}
	nv := n.AsVector()
	root := centroidOf(faceVertexPoints(face))
	d := root.VectorTo(p).Dot(nv)
	return p.TranslateBy(nv.Scale(-d))
}

// buildTool returns the clean drill cylinder a pattern replicates: the bore radius along the
// inward normal, spanning the body for a through hole or the requested depth for a blind one.
// It mirrors the cut the brep cutters apply (a counterbore's recess is omitted — the bore is
// the dominant geometry), so a pattern of the hole reproduces a sensible bore at each spot.
func (h *HoleFeature) buildTool(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) *topo.Body {
	if h.def.ThroughAll {
		depth = throughDepth(body, center, into)
	}
	return drillTool(center, into, r, depth, featOr(h.featName, "hole"))
}

// drill cuts the hole by its type: a counterbore is a shallow recess plus the bore, anything
// else a single cylinder.
func (h *HoleFeature) drill(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) (*topo.Body, error) {
	switch h.def.Type {
	case CounterboreHole:
		return h.cutCounterbore(body, center, into, r, depth)
	case CountersinkHole:
		return h.cutCountersink(body, center, into, r, depth)
	default:
		return h.cutDrilled(body, center, into, r, depth)
	}
}

// cutDrilled cuts a plain drilled hole: through, or blind with either a conical drill point
// (PointAngle > 0) or a flat bottom. A conical point that the part can't fit falls back to the
// flat/faceted blind cut.
func (h *HoleFeature) cutDrilled(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) (*topo.Body, error) {
	if h.def.ThroughAll {
		return h.cutCylinder(body, center, into, r, depth, true)
	}
	if angle := callOrZero(h.def.PointAngle); angle > 0 {
		if res, err := brep.CutBlindConicalHole(body, center, into.AsVector(), r, depth, angle/2); err == nil {
			return res, nil
		}
	}
	return h.cutCylinder(body, center, into, r, depth, false)
}

// cutCountersink drills a conical countersink (recess widening to the sink diameter at the
// surface) above the bore. The exact builder produces a true cone wall; an unsupported part
// shape returns the error (a faceted cone approximation would be worse than a clear failure).
func (h *HoleFeature) cutCountersink(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) (*topo.Body, error) {
	cr, angle := callOrZero(h.def.CounterDiameter)/2, callOrZero(h.def.CounterAngle)
	if cr <= r {
		return nil, fmt.Errorf("countersink: sink Ø %g must exceed bore Ø %g", 2*cr, 2*r)
	}
	if angle <= 0 || angle >= stdmath.Pi {
		return nil, fmt.Errorf("countersink: included angle %g must be in (0, π)", angle)
	}
	half := angle / 2
	depthCS := (cr - r) / stdmath.Tan(half)
	if !h.def.ThroughAll && depth <= depthCS {
		return nil, fmt.Errorf("countersink: total depth %g must exceed the sink depth %g", depth, depthCS)
	}
	return brep.CutCountersinkHole(body, center, into.AsVector(), r, depth-depthCS, cr, half, h.def.ThroughAll)
}

// cutCounterbore drills a counterbore: a shallow recess stepping down to the bore. The exact
// path (brep.CutCounterboreHole) builds the stepped result in one shot — two cylinder walls and
// an annular shoulder — from the planar slab; it does NOT chain two curved cuts (the second
// would feed a curved body to the planar-only boolean). The faceted fallback cuts the recess
// then the bore as sequential planar prisms (each stays planar, so it chains fine).
func (h *HoleFeature) cutCounterbore(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) (*topo.Body, error) {
	cr, cd := callOrZero(h.def.CounterDiameter)/2, callOrZero(h.def.CounterDepth)
	if cr <= r {
		return nil, fmt.Errorf("counterbore: recess Ø %g must exceed bore Ø %g", 2*cr, 2*r)
	}
	if cd <= 0 || (!h.def.ThroughAll && depth <= cd) {
		return nil, fmt.Errorf("counterbore: recess depth %g must be > 0 and less than total depth %g", cd, depth)
	}
	if res, err := brep.CutCounterboreHole(body, center, into.AsVector(), r, depth-cd, cr, cd, h.def.ThroughAll); err == nil {
		return res, nil
	}
	return h.facetedCounterbore(body, center, into, r, depth, cr, cd)
}

// facetedCounterbore is the fallback for shapes the exact builder rejects: cut the recess prism,
// then the bore prism from the shoulder (both planar cuts, so they chain through the boolean).
func (h *HoleFeature) facetedCounterbore(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth, cr, cd float64) (*topo.Body, error) {
	stepped, err := ops.Boolean(ops.Cut, body, drillTool(center, into, cr, cd, featOr(h.featName, "hole")))
	if err != nil {
		return nil, err
	}
	shoulder := center.TranslateBy(into.AsVector().Scale(math.Scalar(cd)))
	boreLen := depth - cd
	if h.def.ThroughAll {
		boreLen = throughDepth(stepped, shoulder, into)
	}
	return ops.Boolean(ops.Cut, stepped, drillTool(shoulder, into, r, boreLen, featOr(h.featName, "hole")))
}

// cutCylinder cuts a single cylindrical hole, preferring an EXACT cylinder wall (K1b): a
// through hole via brep.CutCylindricalHole, a blind hole via brep.CutBlindCylindricalHole
// (wall + flat bottom). When the part shape isn't supported (the bore clips a face, or a blind
// depth would exit), it falls back to the faceted boolean — a through-cut when `through`, the
// requested depth otherwise.
func (h *HoleFeature) cutCylinder(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64, through bool) (*topo.Body, error) {
	if through {
		if res, err := brep.CutCylindricalHole(body, center, into.AsVector(), r); err == nil {
			return res, nil
		}
		depth = throughDepth(body, center, into) // unsupported shape → faceted through-cut
	} else if res, err := brep.CutBlindCylindricalHole(body, center, into.AsVector(), r, depth); err == nil {
		return res, nil
	}
	tool := drillTool(center, into, r, depth, featOr(h.featName, "hole"))
	return ops.Boolean(ops.Cut, body, tool)
}

// BossDefinition is the recipe for a boss: a raised cylinder on a placement face.
type BossDefinition struct {
	PlacementFaceKey []byte
	Diameter         func() float64
	Height           func() float64
	// FaceAnchors maps the placement face key to its mint-time centroid for the geometric
	// recovery tier (ADR-0043 P6 / #1579); see FilletDefinition.EdgeAnchors.
	FaceAnchors map[string]math.Point3
}

// BossFeature adds a cylindrical boss to the running solid.
type BossFeature struct {
	def      *BossDefinition
	featName string
	tool     *topo.Body // the boss cylinder of the last recompute, for pattern replication
}

func (b *BossFeature) Definition() *BossDefinition { return b.def }
func (b *BossFeature) Kind() string                { return "boss" }

// Recompute resolves the placement face and raises the boss cylinder from its centroid along
// the outward normal, joining it to the running body. The tool's small entry overhang sits
// INSIDE the body (drillTool's near span), so the union always overlaps cleanly.
func (b *BossFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, mt, err := bindFace(body, b.def.PlacementFaceKey, anchorFor(b.def.PlacementFaceKey, b.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("boss: %w", err)
	}
	r, h := callOrZero(b.def.Diameter)/2, callOrZero(b.def.Height)
	if r <= 0 || h <= 0 {
		return Output{}, fmt.Errorf("boss: diameter %g and height %g must be > 0", 2*r, h)
	}
	out, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0))
	if err != nil {
		return Output{}, fmt.Errorf("boss: placement face has no normal")
	}
	b.tool = drillTool(centroidOf(faceVertexPoints(face)), out, r, h, featOr(b.featName, "boss"))
	res, err := ops.Boolean(ops.Join, body, b.tool)
	if err != nil {
		return Output{}, fmt.Errorf("boss: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: faceHeal(b.def.PlacementFaceKey, mt)}, nil
}

// Operation reports that a boss adds material, so a pattern of a boss unions its raised
// cylinder at each occurrence (one body with N studs) instead of copying the whole solid.
// Implements [OperationalFeature].
func (b *BossFeature) Operation() ops.PartFeatureOperation { return ops.Join }

// ToolBody returns the boss cylinder the last recompute joined, so a pattern replicates a
// clean stud at each occurrence. Implements [ToolFeature].
func (b *BossFeature) ToolBody() *topo.Body { return b.tool }

// HoleFeatures and BossFeatures add hole/boss features into the engine.
type (
	HoleFeatures struct{ engine *PartFeatures }
	BossFeatures struct{ engine *PartFeatures }
)

// NewHoleFeatures / NewBossFeatures bind the collections to an engine.
func NewHoleFeatures(engine *PartFeatures) *HoleFeatures { return &HoleFeatures{engine} }
func NewBossFeatures(engine *PartFeatures) *BossFeatures { return &BossFeatures{engine} }

// AddDrilled adds a simple drilled hole on the placement face.
func (c *HoleFeatures) AddDrilled(faceKey []byte, diameter, depth func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole})
}

// AddDrilledThrough adds a drilled hole that goes all the way through the part (an exact
// cylinder wall on a planar-faced slab); depth is unused.
func (c *HoleFeatures) AddDrilledThrough(faceKey []byte, diameter func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: constFloat(0), ThroughAll: true, Type: DrilledHole})
}

// AddCounterbore adds a counterbore hole: a shallow recess (counterDiameter × counterDepth)
// at the placement face above the main bore (diameter × depth). Set ThroughAll on the returned
// definition for a through bore.
func (c *HoleFeatures) AddCounterbore(faceKey []byte, diameter, depth, counterDiameter, counterDepth func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{
		PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth,
		CounterDiameter: counterDiameter, CounterDepth: counterDepth, Type: CounterboreHole,
	})
}

// AddCountersink adds a countersink hole: a conical recess (sinkDiameter at the surface,
// included angle) above the main bore (diameter × depth). Set ThroughAll for a through bore.
func (c *HoleFeatures) AddCountersink(faceKey []byte, diameter, depth, sinkDiameter, includedAngle func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{
		PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth,
		CounterDiameter: sinkDiameter, CounterAngle: includedAngle, Type: CountersinkHole,
	})
}

// AddTapped adds a tapped hole with thread data.
func (c *HoleFeatures) AddTapped(faceKey []byte, diameter, depth func() float64, designation string) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole, Tap: HoleTapInfo{Tapped: true, Designation: designation}})
}

// addHole adds a hole feature, naming it (Hole1, Hole2, …) so its generated topology has
// a stable, distinct lineage.
// resolveHoleFace finds the hole's placement face: by lineage key when present, else by a
// geometric descriptor (the externally-authored path, ADR-0040), else lost.
func resolveHoleFace(body *topo.Body, def *HoleDefinition) (*topo.Face, bool) {
	if len(def.PlacementFaceKey) > 0 {
		return body.FindFaceByKey(def.PlacementFaceKey)
	}
	if def.GeomFace != nil {
		return body.FindFaceByGeometry(*def.GeomFace, geomEdgeBindTol)
	}
	return nil, false
}

func (c *HoleFeatures) addHole(def *HoleDefinition) *PartFeature {
	hf := &HoleFeature{def: def}
	pf := c.engine.Add(hf)
	pf.SetName(c.engine.UniqueName("Hole"))
	hf.featName = pf.name
	return pf
}

// Add adds a cylindrical boss on the placement face, naming it (Boss1, Boss2, …) so its
// generated topology has a stable, distinct lineage.
func (c *BossFeatures) Add(faceKey []byte, diameter, height func() float64) *PartFeature {
	bf := &BossFeature{def: &BossDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Height: height}}
	pf := c.engine.Add(bf)
	pf.SetName(c.engine.UniqueName("Boss"))
	bf.featName = pf.name
	return pf
}
