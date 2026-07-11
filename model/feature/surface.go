// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Surface-creation features (M10-F01) turn curves into surface bodies (open
// quilts), distinct from the solid sketched features (M08). Phase A builds the real
// planar/ruled geometry; the genuinely-curved cases (a non-planar boundary blend, a
// tangent/perpendicular ruling that needs adjacent-face data) are NURBS phase B,
// deferred behind the same Definition surface.

// PatchCondition is the continuity a boundary patch maintains with the surfaces
// adjacent to a boundary loop: G0 position only (Free), G1 tangency (Tangent), or
// G2 curvature (Curvature). Modernizes Inventor's BoundaryConditionEnum.
type PatchCondition int

const (
	// PatchFree honors position only (G0).
	PatchFree PatchCondition = iota
	// PatchTangent is tangent-continuous to the adjacent faces (G1).
	PatchTangent
	// PatchCurvature is curvature-continuous to the adjacent faces (G2).
	PatchCurvature
)

// BoundaryPatchLoop is one boundary of a patch: a closed sketch profile plus the
// continuity condition the patch honors along it.
type BoundaryPatchLoop struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Condition    PatchCondition
}

// BoundaryPatchLoops is the ordered set of boundary loops of a patch.
type BoundaryPatchLoops struct {
	items []*BoundaryPatchLoop
}

// Add appends a boundary loop (a closed profile + its continuity condition).
func (ls *BoundaryPatchLoops) Add(skt *sketch.Sketch, profileIndex int, cond PatchCondition) *BoundaryPatchLoop {
	l := &BoundaryPatchLoop{Sketch: skt, ProfileIndex: profileIndex, Condition: cond}
	ls.items = append(ls.items, l)
	return l
}

// Count returns the number of loops; Item returns the i-th.
func (ls *BoundaryPatchLoops) Count() int                    { return len(ls.items) }
func (ls *BoundaryPatchLoops) Item(i int) *BoundaryPatchLoop { return ls.items[i] }

// BoundaryPatchDefinition is the recipe for a boundary patch. Loops is the planar-sketch form (the
// first closed loop is filled, the others cut as holes). EdgeKeys is the non-planar 3D edge-loop form
// (#1867): reference keys of surface-body boundary edges filled with a NURBS that blends to their
// adjacent faces at Condition continuity (free/tangent/curvature → G0/G1/G2). GuideRailKeys and
// TangentWeight are recorded for the interior-interpolation phase (accepted, not yet honoured).
type BoundaryPatchDefinition struct {
	Loops         *BoundaryPatchLoops
	EdgeKeys      [][]byte
	Condition     PatchCondition
	GuideRailKeys [][]byte
	TangentWeight float64
}

// BoundaryPatchFeature fills a closed boundary with a trimmed surface patch.
type BoundaryPatchFeature struct {
	def      *BoundaryPatchDefinition
	featName string
}

// Definition returns the patch recipe.
func (b *BoundaryPatchFeature) Definition() *BoundaryPatchDefinition { return b.def }

// Kind implements [Feature].
func (b *BoundaryPatchFeature) Kind() string { return "boundary-patch" }

// Recompute resolves the boundary loop and fills it with a planar surface patch.
// The per-loop continuity conditions are carried on the definition; an isolated
// planar loop satisfies any condition vacuously (there is no adjacent face to blend
// to). Patching a non-planar 3D boundary is the NURBS phase-B case (deferred).
func (b *BoundaryPatchFeature) Recompute(in Input) (Output, error) {
	if len(b.def.EdgeKeys) > 0 {
		return b.fillEdgeLoop(in)
	}
	if b.def.Loops == nil || b.def.Loops.Count() == 0 {
		return Output{}, errors.New("boundary patch: no boundary loops")
	}
	loop := b.def.Loops.Item(0)
	profile, err := resolveClosedProfile(loop.Sketch, loop.ProfileIndex, "boundary patch")
	if err != nil {
		return Output{}, err
	}
	patch := buildPlanarPatch(profile, loop.Sketch.Plane(), b.featName)
	return Output{Bodies: appendBody(in.Bodies, patch)}, nil
}

