// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Rib feature (M48 #2239 split of sketched_features.go). Thickens an open sketch path into a rib wall
// between existing faces: the definition, the feature wrapper and Recompute (normal + to-next depth),
// the wall path derivation and the RibFeatures adder collection.

// RibDefinition is the recipe for a rib: a thin wall from an open profile (a sketch path).
type RibDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int            // index into the sketch's open paths
	Thickness    func() float64 // wall thickness, centered on the path
	Depth        func() float64 // signed extent along the sketch-plane normal
	// ToNext extends the wall until it fully lands on the existing material
	// (the reference "to-next" rib, M08 PBI-096 #316); Depth then only picks
	// the direction (its sign; nil/0 ⇒ +normal).
	ToNext    bool
	Operation ops.PartFeatureOperation
	// The wall's cross-section options (#1882; see rib_wall.go). ThickenSide picks which side of
	// the path the thickness lands on; Draft (radians) tapers the wall, opening it toward the root;
	// HoldThicknessAtRoot measures Thickness at the root instead of at the sketch plane (Inventor's
	// kRibThicknessAtRoot), which is observable only under Draft; ExtendProfile lengthens the
	// path's ends onto the part. Every zero value is the behaviour that predates the options.
	ThickenSide         RibThickenSide
	Draft               float64
	HoldThicknessAtRoot bool
	ExtendProfile       bool
	// Direction is whether the profile is projected NORMAL to the sketch plane (a web — the default
	// and zero value, so a definition written before the option keeps its behaviour) or PARALLEL to
	// it (a rib), mirroring Inventor's RibDefinition.IsRib (#2064). A parallel rib thickens the
	// profile ALONG the plane normal and grows the wall IN the plane until it lands on the part — the
	// 90°-rotated form used for a moulded part sectioned through its wall.
	Direction RibDirection
	// DraftProfileEnds says whether a NORMAL rib's draft tapers the profile's END caps along with its
	// sides (Inventor's RibDefinition.DraftProfileEnds, default True). nil ⇒ the default (drafted
	// ends, the pre-option behaviour); a non-nil false leaves the ends square. It has no effect
	// without a draft, and none on a parallel rib (which refuses a draft, as Inventor does).
	DraftProfileEnds *bool
}

// RibDirection selects how a rib projects its sketch profile — Inventor's RibDefinition.IsRib.
type RibDirection int

const (
	// RibNormalToSketch thickens the profile in the sketch plane and extrudes it ALONG the plane
	// normal (a web). The default and zero value.
	RibNormalToSketch RibDirection = iota
	// RibParallelToSketch thickens the profile along the plane normal and grows the wall IN the
	// plane until it lands on the part (a rib), rotated 90° from the web form.
	RibParallelToSketch
)

// RibFeature thickens an open sketch profile (a path) into a wall: the path is offset in the
// sketch plane to a closed band (by ±Thickness/2, or wholly to one side — see [RibThickenSide]),
// then extruded Depth along the plane normal, optionally drafted, and combined with the running
// body. rib_wall.go shapes the band; ribDepth resolves the extent.
type RibFeature struct {
	def  *RibDefinition
	tool *topo.Body // last rib wall, exposed so a pattern can replicate it
}

func (r *RibFeature) Definition() *RibDefinition { return r.def }
func (r *RibFeature) Kind() string               { return "rib" }

// Operation and ToolBody let a pattern/mirror replicate this feature (see [ToolFeature]).
func (r *RibFeature) Operation() ops.PartFeatureOperation { return r.def.Operation }
func (r *RibFeature) ToolBody() *topo.Body                { return r.tool }

func (r *RibFeature) Recompute(in Input) (Output, error) {
	pts, err := r.wallPath(in)
	if err != nil {
		return Output{}, err
	}
	if r.def.Thickness == nil {
		return Output{}, errors.New("rib: thickness must be set")
	}
	t := r.def.Thickness()
	if t <= 0 {
		return Output{}, fmt.Errorf("rib: need positive thickness, got t=%g", t)
	}
	if r.def.Direction == RibParallelToSketch {
		return r.recomputeParallel(in, pts, t)
	}
	return r.recomputeNormal(in, pts, t)
}

