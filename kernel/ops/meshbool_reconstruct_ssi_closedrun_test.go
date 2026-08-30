// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	m "oblikovati.org/math"
)

// A closed run matched to a non-periodic (unbounded-domain) SSI curve must decline, not emit an
// Inf edge (#2247, ADR-0056 Layer 3). The chained coplanar proud-tab Join makes the arrangement
// trace a DEGENERATE collinear sliver as a "closed" run — its vertices run down a shared vertical
// edge and back to the start (z: 1.5 → 1.1 → 0.5 → 1.5, all on the line x=-0.5, y=-0.3). Its two
// incident planes are transverse, so IntersectSurfacesAnalytic hands back a LINE, whose domain is
// ±Inf. Without the guard, orientRunEdge → closedRunEdge would take that unbounded domain as the
// edge interval and synthesize Inf/NaN endpoints — a body that Validate rejects only after
// tessellation can already panic on the non-finite coordinate (predicates.Orient2D on ±Inf). The
// guard refuses it at classification instead, leaving the caller on the exact faceted fallback.

// closedSliverRun builds the degenerate 4-vertex closed run and the two transverse planes whose
// analytic intersection is the vertical line the run lies on — the reproduction from the chained
// corner-seam proud-tab Join.
func closedSliverRun() (meshbool.ArrangementRun, []meshbool.Point, geom.Surface, geom.Surface, geom.Resolution) {
	verts := []meshbool.Point{
		meshbool.FromCoords(-0.5, -0.3, 1.5),
		meshbool.FromCoords(-0.5, -0.3, 1.1),
		meshbool.FromCoords(-0.5, -0.3, 0.5),
		meshbool.FromCoords(-0.5, -0.3, 1.5), // == vert 0: the run loops back on itself
	}
	run := meshbool.ArrangementRun{Verts: []int{0, 1, 2, 3}}
	wall, _ := geom.NewPlane(m.P3(-0.5, -0.3, 0), m.V3(1, 0, 0)) // x = -0.5
	side, _ := geom.NewPlane(m.P3(-0.5, -0.3, 0), m.V3(0, 1, 0)) // y = -0.3
	res := geom.ResolutionForBox(m.NewBox(m.P3(-1, -1, 0), m.P3(1, 1, 2)))
	return run, verts, wall, side, res
}

// TestIntersectionRunEdgeDeclinesClosedRunOnLine: the degenerate closed sliver run, whose SSI is an
// unbounded line, must decline — the guard's whole purpose.
func TestIntersectionRunEdgeDeclinesClosedRunOnLine(t *testing.T) {
	run, verts, wall, side, res := closedSliverRun()

	// Preconditions: the run IS closed and the SSI IS an unbounded line (else the test proves nothing).
	if !runIsClosed(run, verts, res.Weld()) {
		t.Fatal("fixture run is not closed; the guard would not apply")
	}
	curves, handled := geom.IntersectSurfacesAnalytic(wall, side, res)
	if !handled || len(curves) != 1 {
		t.Fatalf("wall∩side must be one analytic curve, got handled=%v n=%d", handled, len(curves))
	}
	if boundedDomain(curves[0]) {
		t.Fatalf("wall∩side must be an unbounded line, got a bounded-domain %T", curves[0])
	}

	edge, ok := intersectionRunEdge(run, wall, side, verts, res)
	if ok {
		// The guard failed: prove the harm it was meant to prevent — the emitted edge's endpoints
		// are non-finite, which crashes downstream planar triangulation.
		a, b := edge.Curve.PointAt(edge.T0), edge.Curve.PointAt(edge.T1)
		t.Fatalf("closed run on an unbounded line reconstructed instead of declining; "+
			"edge endpoints a=%v b=%v (T0=%v T1=%v)", a, b, edge.T0, edge.T1)
	}
}

// TestBoundedDomainDistinguishesLineFromCircle locks the predicate the decline rests on: a periodic
// circle (a plane∩cylinder ⊥ axis) has a bounded domain and must NOT be declined; only the line is.
func TestBoundedDomainDistinguishesLineFromCircle(t *testing.T) {
	line, _ := geom.NewLine(m.P3(0, 0, 0), m.V3(0, 0, 1))
	if boundedDomain(line) {
		t.Error("a line has an unbounded (±Inf) domain; boundedDomain must report false")
	}
	circle, err := geom.NewCircle(m.P3(0, 0, 0), m.V3(0, 0, 1), 1)
	if err != nil {
		t.Fatalf("circle: %v", err)
	}
	if !boundedDomain(circle) {
		t.Error("a circle has a bounded ([0,2π]) domain; boundedDomain must report true")
	}
}
