// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1321: SelfIntersections skipped any face pair sharing a single
// vertex, hiding interpenetrations away from that vertex. The two fixtures below share exactly one
// vertex (bowtie) or exactly one edge (tent) so the shared-topology pointers are real.

// triFace adds one triangular face through the three given (already-added) vertices to bld, in a plane
// fit to the triangle's winding.
func triFace(bld *topo.Builder, feat string, v0, v1, v2 *topo.Vertex) {
	p0, p1, p2 := v0.Point(), v1.Point(), v2.Point()
	surf, _ := geom.NewPlane(p0, p0.VectorTo(p1).Cross(p0.VectorTo(p2)))
	e0 := bld.AddEdge(geom.NewLineSegment(p0, p1), v0, v1, topo.NewLineage(topo.Tok(feat, "e", 0)))
	e1 := bld.AddEdge(geom.NewLineSegment(p1, p2), v1, v2, topo.NewLineage(topo.Tok(feat, "e", 1)))
	e2 := bld.AddEdge(geom.NewLineSegment(p2, p0), v2, v0, topo.NewLineage(topo.Tok(feat, "e", 2)))
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "f", 0)),
		topo.OuterLoop(topo.Fwd(e0), topo.Fwd(e1), topo.Fwd(e2)))
}

// bowtieBody builds two triangles sharing ONLY the apex vertex at the origin. Face A lies in z=0;
// face B is tilted so its far edge pierces z=0 at (3,1.5,0) — inside face A's interior and far from
// the shared apex. This is a real interpenetration the old vertex-skip rule hid.
func bowtieBody() *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("bowtie", "body", 0)))
	p := gmath.P3
	apex := bld.AddVertex(p(0, 0, 0), topo.NewLineage(topo.Tok("bowtie", "v", 0)))
	a1 := bld.AddVertex(p(4, 0, 0), topo.NewLineage(topo.Tok("bowtie", "v", 1)))
	a2 := bld.AddVertex(p(4, 4, 0), topo.NewLineage(topo.Tok("bowtie", "v", 2)))
	b1 := bld.AddVertex(p(3, 1.5, -1), topo.NewLineage(topo.Tok("bowtie", "v", 3)))
	b2 := bld.AddVertex(p(3, 1.5, 1), topo.NewLineage(topo.Tok("bowtie", "v", 4)))
	triFace(bld, "A", apex, a1, a2)
	triFace(bld, "B", apex, b1, b2)
	return bld.Build()
}

// tentBody builds two triangles sharing ONE edge (the ridge from (0,0,0) to (4,0,0)), folded like a
// roof so they meet only along that edge — a legitimate manifold contact that must NOT be flagged.
func tentBody() *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("tent", "body", 0)))
	p := gmath.P3
	r0 := bld.AddVertex(p(0, 0, 0), topo.NewLineage(topo.Tok("tent", "v", 0)))
	r1 := bld.AddVertex(p(4, 0, 0), topo.NewLineage(topo.Tok("tent", "v", 1)))
	left := bld.AddVertex(p(2, 2, 1), topo.NewLineage(topo.Tok("tent", "v", 2)))
	right := bld.AddVertex(p(2, -2, 1), topo.NewLineage(topo.Tok("tent", "v", 3)))
	ridge := bld.AddEdge(geom.NewLineSegment(p(0, 0, 0), p(4, 0, 0)), r0, r1, topo.NewLineage(topo.Tok("tent", "ridge", 0)))
	addTriWithRidge(bld, "L", ridge, r0, r1, left)
	addTriWithRidge(bld, "R", ridge, r1, r0, right)
	return bld.Build()
}