// fillEdgeLoop fills a non-planar 3D edge loop taken from surface-body boundary edges with a NURBS
// patch that blends to the adjacent faces at the definition's continuity (#1867).
func (b *BoundaryPatchFeature) fillEdgeLoop(in Input) (Output, error) {
	edges, err := resolveLoopEdges(in.Bodies, b.def.EdgeKeys)
	if err != nil {
		return Output{}, err
	}
	patch, err := ops.FillEdgeLoop(edges, patchContinuityOrder(b.def.Condition))
	if err != nil {
		return Output{}, fmt.Errorf("boundary patch: %w", err)
	}
	return Output{Bodies: appendBody(in.Bodies, patch)}, nil
}

// resolveLoopEdges resolves each boundary-edge reference key against the running bodies.
func resolveLoopEdges(bodies []*topo.Body, keys [][]byte) ([]*topo.Edge, error) {
	edges := make([]*topo.Edge, 0, len(keys))
	for _, k := range keys {
		e, ok := findEdgeByKey(bodies, k)
		if !ok {
			return nil, fmt.Errorf("boundary patch: edge %x not found in the model", k)
		}
		edges = append(edges, e)
	}
	return edges, nil
}

// findEdgeByKey looks a reference key up across all running bodies.
func findEdgeByKey(bodies []*topo.Body, key []byte) (*topo.Edge, bool) {
	for _, bd := range bodies {
		if e, ok := bd.FindEdgeByKey(key); ok {
			return e, true
		}
	}
	return nil, false
}

// patchContinuityOrder maps a patch condition to the fill's NURBS continuity order (free→G0,
// tangent→G1, curvature→G2).
func patchContinuityOrder(c PatchCondition) int {
	switch c {
	case PatchTangent:
		return 1
	case PatchCurvature:
		return 2
	default:
		return 0
	}
}

// BoundaryPatchFeatures adds boundary patches into the engine.
type BoundaryPatchFeatures struct{ engine *PartFeatures }

// NewBoundaryPatchFeatures binds the collection to a feature engine.
func NewBoundaryPatchFeatures(engine *PartFeatures) *BoundaryPatchFeatures {
	return &BoundaryPatchFeatures{engine: engine}
}

// Add fills the closed profileIndex profile of skt with a patch honoring cond.
func (c *BoundaryPatchFeatures) Add(skt *sketch.Sketch, profileIndex int, cond PatchCondition) *PartFeature {
	loops := &BoundaryPatchLoops{}
	loops.Add(skt, profileIndex, cond)
	return c.add(&BoundaryPatchDefinition{Loops: loops})
}

// AddEdgeLoop fills a non-planar loop of surface-body boundary edges (reference keys) with a NURBS
// patch at the given continuity, blending to the loop's adjacent faces (#1867). guideRails and
// tangentWeight are recorded for a later interior-interpolation phase.
func (c *BoundaryPatchFeatures) AddEdgeLoop(edgeKeys [][]byte, cond PatchCondition, guideRails [][]byte, tangentWeight float64) *PartFeature {
	return c.add(&BoundaryPatchDefinition{EdgeKeys: edgeKeys, Condition: cond, GuideRailKeys: guideRails, TangentWeight: tangentWeight})
}

// add registers a boundary-patch feature from its definition and back-fills the feature name.
func (c *BoundaryPatchFeatures) add(def *BoundaryPatchDefinition) *PartFeature {
	bf := &BoundaryPatchFeature{def: def, featName: "BoundaryPatch"}
	pf := c.engine.Add(bf)
	bf.featName = pf.name
	return pf
}

// RuledSurfaceType is the direction convention for a ruled surface's straight rulings (Inventor's
// RuledSurfaceTypeEnum): along the source plane Normal, Tangent to the adjacent face, or Sweep along
// an explicit direction vector (kSweepRuledSurfaceType) (#1868).
type RuledSurfaceType int

const (
	// RuledNormal rules along the source profile-plane normal.
	RuledNormal RuledSurfaceType = iota
	// RuledTangent rules tangent to the adjacent face — needs edge-face pairs off a body (phase C).
	RuledTangent
	// RuledSweep rules along an explicit direction vector (Inventor kSweepRuledSurfaceType) (#1868).
	RuledSweep
)

// RuledSurfaceDefinition is the recipe for a ruled surface: a profile whose edges are swept by
// straight rulings over a distance. Direction is the ruling vector for RuledSweep (RuledNormal uses
// the profile-plane normal); DraftAngle tilts each ruling outward; Flip reverses the ruling side.
type RuledSurfaceDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Type         RuledSurfaceType
	Distance     func() float64
	Direction    math.Vector3
	DraftAngle   func() float64
	Flip         bool
}

