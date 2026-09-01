// SPDX-License-Identifier: GPL-2.0-only

package heal_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearestBoundaryEdge returns the body face's boundary edge whose midpoint is closest to center — the
// inner edge of a neighbour facing the opening, as a *topo.Edge for FillEdgeLoop.
func nearestBoundaryEdge(t *testing.T, b *topo.Body, center math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, f := range b.Faces() {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				e := u.Edge()
				c := e.Geometry()
				lo, hi := c.Domain()
				if d := float64(c.PointAt((lo + hi) / 2).DistanceTo(center)); d < bestD {
					best, bestD = e, d
				}
			}
		}
	}
	if best == nil {
		t.Fatal("no boundary edge found")
	}
	return best
}

// innerLoopEdges collects each neighbour's inner boundary edge (facing the origin) into a loop.
func innerLoopEdges(t *testing.T, bodies []*topo.Body) []*topo.Edge {
	t.Helper()
	edges := make([]*topo.Edge, len(bodies))
	for i, b := range bodies {
		edges[i] = nearestBoundaryEdge(t, b, math.P3(0, 0, 0))
	}
	return edges
}

// TestFillEdgeLoopClosesFourSided fills the square opening from an explicit loop of four surface-body
// edges (the #1867 3D-edge-loop input) and checks it interpolates the loop corners (G0 closure).
func TestFillEdgeLoopClosesFourSided(t *testing.T) {
	t.Parallel()
	bodies, verts := polygonNeighbours(t, 4, 1, flat)
	out, err := heal.FillEdgeLoop(innerLoopEdges(t, bodies), 1)
	if err != nil {
		t.Fatalf("FillEdgeLoop(4): %v", err)
	}
	fill, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want BSplineSurface", out.Faces()[0].Geometry())
	}
	fillInterpolatesVertices(t, fill, verts)
}

// TestFillEdgeLoopNonPlanar fills a NON-planar five-edge loop (alternating corner heights) — the
// headline #1867 case: a boundary patch over a 3D edge loop, not just a planar sketch profile.
func TestFillEdgeLoopNonPlanar(t *testing.T) {
	t.Parallel()
	z := func(i int) float64 {
		if i%2 == 0 {
			return 0.3
		}
		return -0.3
	}
	bodies, verts := polygonNeighbours(t, 5, 1, z)
	out, err := heal.FillEdgeLoop(innerLoopEdges(t, bodies), 2)
	if err != nil {
		t.Fatalf("FillEdgeLoop(non-planar 5): %v", err)
	}
	fill, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want BSplineSurface", out.Faces()[0].Geometry())
	}
	fillInterpolatesVertices(t, fill, verts)
	// The loop is genuinely non-planar: at least one corner sits off z=0.
	off := false
	for _, v := range verts {
		if stdmath.Abs(float64(v.Z)) > 1e-6 {
			off = true
		}
	}
	if !off {
		t.Fatal("test setup error: the loop is planar")
	}
}

// TestFillEdgeLoopTooFew: fewer than three edges cannot bound an opening.
func TestFillEdgeLoopTooFew(t *testing.T) {
	t.Parallel()
	bodies, _ := polygonNeighbours(t, 4, 1, flat)
	edges := innerLoopEdges(t, bodies)
	if _, err := heal.FillEdgeLoop(edges[:2], 1); err == nil {
		t.Error("FillEdgeLoop with 2 edges should error")
	}
}
