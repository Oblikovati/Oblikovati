// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-seam corner-edge selection must be robust to body EDGE ORDER (#2247). Several vertical
// edges stand over one bent corner — the two walls' tall flat-face edges and the sheet's short
// through-thickness corner edge. The lap tab lies on a wall's flat face, so the seam must pick a
// TALL wall edge, not the short thickness edge, whatever order the edges arrive in. First-match
// order used to decide it, which the analytic and reconstructed booleans need not share (they order
// edges differently), so a reconstruction-shaped body silently built a tab one thickness tall.

// vertEdge builds a Z-aligned straight edge between (x,y,z0) and (x,y,z1) on a throwaway builder.
func vertEdge(t *testing.T, bld *topo.Builder, x, y, z0, z1 float64, i int) *topo.Edge {
	t.Helper()
	p0, p1 := math.P3(x, y, z0), math.P3(x, y, z1)
	v0 := bld.AddVertex(p0, topo.NewLineage(topo.Tok("t", "v", 2*i)))
	v1 := bld.AddVertex(p1, topo.NewLineage(topo.Tok("t", "v", 2*i+1)))
	return bld.AddEdge(geom.NewLineSegment(p0, p1), v0, v1, topo.NewLineage(topo.Tok("t", "e", i)))
}

// cornerBends returns two perpendicular bends sharing the corner (0,0,0.2), both folded up (+Z) —
// the junction the seam finishes.
func cornerBends() []BendPlacement {
	up := math.V3(0, 0, 1).AsUnit()
	return []BendPlacement{
		{AxisStart: math.P3(0, 0, 0.2), AxisEnd: math.P3(4, 0, 0.2), Up: up, Radius: 0.3, Thickness: 0.2},
		{AxisStart: math.P3(0, 0, 0.2), AxisEnd: math.P3(0, 4, 0.2), Up: up, Radius: 0.3, Thickness: 0.2},
	}
}

// TestSeamCornersPicksWallFaceEdgeRegardlessOfOrder: with the short thickness edge before OR after
// the tall wall edge, the seam finishes the same corner on the tall wall edge.
func TestSeamCornersPicksWallFaceEdgeRegardlessOfOrder(t *testing.T) {
	t.Parallel()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("t", "body", 0)))
	thickness := vertEdge(t, bld, 0, 0, 0.0, 0.2, 0) // sheet corner, sits AT the junction, 0.2 tall
	wallA := vertEdge(t, bld, 0, -0.3, 0.5, 1.5, 1)  // wall flat-face edge, stands off, 1.0 tall
	wallB := vertEdge(t, bld, -0.3, 0, 0.5, 1.5, 2)  // the other wall's flat-face edge

	for _, tc := range []struct {
		name  string
		edges []*topo.Edge
	}{
		{"thickness-first", []*topo.Edge{thickness, wallA, wallB}},
		{"thickness-last", []*topo.Edge{wallA, wallB, thickness}},
		{"thickness-middle", []*topo.Edge{wallA, thickness, wallB}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corners := seamCorners(tc.edges, cornerBends())
			if len(corners) != 1 {
				t.Fatalf("want exactly 1 finished corner, got %d", len(corners))
			}
			_, rise := edgeVerticalSpan(corners[0].edge, corners[0].j.at, corners[0].j.a.Up)
			if rise < 0.9 {
				t.Fatalf("seam picked a %.3f-tall edge (the through-thickness edge), want the ~1.0 wall face", rise)
			}
		})
	}
}
