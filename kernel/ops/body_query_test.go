// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestLocateUsingPointFindsEachKind: a query near a corner finds the vertex,
// near a mid-edge the edge, near a face center the face — and the kind filter
// restricts the search.
func TestLocateUsingPointFindsEachKind(t *testing.T) {
	t.Parallel()
	b := tetraBox(t, math.P3(0, 0, 0), 2)
	q := DefaultQuality()
	if hit, ok := LocateUsingPoint(b, 0, math.P3(-0.01, -0.01, -0.01), 0.1, q); !ok || hit.Kind != topo.KindVertex {
		t.Errorf("corner query = (%v, %v), want a vertex", hit.Kind, ok)
	}
	if hit, ok := LocateUsingPoint(b, 0, math.P3(1, -0.01, -0.01), 0.1, q); !ok || hit.Kind != topo.KindEdge {
		t.Errorf("mid-edge query = (%v, %v), want an edge", hit.Kind, ok)
	}
	if hit, ok := LocateUsingPoint(b, 0, math.P3(1, 1, 2.01), 0.1, q); !ok || hit.Kind != topo.KindFace {
		t.Errorf("face-center query = (%v, %v), want a face", hit.Kind, ok)
	}
	if hit, ok := LocateUsingPoint(b, topo.KindFace, math.P3(-0.01, -0.01, -0.01), 0.1, q); !ok || hit.Kind != topo.KindFace {
		t.Errorf("kind-filtered corner query = (%v, %v), want the nearest FACE", hit.Kind, ok)
	}
	if _, ok := LocateUsingPoint(b, 0, math.P3(5, 5, 5), 0.1, q); ok {
		t.Error("a far point must not locate anything within tolerance")
	}
}