// RuledSurfaceFeature sweeps a profile's edges by straight rulings into a band.
type RuledSurfaceFeature struct {
	def      *RuledSurfaceDefinition
	featName string
}

// Definition returns the ruled-surface recipe.
func (r *RuledSurfaceFeature) Definition() *RuledSurfaceDefinition { return r.def }

// Kind implements [Feature].
func (r *RuledSurfaceFeature) Kind() string { return "ruled-surface" }

// Recompute resolves the profile and builds the ruled band: Normal rules along the profile-plane
// normal, Sweep along the explicit Direction, both honouring the draft angle and flip. Tangent
// rulings need the adjacent face's tangent along a body edge (edge-face pairs) — phase C, deferred.
func (r *RuledSurfaceFeature) Recompute(in Input) (Output, error) {
	profile, err := resolveClosedProfile(r.def.Sketch, r.def.ProfileIndex, "ruled surface")
	if err != nil {
		return Output{}, err
	}
	if r.def.Type == RuledTangent {
		return Output{Bodies: in.Bodies}, ErrDeferred
	}
	dist := measure(r.def.Distance)
	if dist == 0 {
		return Output{}, errors.New("ruled surface: distance is zero")
	}
	plane := r.def.Sketch.Plane()
	dir := plane.Normal().AsVector()
	if r.def.Type == RuledSweep {
		if r.def.Direction.LengthSquared() < 1e-18 {
			return Output{}, errors.New("ruled surface: sweep needs a non-zero direction")
		}
		dir = r.def.Direction
	}
	if r.def.Flip {
		dir = dir.Scale(-1)
	}
	band := buildRuledBandDir(profile.OuterLoop().Polygon(), plane, dir, dist, measure(r.def.DraftAngle), r.featName)
	return Output{Bodies: appendBody(in.Bodies, band)}, nil
}

// RuledSurfaceFeatures adds ruled surfaces into the engine.
type RuledSurfaceFeatures struct{ engine *PartFeatures }

// NewRuledSurfaceFeatures binds the collection to a feature engine.
func NewRuledSurfaceFeatures(engine *PartFeatures) *RuledSurfaceFeatures {
	return &RuledSurfaceFeatures{engine: engine}
}

// AddByDistance rules the closed profileIndex profile of skt by distance, in the
// kind direction (Normal/Tangent — the plain rulings, no draft or explicit vector).
func (c *RuledSurfaceFeatures) AddByDistance(skt *sketch.Sketch, profileIndex int, kind RuledSurfaceType, distance func() float64) *PartFeature {
	return c.AddRuled(&RuledSurfaceDefinition{Sketch: skt, ProfileIndex: profileIndex, Type: kind, Distance: distance})
}

// AddRuled adds a ruled surface from a fully-specified definition (sweep direction, draft, flip) (#1868).
func (c *RuledSurfaceFeatures) AddRuled(def *RuledSurfaceDefinition) *PartFeature {
	rf := &RuledSurfaceFeature{def: def, featName: "RuledSurface"}
	pf := c.engine.Add(rf)
	rf.featName = pf.name
	return pf
}

// resolveClosedProfile re-derives the closed profile at index from a sketch, erroring
// (→ sick) when the sketch/profile is missing or the profile is open.
func resolveClosedProfile(skt *sketch.Sketch, index int, what string) (*sketch.Profile, error) {
	if skt == nil {
		return nil, fmt.Errorf("%s: no sketch", what)
	}
	profiles := skt.Profiles()
	if index < 0 || index >= profiles.Count() {
		return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", what, index, profiles.Count())
	}
	p := profiles.Item(index)
	if !p.IsClosed() {
		return nil, fmt.Errorf("%s: profile is open (cannot bound a surface)", what)
	}
	return p, nil
}

// measure evaluates a distance closure, treating a nil closure as zero.
func measure(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}

// appendBody returns a fresh slice with body appended to running (no aliasing).
func appendBody(running []*topo.Body, body *topo.Body) []*topo.Body {
	return append(append([]*topo.Body(nil), running...), body)
}

