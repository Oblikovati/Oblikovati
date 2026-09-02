// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
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

// TestSharedContactHoldsACurvePointAndAVertex covers sharedContact.holds directly: a point on the
// SECOND shared curve must match (the scan continues past a non-matching first), a point at a shared
// vertex must match, and a point off all of it must not — plus the empty-contact case.
func TestSharedContactHoldsACurvePointAndAVertex(t *testing.T) {
	t.Parallel()
	p := gmath.P3
	shared := sharedContact{
		curves: []geom.Curve3{
			geom.NewLineSegment(p(0, 0, 0), p(1, 0, 0)),  // first curve (far from the probe)
			geom.NewLineSegment(p(0, 5, 0), p(0, 5, 10)), // second curve (the probe lies on this one)
		},
		points: []gmath.Point3{p(7, 7, 7)},
	}
	if !shared.holds(p(0, 5, 4), 1e-6) {
		t.Error("a point on the second shared curve should be reported as shared contact")
	}
	if !shared.holds(p(7, 7, 7), 1e-6) {
		t.Error("a point at a shared vertex should be reported as shared contact")
	}
	if shared.holds(p(9, 9, 9), 1e-6) {
		t.Error("a point far from every shared entity should not be reported as shared contact")
	}
	if (sharedContact{}).holds(p(0, 0, 0), 1e-6) {
		t.Error("with no shared topology nothing is shared contact")
	}
}

// TestSharedContactUsesTheEdgeCurveNotItsChord is the regression for the filter's own trap: a shared
// ARC bows away from the chord between its vertices by the sagitta, so a chord-based filter reads the
// legitimate contact along a tangent blend's edge as an interpenetration. Here the arc's midpoint sits
// 1 − cos(π/4) ≈ 0.293 off its own chord, three decades above the tolerance asked for.
func TestSharedContactUsesTheEdgeCurveNotItsChord(t *testing.T) {
	t.Parallel()
	arc, err := geom.NewArc3d(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), gmath.V3(1, 0, 0), 1, 0, math.Pi/2)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	shared := sharedContact{curves: []geom.Curve3{arc}}
	if mid := arc.PointAt(0.5); !shared.holds(mid, 1e-9) {
		t.Errorf("the shared arc's own midpoint %v must read as shared contact", mid)
	}
}

// TestSelfIntersectionSharedVertexPokeThrough is the core #1321 regression: two faces share one
// apex vertex but one pierces the other far from it. Exactly one self-intersection, witness off apex.
func TestSelfIntersectionSharedVertexPokeThrough(t *testing.T) {
	t.Parallel()
	hits := SelfIntersections(bowtieBody(), mesh.DefaultQuality())
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
	t.Parallel()
	if hits := SelfIntersections(tentBody(), mesh.DefaultQuality()); len(hits) != 0 {
		t.Errorf("tent (clean shared edge) reports %d self-intersections, want 0: %+v", len(hits), hits)
	}
}

// TestSharedFaceContactFindsEdgeAndVertex checks the collector directly: the tent's two faces share an
// edge (so the contact carries its curve); the bowtie's share only the apex vertex (points, no curve).
func TestSharedFaceContactFindsEdgeAndVertex(t *testing.T) {
	t.Parallel()
	tent := tentBody()
	shared := sharedFaceContact(tent.Faces()[0], tent.Faces()[1])
	if len(shared.curves) != 1 {
		t.Errorf("tent faces should share exactly one edge curve, got %d", len(shared.curves))
	}
	bowtie := bowtieBody()
	bow := sharedFaceContact(bowtie.Faces()[0], bowtie.Faces()[1])
	if len(bow.curves) != 0 {
		t.Errorf("bowtie faces share no edge, got %d curve(s)", len(bow.curves))
	}
	if len(bow.points) != 1 || bow.points[0].DistanceTo(gmath.P3(0, 0, 0)) > 1e-9 {
		t.Errorf("bowtie faces should share exactly the apex vertex, got %+v", bow.points)
	}
}
