// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
	"testing"
)

// TestBooleanTwoBoxes is the end-to-end payoff: two unit-offset 2-cubes
// (A=[0,2]^3, B=[1,3]^3, overlap [1,2]^3 = volume 1) through each operation, gated
// by the exact expected volume AND watertightness of the result.
func TestBooleanTwoBoxes(t *testing.T) {
	a := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	b := boxMesh([3]float64{1, 1, 1}, [3]float64{3, 3, 3})
	cases := []struct {
		name string
		op   Op
		vol  float64
	}{
		{"union", Union, 15},          // 8 + 8 - 1
		{"difference", Difference, 7}, // 8 - 1
		{"intersection", Intersection, 1},
	}
	for _, tc := range cases {
		res := Boolean(a, b, tc.op)
		if len(res) == 0 {
			t.Fatalf("%s: empty result", tc.name)
		}
		if v := meshVolume(res); math.Abs(v-tc.vol) > 1e-9 {
			t.Fatalf("%s: volume = %.6f, want %.0f", tc.name, v, tc.vol)
		}
		assertWatertight(t, tc.name, res)
	}
}

// TestBooleanCoplanarBoxes overlaps two boxes offset only in x (A=[0,2]^3,
// B=[1,3]x[0,2]x[0,2]), so their top/bottom/front/back faces are coplanar and
// coincident on the SAME side over x in [1,2] — the case plain winding
// classification gets wrong (it would drop both copies and leave a hole). The
// coplanar-keep rule must retain exactly one copy. Overlap volume is 4.
func TestBooleanCoplanarBoxes(t *testing.T) {
	a := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	b := boxMesh([3]float64{1, 0, 0}, [3]float64{3, 2, 2})
	cases := []struct {
		name string
		op   Op
		vol  float64
	}{
		{"union", Union, 12},          // 8 + 8 - 4
		{"difference", Difference, 4}, // 8 - 4
		{"intersection", Intersection, 4},
	}
	for _, tc := range cases {
		res := Boolean(a, b, tc.op)
		if v := meshVolume(res); math.Abs(v-tc.vol) > 1e-9 {
			t.Fatalf("%s: volume = %.6f, want %.0f", tc.name, v, tc.vol)
		}
		assertWatertight(t, tc.name, res)
	}
}

// TestBooleanDisjointUnion unions two separated boxes: the result is just both
// boxes (volume 16) and watertight.
func TestBooleanDisjointUnion(t *testing.T) {
	a := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	b := boxMesh([3]float64{5, 5, 5}, [3]float64{7, 7, 7})
	res := Boolean(a, b, Union)
	if v := meshVolume(res); math.Abs(v-16) > 1e-9 {
		t.Fatalf("disjoint union volume = %.6f, want 16", v)
	}
	assertWatertight(t, "disjoint union", res)
}

// assertWatertight checks the mesh is a closed, orientable 2-manifold: every
// directed edge occurs exactly once and its reverse exactly once (no boundary
// edge, no non-manifold edge). Vertices are matched exactly.
func assertWatertight(t *testing.T, name string, mesh [][3]Point) {
	t.Helper()
	var verts []Point
	idx := func(p Point) int {
		for i, v := range verts {
			if v.Equal(p) {
				return i
			}
		}
		verts = append(verts, p)
		return len(verts) - 1
	}
	edge := map[[2]int]int{}
	for _, tt := range mesh {
		i, j, k := idx(tt[0]), idx(tt[1]), idx(tt[2])
		edge[[2]int{i, j}]++
		edge[[2]int{j, k}]++
		edge[[2]int{k, i}]++
	}
	for e, c := range edge {
		rev := edge[[2]int{e[1], e[0]}]
		if c != 1 || rev != 1 {
			t.Fatalf("%s: not a closed 2-manifold — directed edge %v count %d, reverse count %d", name, e, c, rev)
		}
	}
}