// buildPlanarPatch fills a closed planar profile with a single trimmed planar face
// (the outer loop bounds it, inner loops cut holes), producing an open surface body.
func buildPlanarPatch(profile *sketch.Profile, plane sketch.Plane, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	origin := plane.ToModel(profile.OuterLoop().Polygon()[0])
	surf, _ := geom.NewPlane(origin, plane.Normal().AsVector())
	specs := []topo.LoopSpec{loopSpec(bld, profile.OuterLoop(), plane, feat, "outer", 0, true)}
	for i, inner := range profile.InnerLoops() {
		specs = append(specs, loopSpec(bld, inner, plane, feat, "inner", i, false))
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "patch", 0)), specs...)
	return bld.Build()
}

// loopSpec builds the vertices and edges of one sketch loop and returns its loop
// specification (outer boundary or inner hole) for a face.
func loopSpec(bld *topo.Builder, loop sketch.Loop, plane sketch.Plane, feat, role string, idx int, outer bool) topo.LoopSpec {
	verts := loopVertices(bld, loop, plane, feat, role, idx)
	uses := loopEdges(bld, verts, feat, role, idx)
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// loopVertices maps a loop's polygon to model-space vertices with per-loop lineage.
func loopVertices(bld *topo.Builder, loop sketch.Loop, plane sketch.Plane, feat, role string, idx int) []*topo.Vertex {
	poly := loop.Polygon()
	verts := make([]*topo.Vertex, len(poly))
	r := fmt.Sprintf("%s%d-vertex", role, idx)
	for i, p := range poly {
		verts[i] = bld.AddVertex(plane.ToModel(p), topo.NewLineage(topo.Tok(feat, r, i)))
	}
	return verts
}

// loopEdges links consecutive vertices into a closed chain of forward edge uses.
func loopEdges(bld *topo.Builder, verts []*topo.Vertex, feat, role string, idx int) []topo.Use {
	n := len(verts)
	uses := make([]topo.Use, n)
	r := fmt.Sprintf("%s%d-edge", role, idx)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		e := bld.AddEdge(geom.NewLineSegment(verts[i].Point(), verts[j].Point()), verts[i], verts[j], topo.NewLineage(topo.Tok(feat, r, i)))
		uses[i] = topo.Fwd(e)
	}
	return uses
}

// buildRuledBandDir sweeps a closed polygon by dist along the ruling direction dir into a band of
// planar quad faces (no caps) — an open surface body whose straight rulings connect the bottom and
// top loops. A non-zero draft flares each ruling radially outward by dist·tan(draft), producing a
// tapered/flared band (#1868). dir need not be the plane normal. Reuses the prism edge/side helpers.
func buildRuledBandDir(poly []math.Point2, plane sketch.Plane, dir math.Vector3, dist, draft float64, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	u, err := math.UnitVector3FromVector(dir)
	if err != nil {
		u = plane.Normal()
	}
	up := u.AsVector().Scale(dist)
	center := plane.ToModel(polygonCentroid2(poly))
	bottom, top := ruledBandVertices(bld, poly, plane, up, center, dist*stdmath.Tan(draft), feat)
	be, te, ve := prismEdges(bld, bottom, top, feat)
	addSides(bld, bottom, top, be, te, ve, outwardSign(poly), feat)
	return bld.Build()
}

// ruledBandVertices builds the bottom loop (the profile in model space) and the top loop (each vertex
// swept by up, then flared radially outward from center by flare for a draft taper) (#1868).
func ruledBandVertices(bld *topo.Builder, poly []math.Point2, plane sketch.Plane, up math.Vector3, center math.Point3, flare float64, feat string) (bottom, top []*topo.Vertex) {
	n := len(poly)
	bottom = make([]*topo.Vertex, n)
	top = make([]*topo.Vertex, n)
	for i, p := range poly {
		b := plane.ToModel(p)
		t := b.TranslateBy(up)
		if flare != 0 {
			if out, oerr := math.UnitVector3FromVector(center.VectorTo(b)); oerr == nil {
				t = t.TranslateBy(out.AsVector().Scale(flare))
			}
		}
		bottom[i] = bld.AddVertex(b, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(t, topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}
	return bottom, top
}

// polygonCentroid2 averages a 2D polygon's vertices.
func polygonCentroid2(poly []math.Point2) math.Point2 {
	var sx, sy float64
	for _, p := range poly {
		sx, sy = sx+float64(p.X), sy+float64(p.Y)
	}
	n := float64(len(poly))
	return math.P2(sx/n, sy/n)
}
