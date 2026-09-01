// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

// TestInsideExactCube checks the exact ray-cast classifier on a cube. The centre
// (0.5,0.5,0.5) is deliberate: its +x ray hits the +x face exactly on that face's
// triangle diagonal, a degeneracy, so this also exercises the retry-in-another-
// direction path.
func TestInsideExactCube(t *testing.T) {
	t.Parallel()
	cube := boxMesh([3]float64{0, 0, 0}, [3]float64{1, 1, 1})
	grid := newFaceGrid(cube)
	inside := []Point{
		pt([3]float64{0.5, 0.5, 0.5}), // ray along +x grazes the face diagonal → retry
		pt([3]float64{0.1, 0.9, 0.5}),
		pt([3]float64{0.01, 0.01, 0.01}),
	}
	outside := []Point{
		pt([3]float64{2, 2, 2}),
		pt([3]float64{-0.5, 0.5, 0.5}),
		pt([3]float64{0.5, 0.5, 1.5}),
	}
	for _, p := range inside {
		if !insideExact(p, cube, grid) {
			t.Fatalf("point %v classified outside the cube", p.Round())
		}
	}
	for _, p := range outside {
		if insideExact(p, cube, grid) {
			t.Fatalf("point %v classified inside the cube", p.Round())
		}
	}
}

func TestSegmentPiercesTriExact(t *testing.T) {
	t.Parallel()
	a := pt([3]float64{0, 0, 0})
	b := pt([3]float64{4, 0, 0})
	c := pt([3]float64{0, 4, 0})
	cases := []struct {
		name              string
		p, q              Point
		cross, degenerate bool
	}{
		{"clean pierce", pt([3]float64{1, 1, -1}), pt([3]float64{1, 1, 1}), true, false},
		{"straddle but miss", pt([3]float64{5, 5, -1}), pt([3]float64{5, 5, 1}), false, false},
		{"same side", pt([3]float64{1, 1, 1}), pt([3]float64{1, 1, 2}), false, false},
		{"endpoint on plane", pt([3]float64{1, 1, 0}), pt([3]float64{1, 1, 1}), false, true},
		{"grazes an edge", pt([3]float64{2, 0, -1}), pt([3]float64{2, 0, 1}), false, true},
	}
	for _, tc := range cases {
		cross, degen := segmentPiercesTriExact(tc.p, tc.q, a, b, c)
		if cross != tc.cross || degen != tc.degenerate {
			t.Fatalf("%s: got (cross=%v, degen=%v), want (%v, %v)", tc.name, cross, degen, tc.cross, tc.degenerate)
		}
	}
}
