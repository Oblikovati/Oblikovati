// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// marchedChain is a three-point chain standing in for a marched intersection polyline, stamped with the
// achieved deviation the intersector measured for it.
func marchedChain(t *testing.T, deviation float64) geom.Polyline {
	t.Helper()
	pl, err := geom.NewMarchedPolyline([]math.Point3{math.P3(0, 0, 0), math.P3(5, 1, 0), math.P3(10, 0, 0)}, deviation)
	if err != nil {
		t.Fatalf("NewMarchedPolyline: %v", err)
	}
	return pl
}

// TestAddEdgeInheritsCurveAchievedTolerance: an edge born from a marched curve must record that curve's
// achieved deviation, and an edge born from an analytic curve must record 0 — the boolean never has to
// remember to do it, because AddEdge is the one place an edge is created (#3489).
func TestAddEdgeInheritsCurveAchievedTolerance(t *testing.T) {
	bld := NewBuilder(true, NewLineage(Tok("test", "body", 0)))
	v0 := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("test", "v", 0)))
	v1 := bld.AddVertex(math.P3(10, 0, 0), NewLineage(Tok("test", "v", 1)))
	marched := bld.AddEdge(marchedChain(t, 4.5e-4), v0, v1, NewLineage(Tok("test", "e", 0)))
	if got := marched.Tolerance(); got != 4.5e-4 {
		t.Errorf("edge from a marched curve reports tolerance %g, want 4.5e-4", got)
	}
	exact := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0)), v0, v1, NewLineage(Tok("test", "e", 1)))
	if got := exact.Tolerance(); got != 0 {
		t.Errorf("edge from an exact line segment reports tolerance %g, want 0", got)
	}
}

// TestReplaceEdgeCurveRederivesAchievedTolerance: the tolerance describes the CURVE, so swapping the
// curve must re-derive it. Zeroing it would let an edge re-geometried to a marched approximation claim
// exactness; keeping a stale one would misreport the replaced geometry (#3489).
func TestReplaceEdgeCurveRederivesAchievedTolerance(t *testing.T) {
	bld := NewBuilder(true, NewLineage(Tok("test", "body", 0)))
	v0 := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("test", "v", 0)))
	v1 := bld.AddVertex(math.P3(10, 0, 0), NewLineage(Tok("test", "v", 1)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0)), v0, v1, NewLineage(Tok("test", "e", 0)))
	bld.ReplaceEdgeCurve(e, marchedChain(t, 2e-3))
	if got := e.Tolerance(); got != 2e-3 {
		t.Errorf("after replacing with a marched curve the edge reports tolerance %g, want 2e-3", got)
	}
	bld.ReplaceEdgeCurve(e, geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0)))
	if got := e.Tolerance(); got != 0 {
		t.Errorf("after replacing back with an exact curve the edge reports tolerance %g, want 0", got)
	}
}

// TestAchievedBoundaryToleranceIsTheWorstEdge: the body-level accessor answers "how exact is this
// boundary?" with its WEAKEST link, so one marched edge among many analytic ones still surfaces.
func TestAchievedBoundaryToleranceIsTheWorstEdge(t *testing.T) {
	body := boxBodyWithOneMarchedEdge(t, 7e-4)
	if got := body.AchievedBoundaryTolerance(); got != 7e-4 {
		t.Errorf("AchievedBoundaryTolerance = %g, want the worst edge's 7e-4", got)
	}
}

// TestAchievedBoundaryToleranceOfExactBodyIsZero: a body built entirely from analytic curves describes
// its boundary exactly, so it must report 0 — otherwise the number could not be used to tell an
// approximated boundary from an exact one.
func TestAchievedBoundaryToleranceOfExactBodyIsZero(t *testing.T) {
	body := boxBodyWithOneMarchedEdge(t, 0)
	if got := body.AchievedBoundaryTolerance(); got != 0 {
		t.Errorf("AchievedBoundaryTolerance = %g for an all-analytic body, want 0", got)
	}
}

// boxBodyWithOneMarchedEdge builds a one-face body whose boundary is three exact line segments plus one
// chain carrying `deviation` — the minimum topology that exercises the body-wide worst-edge scan.
func boxBodyWithOneMarchedEdge(t *testing.T, deviation float64) *Body {
	t.Helper()
	bld := NewBuilder(false, NewLineage(Tok("test", "body", 0)))
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(10, 0, 0), math.P3(10, 10, 0), math.P3(0, 10, 0)}
	verts := make([]*Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, NewLineage(Tok("test", "v", i)))
	}
	uses := make([]Use, len(corners))
	for i := range corners {
		j := (i + 1) % len(corners)
		uses[i] = Fwd(bld.AddEdge(sideCurve(t, i, corners[i], corners[j], deviation), verts[i], verts[j], NewLineage(Tok("test", "e", i))))
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	bld.AddFace(plane, NewLineage(Tok("test", "f", 0)), OuterLoop(uses...))
	return bld.Build()
}

// sideCurve returns side 0 as a marched chain carrying `deviation` and every other side as an exact
// segment, so exactly one edge in the body is inexact.
func sideCurve(t *testing.T, side int, a, b math.Point3, deviation float64) geom.Curve3 {
	t.Helper()
	if side != 0 || deviation == 0 {
		return geom.NewLineSegment(a, b)
	}
	pl, err := geom.NewMarchedPolyline([]math.Point3{a, a.Midpoint(b), b}, deviation)
	if err != nil {
		t.Fatalf("NewMarchedPolyline: %v", err)
	}
	return pl
}