// addTriWithRidge adds a triangle ( va, vb, apex) whose first edge IS the shared ridge edge, so both
// tent faces reference the same *topo.Edge.
func addTriWithRidge(bld *topo.Builder, feat string, ridge *topo.Edge, va, vb, apex *topo.Vertex) {
	pa, pb, pc := va.Point(), vb.Point(), apex.Point()
	surf, _ := geom.NewPlane(pa, pa.VectorTo(pb).Cross(pa.VectorTo(pc)))
	e1 := bld.AddEdge(geom.NewLineSegment(pb, pc), vb, apex, topo.NewLineage(topo.Tok(feat, "e", 1)))
	e2 := bld.AddEdge(geom.NewLineSegment(pc, pa), apex, va, topo.NewLineage(topo.Tok(feat, "e", 2)))
	ridgeUse := topo.Use{Edge: ridge, Reversed: ridge.StartVertex() != va}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "f", 0)),
		topo.OuterLoop(ridgeUse, topo.Fwd(e1), topo.Fwd(e2)))
}

// TestOnSharedBoundaryMatchesAnySegment covers onSharedBoundary directly: a point on the SECOND
// shared segment must match (the loop continues past a non-matching first), and a point off all
// segments must not — plus the empty-boundary case.
func TestOnSharedBoundaryMatchesAnySegment(t *testing.T) {
	p := gmath.P3
	shared := [][2]gmath.Point3{
		{p(0, 0, 0), p(1, 0, 0)},  // first segment (far from the probe)
		{p(0, 5, 0), p(0, 5, 10)}, // second segment (the probe lies on this one)
	}
	if !onSharedBoundary(p(0, 5, 4), shared, 1e-6) {
		t.Error("a point on the second shared segment should be reported on the boundary")
	}
	if onSharedBoundary(p(9, 9, 9), shared, 1e-6) {
		t.Error("a point far from every shared segment should not be on the boundary")
	}
	if onSharedBoundary(p(0, 0, 0), nil, 1e-6) {
		t.Error("with no shared boundary nothing is on it")
	}
}

// TestSelfIntersectionSharedVertexPokeThrough is the core #1321 regression: two faces share one
// apex vertex but one pierces the other far from it. Exactly one self-intersection, witness off apex.
func TestSelfIntersectionSharedVertexPokeThrough(t *testing.T) {
	hits := SelfIntersections(bowtieBody(), DefaultQuality())
	if len(hits) == 0 {
		t.Fatal("shared-vertex poke-through must be reported (was hidden by the vertex-skip rule)")
	}
	for _, h := range hits {
		if d := float64(h.Witness.DistanceTo(gmath.P3(0, 0, 0))); d < 0.5 {
			t.Errorf("witness %v is at the shared apex (dist %g); a real crossing should be off it", h.Witness, d)
		}
	}
}

// TestSelfIntersectionSharedEdgeClean is the negative control: two faces meeting only along their
// shared edge (a roof ridge) are legitimate and must report nothing.
func TestSelfIntersectionSharedEdgeClean(t *testing.T) {
	if hits := SelfIntersections(tentBody(), DefaultQuality()); len(hits) != 0 {
		t.Errorf("tent (clean shared edge) reports %d self-intersections, want 0: %+v", len(hits), hits)
	}
}

// TestSharedFaceBoundaryFindsEdgeAndVertex checks the helper directly: the tent's two faces share an
// edge (a non-degenerate segment); the bowtie's share a point (a degenerate segment).
func TestSharedFaceBoundaryFindsEdgeAndVertex(t *testing.T) {
	tent := tentBody()
	shared := sharedFaceBoundary(tent.Faces()[0], tent.Faces()[1])
	hasSegment := false
	for _, s := range shared {
		if s[0].DistanceTo(s[1]) > 1e-9 {
			hasSegment = true
		}
	}
	if !hasSegment {
		t.Errorf("tent faces should share a non-degenerate edge segment, got %+v", shared)
	}
	bow := bowtieBody()
	bshared := sharedFaceBoundary(bow.Faces()[0], bow.Faces()[1])
	if len(bshared) == 0 {
		t.Error("bowtie faces should share the apex vertex point")
	}
	for _, s := range bshared {
		if math.Abs(float64(s[0].DistanceTo(s[1]))) > 1e-9 {
			t.Errorf("bowtie shared boundary should be a point, got segment %+v", s)
		}
	}
}
