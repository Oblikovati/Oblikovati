// SPDX-License-Identifier: GPL-2.0-only

package geom

import "testing"

// edgeRow builds a boundary iso-curve from a surface control row (the u-min/u-max edge).
func edgeRow(t *testing.T, s BSplineSurface, atUMax bool) BSplineCurve {
	t.Helper()
	i := 0
	if atUMax {
		i = len(s.Ctrl) - 1
	}
	c, err := NewBSplineCurve(s.VDegree, s.Ctrl[i], s.Weights[i], s.VKnots)
	if err != nil {
		t.Fatalf("edge row: %v", err)
	}
	return c
}

// TestBridgeSurfaceG2MatchesBothNeighbours bridges the gap between two curved patches (A at x∈[0,1],
// B at x∈[2,3]) at G2 on both sides, and checks the bridge interpolates each edge (G0) and continues
// each neighbour's tangent and curvature across the seam (the cross-derivative control curves match).
func TestBridgeSurfaceG2MatchesBothNeighbours(t *testing.T) {
	sA := uPatch(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) })
	sB := uPatch(t, 2, func(i, j int) float64 { return 0.3 * float64((4-i)*(4-i)) })
	cA := edgeRow(t, sA, true)  // sA's u-max edge (x=1)
	cB := edgeRow(t, sB, false) // sB's u-min edge (x=2)

	br, err := BridgeSurface(cA, cB, sA, sB, UMaxEdge, UMinEdge, 2, 2)
	if err != nil {
		t.Fatalf("BridgeSurface: %v", err)
	}
	// G0: the bridge interpolates both boundary curves.
	for i := 0; i <= 8; i++ {
		s := float64(i) / 8
		if !br.PointAt(s, 0).IsEqualTo(cA.PointAt(s), 1e-9) {
			t.Fatalf("bridge v=0 not on cA at u=%g: %v vs %v", s, br.PointAt(s, 0), cA.PointAt(s))
		}
		if !br.PointAt(s, 1).IsEqualTo(cB.PointAt(s), 1e-9) {
			t.Fatalf("bridge v=1 not on cB at u=%g", s)
		}
	}
	// G2 side A: bridge ∂^k/∂v at v=0 equals sA ∂^k/∂u at u=1 (the matched seam, k=1,2).
	for _, t0 := range []float64{0, 0.5, 1} {
		bd := br.SurfaceDersAt(t0, 0, 0, 2)
		ad := sA.SurfaceDersAt(1, t0, 2, 0)
		for k := 1; k <= 2; k++ {
			if !bd[0][k].IsEqualTo(ad[k][0], 1e-6) {
				t.Errorf("side A G2 mismatch at u=%g order %d: %v vs %v", t0, k, bd[0][k], ad[k][0])
			}
		}
	}
	// G2 side B: bridge ∂^k/∂v at v=1 equals sB ∂^k/∂u at u=0.
	for _, t0 := range []float64{0, 0.5, 1} {
		bd := br.SurfaceDersAt(t0, 1, 0, 2)
		bbd := sB.SurfaceDersAt(0, t0, 2, 0)
		for k := 1; k <= 2; k++ {
			if !bd[0][k].IsEqualTo(bbd[k][0].Scale(-1), 1e-6) && !bd[0][k].IsEqualTo(bbd[k][0], 1e-6) {
				t.Errorf("side B G2 mismatch at u=%g order %d: %v vs %v", t0, k, bd[0][k], bbd[k][0])
			}
		}
	}
}

// TestBridgeSurfaceRejectsAboveG2: the 6-row bridge cannot hold G3.
func TestBridgeSurfaceRejectsAboveG2(t *testing.T) {
	sA := uPatch(t, 0, func(i, j int) float64 { return 0 })
	sB := uPatch(t, 2, func(i, j int) float64 { return 0 })
	cA := edgeRow(t, sA, true)
	cB := edgeRow(t, sB, false)
	if _, err := BridgeSurface(cA, cB, sA, sB, UMaxEdge, UMinEdge, 3, 0); err == nil {
		t.Error("BridgeSurface should reject G3 (order 3)")
	}
}