// TestFindUsingRayOrdersHits: a ray through the box reports entry and exit
// faces nearest-first; findFirstOnly truncates; radius picks up a grazed edge.
func TestFindUsingRayOrdersHits(t *testing.T) {
	t.Parallel()
	b := tetraBox(t, math.P3(0, 0, 0), 2)
	q := DefaultQuality()
	hits := FindUsingRay(b, math.P3(1, 1, -5), math.V3(0, 0, 1), 0, q, false)
	if len(hits) != 2 {
		t.Fatalf("through-ray hit %d faces, want 2 (entry+exit)", len(hits))
	}
	if hits[0].Distance >= hits[1].Distance {
		t.Error("hits must be sorted nearest first")
	}
	if first := FindUsingRay(b, math.P3(1, 1, -5), math.V3(0, 0, 1), 0, q, true); len(first) != 1 {
		t.Errorf("findFirstOnly returned %d hits, want 1", len(first))
	}
	// A ray skimming 1e-3 outside the x=0,y=0 vertical edge only shows up
	// with a radius.
	if grazing := FindUsingRay(b, math.P3(-0.001, -0.001, -5), math.V3(0, 0, 1), 0, q, false); len(grazing) != 0 {
		t.Errorf("zero-radius grazing ray hit %d entities, want 0", len(grazing))
	}
	grazed := FindUsingRay(b, math.P3(-0.001, -0.001, -5), math.V3(0, 0, 1), 0.01, q, false)
	foundEdge := false
	for _, h := range grazed {
		if h.Kind == topo.KindEdge {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Error("radius ray should pick up the grazed vertical edge")
	}
}

// TestBodyEdgeConvexityCube: every cube edge is convex.
func TestBodyEdgeConvexityCube(t *testing.T) {
	t.Parallel()
	byClass := blend.BodyEdgeConvexity(tetraBox(t, math.P3(0, 0, 0), 2))
	if n := len(byClass[blend.EdgeConvex]); n != 12 {
		t.Errorf("cube has %d convex edges, want 12 (got concave=%d tangent=%d unknown=%d)",
			n, len(byClass[blend.EdgeConcave]), len(byClass[blend.EdgeTangent]), len(byClass[blend.EdgeConvexityUnknown]))
	}
}

// TestBodyEdgeConvexityCavity: the cavity's inner skin edges are concave from
// the material's perspective (the material wraps 270° around them).
func TestBodyEdgeConvexityCavity(t *testing.T) {
	t.Parallel()
	byClass := blend.BodyEdgeConvexity(cavityBody(t))
	if n := len(byClass[blend.EdgeConcave]); n != 12 {
		t.Errorf("cavity body has %d concave edges, want the 12 inner ones (convex=%d)",
			n, len(byClass[blend.EdgeConvex]))
	}
	if n := len(byClass[blend.EdgeConvex]); n != 12 {
		t.Errorf("cavity body has %d convex edges, want the 12 outer ones", n)
	}
}

// TestOrientedMinimumRangeBoxRotatedBox: a box rotated 45° about Z still gets
// a tight OBB of its true dimensions (an AABB would inflate by √2).
func TestOrientedMinimumRangeBoxRotatedBox(t *testing.T) {
	t.Parallel()
	src := tetraBox(t, math.P3(0, 0, 0), 2)
	axis, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	rot := math.Rotation4(stdmath.Pi/4, axis, math.P3(0, 0, 0))
	rotated, err := transform.TransformBody(src, rot, func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("transform.TransformBody: %v", err)
	}
	obb, err := OrientedMinimumRangeBox(rotated)
	if err != nil {
		t.Fatalf("OrientedMinimumRangeBox: %v", err)
	}
	if v := obb.Volume(); stdmath.Abs(v-8) > 1e-6 {
		t.Errorf("rotated-box OBB volume = %g, want 8 (tight)", v)
	}
	edges := obb.EdgeVectors()
	dims := []float64{
		float64(edges[0].Length()),
		float64(edges[1].Length()),
		float64(edges[2].Length()),
	}
	for _, d := range dims {
		if stdmath.Abs(d-2) > 1e-6 {
			t.Errorf("OBB dimensions = %v, want all 2", dims)
			break
		}
	}
}

// TestPreciseRangeBoxSeesFaceBulge is guarded by the analytic cylinder: the
// topology RangeBox already samples edges, so both should agree on a box —
// the point is PreciseRangeBox includes mesh interiors and wires.
func TestPreciseRangeBoxCoversBody(t *testing.T) {
	t.Parallel()
	b := tetraBox(t, math.P3(1, 1, 1), 2)
	box := PreciseRangeBox(b, DefaultQuality())
	if box.Min.X > 1+1e-9 || box.Max.X < 3-1e-9 {
		t.Errorf("precise box = %+v, want [1,3] on X", box)
	}
}

// TestValidateBodyEntitiesLevels: a clean solid passes both levels; merged
// interpenetrating shells pass topology but fail the geometry level with
// face problems carrying keys.
func TestValidateBodyEntitiesLevels(t *testing.T) {
	t.Parallel()
	clean := tetraBox(t, math.P3(0, 0, 0), 2)
	if ok, problems := ValidateBodyEntities(clean, CheckGeometry, DefaultQuality()); !ok {
		t.Errorf("clean box reports problems: %+v", problems)
	}
	a := tetra(1, math.V3(0, 0, 0))
	bb := tetra(1, math.V3(0.2, 0.2, 0.2))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), true, a, bb)
	if ok, _ := ValidateBodyEntities(merged, CheckTopology, DefaultQuality()); !ok {
		t.Error("interpenetrating shells are still topologically valid")
	}
	ok, problems := ValidateBodyEntities(merged, CheckGeometry, DefaultQuality())
	if ok || len(problems) == 0 {
		t.Fatal("geometry-level check must flag the interpenetration")
	}
	if problems[0].Kind != topo.KindFace || len(problems[0].ReferenceKey) == 0 {
		t.Errorf("problem entity = %+v, want a face with a reference key", problems[0])
	}
}

// TestEntityLevelChecks: per-entity validity verdicts.
func TestEntityLevelChecks(t *testing.T) {
	t.Parallel()
	b := tetraBox(t, math.P3(0, 0, 0), 2)
	for _, e := range b.Edges() {
		if !EdgeEntityValid(e) {
			t.Fatalf("cube edge %d should be valid", e.ID())
		}
	}
	for _, f := range b.Faces() {
		if !FaceEntityValid(f) {
			t.Fatalf("cube face %d should be valid", f.ID())
		}
	}
}

// TestBindTransientKey: each entity's session id binds back to it; an unknown
// key reports false.
func TestBindTransientKey(t *testing.T) {
	t.Parallel()
	b := tetraBox(t, math.P3(0, 0, 0), 2)
	f := b.Faces()[2]
	ref, ok := b.BindTransientKey(f.ID())
	if !ok || ref.Kind != topo.KindFace || ref.Face != f {
		t.Errorf("face key bound to %+v (ok=%v), want the face back", ref, ok)
	}
	sh := b.Shells()[0]
	if ref, ok := b.BindTransientKey(sh.ID()); !ok || ref.Kind != topo.KindShell || ref.Shell != sh {
		t.Error("shell key must bind to the shell")
	}
	if _, ok := b.BindTransientKey(0xFFFFFFFFFFFF); ok {
		t.Error("an unknown transient key must not bind")
	}
}
