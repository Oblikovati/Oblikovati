// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// rippledSheet builds a degree-3 B-spline sheet over uniform open knots with uSpans × vSpans
// knot spans and one bump per span: control z alternates ±amplitude per control column (and
// per row when eggcrate is true). It is the high-span fixture of Oblikovati#1608 — imported
// STEP surfaces routinely carry dozens of spans, far more than a fixed 16×16 seed grid can see.
func rippledSheet(t *testing.T, uSpans, vSpans int, amplitude float64, eggcrate bool) BSplineSurface {
	t.Helper()
	ctrl, weights := rippledNet(uSpans+3, vSpans+3, amplitude, eggcrate)
	s, err := NewBSplineSurface(3, 3, ctrl, weights, uniformOpenKnots(uSpans, 3), uniformOpenKnots(vSpans, 3))
	if err != nil {
		t.Fatalf("rippledSheet(%d×%d spans): %v", uSpans, vSpans, err)
	}
	return s
}

// rippledNet lays the control points on an integer grid with alternating-sign z.
func rippledNet(nu, nv int, amplitude float64, eggcrate bool) (ctrl [][]math.Point3, weights [][]float64) {
	ctrl = make([][]math.Point3, nu)
	weights = make([][]float64, nu)
	for i := range nu {
		ctrl[i] = make([]math.Point3, nv)
		weights[i] = make([]float64, nv)
		for j := range nv {
			ctrl[i][j] = math.P3(float64(i), float64(j), rippleHeight(i, j, amplitude, eggcrate))
			weights[i][j] = 1
		}
	}
	return ctrl, weights
}

// rippleHeight alternates the bump sign per column (u ripple) or per diagonal (eggcrate).
func rippleHeight(i, j int, amplitude float64, eggcrate bool) float64 {
	parity := i
	if eggcrate {
		parity = i + j
	}
	if parity%2 == 0 {
		return amplitude
	}
	return -amplitude
}

// uniformOpenKnots returns the clamped knot vector with `spans` uniform interior spans on
// [0, 1] for the given degree.
func uniformOpenKnots(spans, degree int) []float64 {
	knots := make([]float64, 0, spans+2*degree+1)
	for k := 0; k <= degree; k++ {
		knots = append(knots, 0)
	}
	for k := 1; k < spans; k++ {
		knots = append(knots, float64(k)/float64(spans))
	}
	for k := 0; k <= degree; k++ {
		knots = append(knots, 1)
	}
	return knots
}

// TestParamAtHighSpanSurfaceInversion is the #1608 seeding-completeness regression: on a
// 33×33-span eggcrate — one bump per knot span — a fixed 16×16 seed grid skips whole spans,
// the Gauss–Newton polish converges to the wrong bump, and ParamAt returns a foot far from
// an ON-SURFACE query point. Span-aware seeding must recover every sampled point: the
// surface is a height field (injective in xy), so the zero-distance foot is unique.
func TestParamAtHighSpanSurfaceInversion(t *testing.T) {
	t.Parallel()
	s := rippledSheet(t, 33, 33, 8.0, true)
	res := surfaceNetResolution(s)
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	rng := rand.New(rand.NewSource(1)) // deterministic: repeatable failures
	failures := 0
	for range 100 {
		u := uLo + (uHi-uLo)*rng.Float64()
		v := vLo + (vHi-vLo)*rng.Float64()
		p := s.PointAt(u, v)
		ru, rv := s.ParamAt(p)
		if d := s.PointAt(ru, rv).DistanceTo(p); float64(d) > res.Sew() {
			failures++
			t.Errorf("ParamAt foot off by %g (> %g) for on-surface point at (u,v)=(%.4f,%.4f)", d, res.Sew(), u, v)
		}
	}
	if failures > 0 {
		t.Logf("%d/100 on-surface points inverted to the wrong span", failures)
	}
}

// surfaceNetResolution derives the model-relative Resolution from the control net extent.
func surfaceNetResolution(s BSplineSurface) Resolution {
	var pts []math.Point3
	for _, row := range s.Ctrl {
		pts = append(pts, row...)
	}
	return ResolutionForPoints(pts)
}
