// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// closedBarrelSurface builds a degree-3, clamped B-spline closed in u and open in v: cols control
// columns around a circle whose LAST column repeats the first, so S(0,v) = S(1,v) exactly. That is
// the structure #1510's imported barrel wall actually has (degree 3, 30 clamped columns, 29 uniform
// spans, first column == last), and its last knot span is where the inversion defect lived.
func closedBarrelSurface(t *testing.T, radius, height float64, cols int) BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, cols)
	weights := make([][]float64, cols)
	for c := range cols {
		ang := 2 * stdmath.Pi * float64(c) / float64(cols-1) // c = cols−1 lands back on c = 0
		x, y := radius*stdmath.Cos(ang), radius*stdmath.Sin(ang)
		ctrl[c] = []math.Point3{math.P3(x, y, 0), math.P3(x, y, height)}
		weights[c] = []float64{1, 1}
	}
	ctrl[cols-1] = ctrl[0] // bit-exact closure: cos/sin(2π) is only 1e-16 close, and the defect needs an exact seed tie
	s, err := NewBSplineSurface(3, 1, ctrl, weights, clampedUniformKnots(cols-1, 3), clampedUniformKnots(1, 1))
	if err != nil {
		t.Fatalf("closed barrel (%d cols): %v", cols, err)
	}
	if s.PointAt(0, 0.5) != s.PointAt(1, 0.5) {
		t.Fatalf("fixture must close BIT-EXACTLY in u, or the coincident seed pair the defect needs never forms")
	}
	return s
}

// roundTripInversionError is the measuring function used throughout this file: take an exact
// surface point, invert it, and report how far the inverted parameters land from that same point in
// 3D. It has an exact target — zero — because the query point is on the surface by construction.
func roundTripInversionError(s BSplineSurface, u, v float64) float64 {
	q := s.PointAt(u, v)
	iu, iv := s.ParamAt(q)
	return float64(q.DistanceTo(s.PointAt(iu, iv)))
}

// TestParamAtInvertsAcrossTheClosingSeam is the regression for the covering-mesh chart collapse: on a
// chart that CLOSES in u, ulo and uhi seed the SAME 3D point, nearestSeed's tie-break takes ulo, and
// the Gauss-Newton step toward a foot just below uhi then points out of the domain and is clamped —
// so every point in the last span inverted to u = ulo, a full period away from its own foot.
// Measured on #1510's cand_radial barrel before the fix: u ∈ (0.9805, 1) all returned 0, up to 0.584
// off the surface, which collapsed four distinct rim samples onto ONE chart node and folded the
// covering-space periodic mesh (3 fold edges, 8 unpaired boundary edges).
//
// It sweeps the WHOLE domain, not just the tail, so the fix is pinned to leave the interior alone.
func TestParamAtInvertsAcrossTheClosingSeam(t *testing.T) {
	s := closedBarrelSurface(t, 5.9, 18, 30)
	const steps = 257
	worstU, worst := 0.0, 0.0
	for i := range steps {
		u := float64(i) / float64(steps-1)
		if e := roundTripInversionError(s, u, 0.5); e > worst {
			worstU, worst = u, e
		}
	}
	// The barrel's own discretization step at property quality is ~0.146; anything near that is a
	// whole chart node landing on the wrong branch, not a convergence residual.
	if worst > 1e-6 {
		t.Errorf("ParamAt round-trip is %.6g off the surface at u=%.6g (want <1e-6): a closing-seam "+
			"inversion landed on the far branch", worst, worstU)
	}
}

// TestParamAtKeepsDistinctSeamSamplesDistinct pins the CONSEQUENCE the mesher depends on: consecutive
// samples that are distinct in 3D must stay distinct in the chart. The defect collapsed a run of four
// onto one node, and a zero-length constraint segment is what folds the covering CDT.
func TestParamAtKeepsDistinctSeamSamplesDistinct(t *testing.T) {
	s := closedBarrelSurface(t, 5.9, 18, 30)
	const n = 256
	prevU, prevV := s.ParamAt(s.PointAt(0, 0))
	collapsed := 0
	for i := 1; i < n; i++ {
		u, v := s.ParamAt(s.PointAt(float64(i)/n, 0))
		if u == prevU && v == prevV {
			collapsed++
		}
		prevU, prevV = u, v
	}
	if collapsed != 0 {
		t.Errorf("%d of %d distinct rim samples inverted to a DUPLICATE chart node; want 0", collapsed, n-1)
	}
}

// TestParamAtLeavesAnOpenChartPinnedAtItsBound is the FALSE-POSITIVE direction. The retry must not
// teleport a foot across the domain on a chart that does NOT close: a query beyond the u=1 edge of an
// open sheet belongs at u=1, and answering u=0 would be exactly the error the retry exists to remove,
// mirrored. It also pins that an interior foot is untouched.
func TestParamAtLeavesAnOpenChartPinnedAtItsBound(t *testing.T) {
	s := rippledSheet(t, 6, 6, 0.2, false)
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	edge := s.PointAt(uHi, (vLo+vHi)/2)
	beyond := math.P3(float64(edge.X)+3, float64(edge.Y), float64(edge.Z))
	u, _ := s.ParamAt(beyond)
	if u != uHi {
		t.Errorf("a point past the open sheet's u=%.6g edge inverted to u=%.6g; want the near bound", uHi, u)
	}
	mid := (uLo + uHi) / 2
	if e := roundTripInversionError(s, mid, (vLo+vHi)/2); e > 1e-9 {
		t.Errorf("interior round-trip moved by %.3g; the retry must not disturb an unpinned foot", e)
	}
}

// TestParamAtAtTheSeamItselfStaysCanonical pins the tie: S(ulo,v) and S(uhi,v) are the same point, so
// either parameter is a correct answer — but the answer must be STABLE, or a rim's development would
// jitter between branches from run to run. The retry compares by distance and keeps the incumbent on
// a tie, so the seam keeps reporting ulo.
func TestParamAtAtTheSeamItselfStaysCanonical(t *testing.T) {
	s := closedBarrelSurface(t, 5.9, 18, 30)
	uLo, _ := s.UDomain()
	for _, v := range []float64{0, 0.25, 0.5, 1} {
		if u, _ := s.ParamAt(s.PointAt(uLo, v)); u != uLo {
			t.Errorf("the seam point at v=%.2f inverted to u=%.17g; want the canonical %.17g", v, u, uLo)
		}
	}
}

// TestMirroredBoundOnlyFiresOnAFiniteBound pins the trigger itself: interior parameters and infinite
// bounds must not spawn a retry (the cost is one extra inversion, so it must stay rare).
func TestMirroredBoundOnlyFiresOnAFiniteBound(t *testing.T) {
	cases := []struct {
		x, lo, hi float64
		want      float64
		wantOK    bool
	}{
		{0, 0, 1, 1, true},
		{1, 0, 1, 0, true},
		{0.5, 0, 1, 0, false},
		{0, stdmath.Inf(-1), stdmath.Inf(1), 0, false},
		{0, 0, 0, 0, false},
	}
	for _, c := range cases {
		got, ok := mirroredBound(c.x, c.lo, c.hi)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("mirroredBound(%v, %v, %v) = (%v, %v); want (%v, %v)", c.x, c.lo, c.hi, got, ok, c.want, c.wantOK)
		}
	}
}
