// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// uPatch builds a degree-3 5×5 B-spline patch occupying x ∈ [xoff, xoff+1], y ∈ [0,1], with the
// height field z(i,j). Clamped uniform knots in both directions (equal intervals → join ratio 1).
func uPatch(t *testing.T, xoff float64, z func(i, j int) float64) BSplineSurface {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(math.Scalar(xoff+float64(i)*0.25), math.Scalar(float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	k := clampedUniformKnots(n-1, 3)
	s, err := NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	return s
}

// seamDersMatch checks that s's u=0 cross-derivatives equal t's u=1 cross-derivatives up to `order`
// at sampled v — i.e. the matched seam is C^order (hence G^order).
func seamDersMatch(t *testing.T, s, target BSplineSurface, order int, tol float64) {
	t.Helper()
	for _, v := range []float64{0, 0.25, 0.5, 0.75, 1} {
		sd := s.SurfaceDersAt(0, v, order, 0)
		td := target.SurfaceDersAt(1, v, order, 0)
		for k := 0; k <= order; k++ {
			if !sd[k][0].IsEqualTo(td[k][0], math.Scalar(tol)) {
				t.Fatalf("G%d mismatch at v=%g, derivative order %d: %v vs %v", order, v, k, sd[k][0], td[k][0])
			}
		}
	}
}

func TestMatchSurfaceG0CoincidesEdge(t *testing.T) {
	target := uPatch(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) })
	s := uPatch(t, 1, func(i, j int) float64 { return 0 }) // flat slab to the right
	got, err := MatchSurface(s, target, UMinEdge, UMaxEdge, 0)
	if err != nil {
		t.Fatalf("MatchSurface G0: %v", err)
	}
	seamDersMatch(t, got, target, 0, 1e-9)
}

func TestMatchSurfaceG1IsTangent(t *testing.T) {
	target := uPatch(t, 0, func(i, j int) float64 { return 0.3 * float64(i) })
	s := uPatch(t, 1, func(i, j int) float64 { return 0 })
	got, err := MatchSurface(s, target, UMinEdge, UMaxEdge, 1)
	if err != nil {
		t.Fatalf("MatchSurface G1: %v", err)
	}
	seamDersMatch(t, got, target, 1, 1e-9)
	// G1 reflection of the control polygon: row1 = 2·row0 − target's second-to-last row.
	p0, p1 := target.Ctrl[4][2], target.Ctrl[3][2]
	wantRow1 := math.P3(2*p0.X-p1.X, 2*p0.Y-p1.Y, 2*p0.Z-p1.Z)
	if !got.Ctrl[1][2].IsEqualTo(wantRow1, 1e-12) {
		t.Errorf("G1 row1 = %v, want reflected %v", got.Ctrl[1][2], wantRow1)
	}
}

func TestMatchSurfaceG2IsCurvatureContinuous(t *testing.T) {
	target := uPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) }) // genuinely curved in u
	s := uPatch(t, 1, func(i, j int) float64 { return 0 })
	got, err := MatchSurface(s, target, UMinEdge, UMaxEdge, 2)
	if err != nil {
		t.Fatalf("MatchSurface G2: %v", err)
	}
	seamDersMatch(t, got, target, 2, 1e-9)
}

func TestMatchSurfaceG3(t *testing.T) {
	target := uPatch(t, 0, func(i, j int) float64 { return 0.2 * float64(i*i*i) })
	s := uPatch(t, 1, func(i, j int) float64 { return 0 })
	got, err := MatchSurface(s, target, UMinEdge, UMaxEdge, 3)
	if err != nil {
		t.Fatalf("MatchSurface G3: %v", err)
	}
	seamDersMatch(t, got, target, 3, 1e-9)
}

// vPatch builds a degree-3 5×5 patch occupying y ∈ [yoff, yoff+1], x ∈ [0,1] — for V-edge matching.
func vPatch(t *testing.T, yoff float64, z func(i, j int) float64) BSplineSurface {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(math.Scalar(float64(i)*0.25), math.Scalar(yoff+float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	k := clampedUniformKnots(n-1, 3)
	s, err := NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("vpatch: %v", err)
	}
	return s
}

func TestMatchSurfaceVEdgeG2(t *testing.T) {
	target := vPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(j*j) }) // curved in v
	s := vPatch(t, 1, func(i, j int) float64 { return 0 })                       // flat slab above
	got, err := MatchSurface(s, target, VMinEdge, VMaxEdge, 2)
	if err != nil {
		t.Fatalf("MatchSurface V G2: %v", err)
	}
	for _, u := range []float64{0, 0.5, 1} {
		sd := got.SurfaceDersAt(u, 0, 0, 2)
		td := target.SurfaceDersAt(u, 1, 0, 2)
		for k := 0; k <= 2; k++ {
			if !sd[0][k].IsEqualTo(td[0][k], 1e-7) {
				t.Fatalf("V-edge G2 mismatch at u=%g order %d: %v vs %v", u, k, sd[0][k], td[0][k])
			}
		}
	}
}

func TestMatchSurfaceReversedEdgesG2(t *testing.T) {
	// The matched surface is on the LEFT, joining its U-max edge to the target's U-min edge — the
	// reversed configuration that exercises the opposite into-seam direction signs.
	target := uPatch(t, 1, func(i, j int) float64 { return 0.5 * float64(i*i) })
	s := uPatch(t, 0, func(i, j int) float64 { return 0 })
	got, err := MatchSurface(s, target, UMaxEdge, UMinEdge, 2)
	if err != nil {
		t.Fatalf("MatchSurface reversed G2: %v", err)
	}
	for _, v := range []float64{0, 0.5, 1} {
		sd := got.SurfaceDersAt(1, v, 2, 0) // matched surface's U-max derivatives
		td := target.SurfaceDersAt(0, v, 2, 0)
		for k := 0; k <= 2; k++ {
			if !sd[k][0].IsEqualTo(td[k][0], 1e-7) {
				t.Fatalf("reversed G2 mismatch at v=%g order %d: %v vs %v", v, k, sd[k][0], td[k][0])
			}
		}
	}
}

func TestMatchSurfaceValidates(t *testing.T) {
	target := uPatch(t, 0, func(i, j int) float64 { return 0 })
	s := uPatch(t, 1, func(i, j int) float64 { return 0 })
	if _, err := MatchSurface(s, target, UMinEdge, UMaxEdge, 4); err == nil {
		t.Error("order > 3 should error")
	}
	// A degree-1 (2-row) surface cannot match to G2 (needs 3 cross rows).
	line, _ := NewBSplineSurface(1, 3,
		twoRowNet(), twoRowWeights(), []float64{0, 0, 1, 1}, clampedUniformKnots(4, 3))
	if _, err := MatchSurface(line, target, UMinEdge, UMaxEdge, 2); err == nil {
		t.Error("matching to G2 with only 2 cross rows should error")
	}
}

func twoRowNet() [][]math.Point3 {
	net := make([][]math.Point3, 2)
	for i := range net {
		net[i] = make([]math.Point3, 5)
		for j := range net[i] {
			net[i][j] = math.P3(math.Scalar(i), math.Scalar(float64(j)*0.25), 0)
		}
	}
	return net
}

func twoRowWeights() [][]float64 {
	w := make([][]float64, 2)
	for i := range w {
		w[i] = []float64{1, 1, 1, 1, 1}
	}
	return w
}
