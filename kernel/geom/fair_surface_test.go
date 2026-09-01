// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// noisyPatch builds an 8×8 bicubic patch over [0,1]² with z jittered by a deterministic pseudo-random
// wrinkle on the interior — a surface with curvature noise to fair out.
func noisyPatch(t *testing.T) BSplineSurface {
	t.Helper()
	const n = 8
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			z := 0.0
			if i > 0 && i < n-1 && j > 0 && j < n-1 { // jitter interior only (boundary stays flat)
				z = 0.2 * float64((i*7+j*13)%5-2) // deterministic ±wrinkle
			}
			ctrl[i][j] = math.P3(math.Scalar(float64(i)/7), math.Scalar(float64(j)/7), math.Scalar(z))
			w[i][j] = 1
		}
	}
	k := clampedUniformKnots(n-1, 3)
	s, err := NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("noisy patch: %v", err)
	}
	return s
}

// meanCurvVariance is the variance of the mean curvature (kMax+kMin)/2 over an interior sample grid.
func meanCurvVariance(s BSplineSurface) float64 {
	var vals []float64
	for i := 1; i < 8; i++ {
		for j := 1; j < 8; j++ {
			_, kMax, kMin := SurfaceCurvatures(s, float64(i)/8, float64(j)/8)
			vals = append(vals, (kMax+kMin)/2)
		}
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	varc := 0.0
	for _, v := range vals {
		varc += (v - mean) * (v - mean)
	}
	return varc / float64(len(vals))
}

// TestFairSurfaceReducesCurvatureVariance: fairing smooths the wrinkle, cutting the mean-curvature
// variance, while the held boundary band (G2: 3 rows each edge) is preserved exactly.
func TestFairSurfaceReducesCurvatureVariance(t *testing.T) {
	t.Parallel()
	s := noisyPatch(t)
	before := meanCurvVariance(s)
	faired := FairSurface(s, 2, 0.5, 40)
	after := meanCurvVariance(faired)
	if after >= before {
		t.Errorf("fairing did not reduce curvature variance: before %g, after %g", before, after)
	}
	// The G2 held band (rows/cols 0–2 and 5–7) is unchanged, so boundary continuity is preserved.
	for i := range 8 {
		for j := range 8 {
			held := i < 3 || i >= 5 || j < 3 || j >= 5
			if held && !faired.Ctrl[i][j].IsEqualTo(s.Ctrl[i][j], 1e-12) {
				t.Errorf("held band CV (%d,%d) moved: %v vs %v", i, j, faired.Ctrl[i][j], s.Ctrl[i][j])
			}
		}
	}
}

// TestFairSurfaceHoldsBoundaryDerivatives: a G1 fairing leaves the surface's boundary position and
// tangent (cross-derivative) unchanged at the edges.
func TestFairSurfaceHoldsBoundaryDerivatives(t *testing.T) {
	t.Parallel()
	s := noisyPatch(t)
	faired := FairSurface(s, 1, 0.5, 20)
	for _, v := range []float64{0, 0.5, 1} {
		for k := 0; k <= 1; k++ {
			if !faired.SurfaceDersAt(0, v, k, 0)[k][0].IsEqualTo(s.SurfaceDersAt(0, v, k, 0)[k][0], 1e-9) {
				t.Errorf("G1 fairing changed u=0 boundary derivative order %d at v=%g", k, v)
			}
		}
	}
}

// TestFairSurfaceNoInteriorIsNoop: a patch too small for the held band is returned unchanged.
func TestFairSurfaceNoInteriorIsNoop(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.1 * float64(i) }) // 5×5
	if faired := FairSurface(s, 2, 0.5, 10); !faired.Ctrl[2][2].IsEqualTo(s.Ctrl[2][2], 1e-12) {
		t.Error("a 5×5 patch has no interior past a G2 band; fairing should be a no-op")
	}
}
