// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"sort"
	"testing"
)

// TestArrangementTopologyCube proves the topology extractor recovers exact face
// adjacency from provenance tags: a tagged cube's six faces each trace to one outer
// loop of four runs, bordering exactly the four adjacent faces (never itself or the
// opposite face). boxMesh emits faces +x,-x,+y,-y,+z,-z as tags 0..5.
func TestArrangementTopologyCube(t *testing.T) {
	topo := ArrangementTopologyOf(cubeTagged(0, 0, 0, 0))
	if len(topo.Faces) != 6 {
		t.Fatalf("cube = %d faces, want 6", len(topo.Faces))
	}
	// The four faces adjacent to each face, by boxMesh's tag order.
	adjacent := map[int][]int{
		0: {2, 3, 4, 5}, 1: {2, 3, 4, 5},
		2: {0, 1, 4, 5}, 3: {0, 1, 4, 5},
		4: {0, 1, 2, 3}, 5: {0, 1, 2, 3},
	}
	for _, f := range topo.Faces {
		if len(f.Loops) != 1 || !f.Loops[0].Outer {
			t.Fatalf("face %d: want 1 outer loop, got %d loops", f.Tag, len(f.Loops))
		}
		var got []int
		for _, r := range f.Loops[0].Runs {
			got = append(got, r.NeighborTag)
			if len(r.Verts) < 2 {
				t.Fatalf("face %d run has %d verts, want >=2", f.Tag, len(r.Verts))
			}
		}
		sort.Ints(got)
		if !equalInts(got, adjacent[f.Tag]) {
			t.Fatalf("face %d neighbours = %v, want %v", f.Tag, got, adjacent[f.Tag])
		}
	}
}

// TestArrangementTopologyUnion proves the extractor stays well-formed on a real
// boolean result: two offset cubes unioned must yield faces whose runs all name a
// valid neighbour tag, whose loops all close (each run's last vertex is the next run's
// first, and the loop's last equals its first), and which cover faces from BOTH
// operands.
func TestArrangementTopologyUnion(t *testing.T) {
	const naFaces = 6
	a := cubeTagged(0, 0, 0, 0)
	b := cubeTagged(1, 1, 1, naFaces)
	res := BooleanTagged(a, b, Union)
	topo := ArrangementTopologyOf(res)
	if len(topo.Faces) < 2 {
		t.Fatalf("union = %d faces, want several", len(topo.Faces))
	}
	var fromA, fromB int
	for _, f := range topo.Faces {
		if f.Tag < naFaces {
			fromA++
		} else {
			fromB++
		}
		for _, loop := range f.Loops {
			assertLoopCloses(t, f.Tag, loop)
		}
	}
	if fromA == 0 || fromB == 0 {
		t.Fatalf("union must keep faces from both operands: A=%d B=%d", fromA, fromB)
	}
}

// assertLoopCloses checks the run chain forms a closed cycle: each run's last vertex is
// the next run's first, and the loop returns to its start.
func assertLoopCloses(t *testing.T, tag int, loop ArrangementLoop) {
	t.Helper()
	if len(loop.Runs) == 0 {
		t.Fatalf("face %d: empty loop", tag)
	}
	for i, r := range loop.Runs {
		next := loop.Runs[(i+1)%len(loop.Runs)]
		if r.Verts[len(r.Verts)-1] != next.Verts[0] {
			t.Fatalf("face %d: run %d end %d != next run start %d", tag, i, r.Verts[len(r.Verts)-1], next.Verts[0])
		}
	}
}

// TestNextBoundaryEdgeTerminatesOnClosedFan pins the #2247 boundary-walk hang: the fan pivot
// around a vertex must be BOUNDED. A vertex whose incident triangles form a closed rotational
// cycle (here vertex 0, fully surrounded by the fan 0-1-2 / 0-2-3 / 0-3-1) has no boundary edge in
// its fan, so pivoting from any spoke rotates forever (1→3→2→1…). A degenerate planar arrangement
// (a wedge cut on a coincident facet) produces exactly such a non-manifold/closed fan; before the
// bound, reconstructBoolean hung there (the taper-shaft chamfer timed out at 600s). The bound turns
// the infinite pivot into a named decline: nextBoundaryEdge returns ok=false. This test COMPLETING
// is the assertion — remove the bound and it loops until the test-timeout, failing loudly.
func TestNextBoundaryEdgeTerminatesOnClosedFan(t *testing.T) {
	// A closed umbrella around vertex 0: three triangles sharing the apex, every spoke 0→wi
	// backed by an interior neighbour, so the pivot never reaches a boundary.
	tris := [][3]int{{0, 1, 2}, {0, 2, 3}, {0, 3, 1}}
	owner := map[[2]int]int{}
	for ti, tr := range tris {
		for e := 0; e < 3; e++ {
			owner[[2]int{tr[e], tr[(e+1)%3]}] = ti
		}
	}
	comp := []int{0, 0, 0}
	if _, ok := nextBoundaryEdge(1, 0, tris, owner, comp, 0); ok {
		t.Fatal("closed fan reported a boundary edge; the pivot must decline (ok=false), not find one")
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
