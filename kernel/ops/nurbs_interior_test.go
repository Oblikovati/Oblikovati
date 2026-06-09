// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// uvSquare returns a densified square loop in (u,v) from (lo,lo) to (hi,hi), n points per side.
func uvSquare(lo, hi float64, n int) []math.Point2 {
	var loop []math.Point2
	add := func(u, v float64) { loop = append(loop, math.P2(math.Scalar(u), math.Scalar(v))) }
	step := (hi - lo) / float64(n)
	for i := 0; i < n; i++ {
		add(lo+float64(i)*step, lo)
	}
	for i := 0; i < n; i++ {
		add(hi, lo+float64(i)*step)
	}
	for i := 0; i < n; i++ {
		add(hi-float64(i)*step, hi)
	}
	for i := 0; i < n; i++ {
		add(lo, hi-float64(i)*step)
	}
	return loop
}

// domeSurface is a strongly-curved bicubic-ish B-spline: a 3×3 net with the centre lifted to z=2.
func domeSurface(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0), math.P3(0, 2, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 2), math.P3(1, 2, 0)},
		{math.P3(2, 0, 0), math.P3(2, 1, 0), math.P3(2, 2, 0)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func TestAdaptiveInteriorNoSpill(t *testing.T) {
	s := domeSurface(t)
	outer := uvSquare(0.1, 0.9, 8)
	holes := [][]math.Point2{uvSquare(0.4, 0.6, 6)}
	nodes := adaptiveInteriorNodes(s, outer, holes, DefaultQuality())
	if len(nodes) == 0 {
		t.Fatal("a curved patch should get interior nodes")
	}
	for _, n := range nodes {
		if !insideUVTrim(outer, holes, n) {
			t.Errorf("node %v is outside the trim (spill)", n)
		}
		if n[0] > 0.4 && n[0] < 0.6 && n[1] > 0.4 && n[1] < 0.6 {
			t.Errorf("node %v landed inside the hole", n)
		}
	}
}

func TestAdaptiveInteriorDensityFollowsCurvature(t *testing.T) {
	flat := unitPatch(t) // z=0 bilinear (from refined_patch_test.go)
	dome := domeSurface(t)
	outer := uvSquare(0.05, 0.95, 8)
	nFlat := len(adaptiveInteriorNodes(flat, outer, nil, DefaultQuality()))
	nDome := len(adaptiveInteriorNodes(dome, outer, nil, DefaultQuality()))
	if nDome <= nFlat {
		t.Errorf("curved patch should get more interior nodes than flat: dome=%d flat=%d", nDome, nFlat)
	}
}

func TestAdaptiveInteriorDeterministic(t *testing.T) {
	s := domeSurface(t)
	outer := uvSquare(0.1, 0.9, 8)
	a := adaptiveInteriorNodes(s, outer, nil, DefaultQuality())
	b := adaptiveInteriorNodes(s, outer, nil, DefaultQuality())
	if len(a) != len(b) {
		t.Fatalf("non-deterministic node count: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic node %d: %v vs %v", i, a[i], b[i])
		}
	}
}
