// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// cProfileSurface extrudes a C-shaped quadratic profile (in y,z) along x: both arm ends sit at
// y=0 (z=0 and z=2) with the bulge at y=2, so a point left of the opening is near BOTH arms —
// projecting it lands on whichever arm the seed is near. The branch-selection ParamNear gives.
func cProfileSurface(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 2, 1), math.P3(0, 0, 2)},
		{math.P3(2, 0, 0), math.P3(2, 2, 1), math.P3(2, 0, 2)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}}
	s, err := NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func TestParamNearRoundTrip(t *testing.T) {
	s := cProfileSurface(t)
	// A point exactly on the surface, seeded from a nearby (u,v): ParamNear must converge back so
	// PointAt round-trips to it.
	for _, uv := range [][2]float64{{0.5, 0.3}, {0.25, 0.7}, {0.8, 0.15}} {
		want := s.PointAt(uv[0], uv[1])
		u, v := s.ParamNear(want, uv[0]+0.05, uv[1]+0.05)
		if got := s.PointAt(u, v); float64(got.DistanceTo(want)) > 1e-4 {
			t.Errorf("ParamNear round-trip at %v: PointAt=%v want %v (off %.5f)", uv, got, want, got.DistanceTo(want))
		}
	}
}

// taperToApexSurface is a fin that tapers from a wide base (v=0) to a SINGLE point (v=1): every
// control point of the last v row is the same apex, so ∂P/∂u is exactly zero all along v=1 and the
// Gauss-Newton frame there is singular. This is the run-out fillet's cap in miniature (#2020).
func taperToApexSurface(t *testing.T) BSplineSurface {
	t.Helper()
	apex := math.P3(1, 1, 2)
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), apex},
		{math.P3(1, -0.5, 0), apex},
		{math.P3(2, 0, 0), apex},
	}
	w := [][]float64{{1, 1}, {1, 1}, {1, 1}}
	s, err := NewBSplineSurface(2, 1, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// TestParamNearRecoversFromDegenerateApexSeed pins the #2020 fix: a march seeded ON a degenerate
// apex cannot take a Gauss-Newton step, so refineSurfaceParam returns the SEED. Before the fix
// ParamNear handed that stuck seed back as if it were a computed foot; now it falls back to the
// fresh inversion whenever the march never reached a perpendicular foot.
func TestParamNearRecoversFromDegenerateApexSeed(t *testing.T) {
	s := taperToApexSurface(t)
	if du, _ := s.DerivativesAt(0.5, 1); float64(du.Length()) != 0 {
		t.Fatalf("test premise broken: apex row must have |∂P/∂u| == 0, got %g", du.Length())
	}
	// A point that genuinely lies on the surface, near the wide base — far from the apex seed.
	want := s.PointAt(0.25, 0.1)
	u, v := s.ParamNear(want, 0.5, 1) // seeded exactly on the degenerate apex row
	if got := s.PointAt(u, v); float64(got.DistanceTo(want)) > 1e-6 {
		t.Errorf("ParamNear seeded at the degenerate apex returned (%.6g,%.6g) → %v, %.4f away from "+
			"its own target %v; a march that cannot step must not report its seed as the foot", u, v, got, got.DistanceTo(want), want)
	}
}

// TestParamNearKeepsMarchWhenFreshIsNoCloser pins the other half of the #2020 fallback: it may only
// IMPROVE the foot. Projecting from beyond the apex, the apex genuinely IS the nearest point, but
// perpendicularity is unreachable there (the surface ends), so the march stops at a degenerate frame
// with the right answer. The fresh inversion must not be allowed to displace it.
func TestParamNearKeepsMarchWhenFreshIsNoCloser(t *testing.T) {
	s := taperToApexSurface(t)
	apex := s.PointAt(0.5, 1)
	q := math.P3(1, 4, 5) // out beyond the apex, off the surface entirely
	u, v := s.ParamNear(q, 0.5, 1)
	got := s.PointAt(u, v)
	if float64(got.DistanceTo(apex)) > 1e-9 {
		t.Errorf("ParamNear moved off the apex to %v (u,v)=(%.6g,%.6g); the apex is the nearest point, "+
			"so the fallback must keep the march's foot", got, u, v)
	}
	// The contract in general: whatever it returns is never farther from q than the march's own foot.
	if float64(got.DistanceTo(q)) > float64(apex.DistanceTo(q))+1e-9 {
		t.Errorf("fallback returned a WORSE foot: %.6f from q vs the march's %.6f", got.DistanceTo(q), apex.DistanceTo(q))
	}
}

// TestProjectCurveSurvivesDegenerateApex is the failure mode that actually broke the mesher: one
// stalled point poisons the REST of the march, because ProjectCurveToSurface seeds each point from
// the previous one. On the run-out cap this pinned 176 consecutive boundary samples to a single
// (u,v), collapsing the CDT boundary and cracking the face (177 open mesh edges on macOS/arm64).
func TestProjectCurveSurvivesDegenerateApex(t *testing.T) {
	s := taperToApexSurface(t)
	// A polyline that walks up to the apex and back down the far side — it must cross the singular row.
	var pts []math.Point3
	for i := 0; i <= 20; i++ {
		f := float64(i) / 20
		pts = append(pts, s.PointAt(0.5*f, stdmath.Min(2*f, 1)))
	}
	for i := 21; i <= 40; i++ {
		f := float64(i-20) / 20
		pts = append(pts, s.PointAt(0.5+0.5*f, 1-f))
	}
	uv := ProjectCurveToSurface(s, pts)
	distinct := map[[2]float64]bool{}
	for i, p := range uv {
		distinct[[2]float64{float64(p.X), float64(p.Y)}] = true
		if d := s.PointAt(float64(p.X), float64(p.Y)).DistanceTo(pts[i]); float64(d) > 1e-6 {
			t.Fatalf("pcurve sample %d is %.4f off its own 3D point: (u,v)=(%.6g,%.6g) → %v, want %v",
				i, d, p.X, p.Y, s.PointAt(float64(p.X), float64(p.Y)), pts[i])
		}
	}
	// The apex is a legitimate repeat (the polyline touches it once), but a stall pins EVERY
	// subsequent sample to it — so the pcurve must stay almost entirely distinct.
	if distinct[[2]float64{0.5, 1}] && len(distinct) < len(uv)-2 {
		t.Errorf("pcurve collapsed to %d distinct (u,v) for %d samples — a stalled march pinned the tail", len(distinct), len(uv))
	}
}

func TestParamNearSelectsBranchBySeed(t *testing.T) {
	s := cProfileSurface(t)
	// A point left of the C's opening, near both arm ends (v≈0 at z=0, v≈1 at z=2). The seed picks
	// the branch — independent grid-seeded ParamAt cannot.
	q := math.P3(1, -0.5, 1)
	_, vLo := s.ParamNear(q, 0.5, 0.05) // seeded near the z=0 arm
	_, vHi := s.ParamNear(q, 0.5, 0.95) // seeded near the z=2 arm
	if vLo > 0.25 {
		t.Errorf("seed near v=0 should stay on the low arm, got v=%.3f", vLo)
	}
	if vHi < 0.75 {
		t.Errorf("seed near v=1 should stay on the high arm, got v=%.3f", vHi)
	}
	if stdmath.Abs(vHi-vLo) < 0.5 {
		t.Errorf("seed should select distinct branches: vLo=%.3f vHi=%.3f", vLo, vHi)
	}
}