// recomputeNormal is the web form: thicken the path IN the sketch plane and extrude ALONG the plane
// normal by the depth. The draft tapers the wall toward the root; DraftProfileEnds decides whether it
// tapers the end caps too.
func (r *RibFeature) recomputeNormal(in Input, pts []math.Point2, t float64) (Output, error) {
	// Square (undrafted) profile ends under a draft need a DIRECTIONAL taper — widen the sides but
	// hold the end caps perpendicular — which buildExtrusionShell's uniform outward taper cannot
	// express. It is refused rather than silently drafting the ends (which would be the wrong shape).
	// Without a draft the flag has no effect and the wall is square-ended either way. (#2064)
	if r.def.Draft != 0 && !r.draftsProfileEnds() {
		return Output{}, errors.New("rib: square profile ends under a draft are not modelled yet " +
			"(the end caps would be drafted with the sides); drop the draft or leave draftProfileEnds set")
	}
	d, err := r.ribDepth(in, pts)
	if err != nil {
		return Output{}, err
	}
	band, taper, err := ribWallBand(pts, t, d, r.def.Draft, r.def.ThickenSide, r.def.HoldThicknessAtRoot)
	if err != nil {
		return Output{}, err
	}
	// The Surface operation (kSurfaceOperation, #1858) builds the rib walls only — an open sheet, no
	// caps — rather than the capped prism; combine() adds it as a surface body (no boolean).
	r.tool = buildExtrusionShell(band, r.def.Sketch.Plane(), orderedSpan(0, d), taper, "rib", r.def.Operation != ops.Surface)
	bodies, err := combine(in, r.tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// draftsProfileEnds resolves DraftProfileEnds: nil ⇒ the default True (Inventor's default and the
// pre-option behaviour), so only a non-nil false leaves the ends square.
func (r *RibFeature) draftsProfileEnds() bool {
	return r.def.DraftProfileEnds == nil || *r.def.DraftProfileEnds
}

// ribDepth resolves the wall extent: the explicit signed depth, or with
// ToNext the distance to the FARTHEST first-hit of the existing material
// among the path's points — the wall must fully land, so the deepest ray
// governs (extrude's to-next takes the nearest; a rib that stopped there
// would leave part of the wall hanging).
func (r *RibFeature) ribDepth(in Input, pts []math.Point2) (float64, error) {
	if !r.def.ToNext {
		d := callOrZero(r.def.Depth)
		if d == 0 {
			return 0, errors.New("rib: need a non-zero depth (or set toNext)")
		}
		return d, nil
	}
	sign := 1.0
	if callOrZero(r.def.Depth) < 0 {
		sign = -1
	}
	return ribToNextDepth(in.Bodies, r.def.Sketch.Plane(), pts, sign)
}

// ribToNextDepth ray-casts each path point along the (signed) plane normal
// into the existing material and returns the farthest first-hit as the signed
// depth; a point with no material ahead is a precise error.
func ribToNextDepth(bodies []*topo.Body, plane sketch.Plane, pts []math.Point2, sign float64) (float64, error) {
	if len(bodies) == 0 {
		return 0, errors.New("rib: to-next needs existing material")
	}
	dir := plane.Normal().AsVector().Scale(math.Scalar(sign))
	deepest := 0.0
	for i, p := range pts {
		origin := plane.ToModel(p)
		hit, ok := nearestBodyHit(bodies, origin, dir)
		if !ok {
			return 0, fmt.Errorf("rib: to-next found no material ahead of path point %d (%v)", i, p)
		}
		if hit > deepest {
			deepest = hit
		}
	}
	return sign * deepest, nil
}

// nearestBodyHit is the closest positive ray hit over all bodies.
func nearestBodyHit(bodies []*topo.Body, origin math.Point3, dir math.Vector3) (float64, bool) {
	best, found := stdmath.Inf(1), false
	for _, b := range bodies {
		if _, t, ok := query.RayCastFaces(b, origin, dir, ops.DefaultQuality()); ok && t > math.DefaultTolerance && t < best {
			best, found = t, true
		}
	}
	return best, found
}

// wallPath is the path the wall is built on: the sketch's open profile, with its ends extended
// onto the existing material when ExtendProfile asks for it (#1882). The extension happens BEFORE
// ribDepth measures a to-next extent, so a lengthened end's ray counts toward the depth the wall
// needs in order to land everywhere.
func (r *RibFeature) wallPath(in Input) ([]math.Point2, error) {
	pts, err := r.pathPoints()
	if err != nil || !r.def.ExtendProfile {
		return pts, err
	}
	return ribExtendedPath(pts, r.def.Sketch.Plane(), in.Bodies), nil
}

// pathPoints resolves the rib's open profile (a sketch path) to its ordered points.
func (r *RibFeature) pathPoints() ([]math.Point2, error) {
	paths := r.def.Sketch.Paths()
	if r.def.ProfileIndex < 0 || r.def.ProfileIndex >= len(paths) {
		return nil, fmt.Errorf("rib: path index %d out of range (%d open paths)", r.def.ProfileIndex, len(paths))
	}
	pts := paths[r.def.ProfileIndex].Points()
	if len(pts) < 2 {
		return nil, errors.New("rib: the open profile needs at least two points")
	}
	return pts, nil
}

// RibFeatures adds ribs into the engine.
type RibFeatures struct{ engine *PartFeatures }

// NewRibFeatures binds the collection to an engine.
func NewRibFeatures(engine *PartFeatures) *RibFeatures { return &RibFeatures{engine} }

// AddDefinition adds a rib from a fully-populated definition (to-next, #316).
func (c *RibFeatures) AddDefinition(def *RibDefinition) *PartFeature {
	pf := c.engine.Add(&RibFeature{def: def})
	pf.SetName(c.engine.UniqueName("Rib"))
	return pf
}

// Add adds a rib that thickens the sketch's open profile (by index) into a wall of the given
// thickness, extruded the signed depth along the sketch-plane normal, joined to the part.
func (c *RibFeatures) Add(skt *sketch.Sketch, profileIndex int, thickness, depth func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RibDefinition{Sketch: skt, ProfileIndex: profileIndex, Thickness: thickness, Depth: depth, Operation: op}
	pf := c.engine.Add(&RibFeature{def: def})
	pf.SetName(c.engine.UniqueName("Rib"))
	return pf
}
