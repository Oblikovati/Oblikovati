// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
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

// BoundaryPatchDefinition is the recipe for a boundary patch: the boundary loops it
// fills. Phase A fills the first closed planar loop (with the others as holes).
type BoundaryPatchDefinition struct {
	Loops *BoundaryPatchLoops
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
	bf := &BoundaryPatchFeature{def: &BoundaryPatchDefinition{Loops: loops}, featName: "BoundaryPatch"}
	pf := c.engine.Add(bf)
	bf.featName = pf.name
	return pf
}

// RuledSurfaceType is the direction convention for a ruled surface's straight
// rulings (Inventor's RuledSurfaceTypeEnum): along the source plane Normal, Tangent
// to the adjacent face, or Perpendicular to a reference plane.
type RuledSurfaceType int

const (
	// RuledNormal rules along the source profile-plane normal (phase A, real).
	RuledNormal RuledSurfaceType = iota
	// RuledTangent rules tangent to the adjacent face (needs face data; phase B).
	RuledTangent
	// RuledPerpendicular rules perpendicular to a reference plane (phase B).
	RuledPerpendicular
)

// RuledSurfaceDefinition is the recipe for a ruled surface: a profile whose edges
// are swept by straight rulings over a distance.
type RuledSurfaceDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Type         RuledSurfaceType
	Distance     func() float64
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

// Recompute resolves the profile and builds the ruled band. Tangent/perpendicular
// rulings need the adjacent face's tangent / a reference plane, which is phase B —
// the inputs are validated then the geometry is deferred (Warning, passthrough).
func (r *RuledSurfaceFeature) Recompute(in Input) (Output, error) {
	profile, err := resolveClosedProfile(r.def.Sketch, r.def.ProfileIndex, "ruled surface")
	if err != nil {
		return Output{}, err
	}
	if r.def.Type != RuledNormal {
		return Output{Bodies: in.Bodies}, ErrDeferred
	}
	dist := measure(r.def.Distance)
	if dist == 0 {
		return Output{}, errors.New("ruled surface: distance is zero")
	}
	band := buildRuledBand(profile.OuterLoop().Polygon(), r.def.Sketch.Plane(), dist, r.featName)
	return Output{Bodies: appendBody(in.Bodies, band)}, nil
}

// RuledSurfaceFeatures adds ruled surfaces into the engine.
type RuledSurfaceFeatures struct{ engine *PartFeatures }

// NewRuledSurfaceFeatures binds the collection to a feature engine.
func NewRuledSurfaceFeatures(engine *PartFeatures) *RuledSurfaceFeatures {
	return &RuledSurfaceFeatures{engine: engine}
}

// AddByDistance rules the closed profileIndex profile of skt by distance, in the
// kind direction.
func (c *RuledSurfaceFeatures) AddByDistance(skt *sketch.Sketch, profileIndex int, kind RuledSurfaceType, distance func() float64) *PartFeature {
	def := &RuledSurfaceDefinition{Sketch: skt, ProfileIndex: profileIndex, Type: kind, Distance: distance}
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

// buildRuledBand sweeps a closed polygon by distance along the plane normal into a
// band of planar quad faces (no caps) — an open surface body whose straight rulings
// connect the bottom and top loops. Reuses the prism edge/side helpers (extrude.go).
func buildRuledBand(poly []math.Point2, plane sketch.Plane, dist float64, feat string) *topo.Body {
	n := len(poly)
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	up := plane.Normal().AsVector().Scale(dist)
	bottom := make([]*topo.Vertex, n)
	top := make([]*topo.Vertex, n)
	for i, p := range poly {
		b := plane.ToModel(p)
		bottom[i] = bld.AddVertex(b, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(b.TranslateBy(up), topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}
	be, te, ve := prismEdges(bld, bottom, top, feat)
	addSides(bld, bottom, top, be, te, ve, outwardSign(poly), feat)
	return bld.Build()
}
