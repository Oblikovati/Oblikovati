// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// skewedPatch is a biquadratic NURBS surface with a sheared, twisted control net, so its
// partials Su, Sv are strongly non-orthogonal (Su·Sv ≠ 0) and the surface curves — exactly
// the parameterisation on which the old per-axis projection (which ignores the Su·Sv
// cross-term) crawls.
func skewedPatch(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0.5}, {X: 2, Y: 2, Z: 0}},
		{{X: 1, Y: 0, Z: 1}, {X: 2, Y: 1, Z: 1.5}, {X: 3, Y: 2, Z: 1}},
		{{X: 2, Y: 0, Z: 0}, {X: 3, Y: 1, Z: 0.5}, {X: 4, Y: 2, Z: 0}},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("skewed patch: %v", err)
	}
	return s
}

// perpCosine returns the larger of the two tangent–residual cosines at (u,v) — the
// scale-invariant perpendicularity measure a converged point inversion drives to zero
// (0 = foot is perpendicular).
func perpCosine(s Surface, q math.Point3, u, v float64) float64 {
	du, dv := s.DerivativesAt(u, v)
	r := s.PointAt(u, v).VectorTo(q)
	rLen := float64(r.Length())
	if rLen < 1e-12 {
		return 0 // q on the surface
	}
	cu := stdmath.Abs(float64(du.Dot(r))) / (float64(du.Length()) * rLen)
	cv := stdmath.Abs(float64(dv.Dot(r))) / (float64(dv.Length()) * rLen)
	return stdmath.Max(cu, cv)
}

// perAxisStep is the OLD point-inversion step (res·d/|d|² per axis, ignoring the Su·Sv
// cross-term), reconstructed here to show Gauss–Newton converges where it stalls.
func perAxisStep(s Surface, q math.Point3, u, v float64) (float64, float64) {
	du, dv := s.DerivativesAt(u, v)
	r := s.PointAt(u, v).VectorTo(q)
	step := func(d math.Vector3) float64 {
		if dd := float64(d.LengthSquared()); dd > 1e-12 {
			return float64(d.Dot(r)) / dd
		}
		return 0
	}
	return clampFinite(u+step(du), 0, 1), clampFinite(v+step(dv), 0, 1)
}

// TestGaussNewtonConvergesWherePerAxisStalls is acceptance criterion 2 of #1401: on a
// skewed patch, the shared damped Gauss–Newton inversion reaches the perpendicularity
// tolerance in a small fixed budget where the per-axis projection does not.
func TestGaussNewtonConvergesWherePerAxisStalls(t *testing.T) {
	t.Parallel()
	s := skewedPatch(t)
	// A target above the patch: the surface point at (0.3, 0.7) pushed out along the normal.
	foot := s.PointAt(0.3, 0.7)
	q := foot.TranslateBy(s.NormalAt(0.3, 0.7).Scale(0.3))

	const budget = 40
	gu, gv, iters, _ := refineSurfaceParam(s, q, 0.5, 0.5, budget)
	gnCos := perpCosine(s, q, gu, gv)
	if gnCos > 1e-7 {
		t.Errorf("Gauss–Newton did not converge: residual %g", gnCos)
	}
	if iters >= budget {
		t.Errorf("Gauss–Newton used the whole budget (%d) — it should stop early", iters)
	}

	// The old per-axis projection, given the SAME budget from the same seed, lands nowhere
	// near perpendicular on this skewed patch — the concrete contrast motivating #1401.
	pu, pv := 0.5, 0.5
	for range budget {
		pu, pv = perAxisStep(s, q, pu, pv)
	}
	perAxisCos := perpCosine(s, q, pu, pv)
	if perAxisCos <= 1e-7 {
		t.Skipf("per-axis also converged (residual %g) — skew too mild to contrast", perAxisCos)
	}
	t.Logf("after %d iters: Gauss–Newton cosine %.2e (converged in %d) vs per-axis %.2e", budget, gnCos, iters, perAxisCos)
}

// TestParamAtConvergesAndStopsEarly is acceptance criterion 3 of #1401: ParamAt on a normal
// patch lands the foot perpendicular AND the refinement exits before the iteration cap
// (rather than always running 40).
func TestParamAtConvergesAndStopsEarly(t *testing.T) {
	t.Parallel()
	s := skewedPatch(t)
	foot := s.PointAt(0.6, 0.25)
	q := foot.TranslateBy(s.NormalAt(0.6, 0.25).Scale(0.3))

	u, v := s.ParamAt(q)
	if r := perpCosine(s, q, u, v); r > 1e-7 {
		t.Errorf("ParamAt foot not perpendicular: residual %g", r)
	}
	// Re-run the refinement from ParamAt's knot-span seed to observe the iteration count.
	sus, svs, spts := s.seedLattice()
	su, sv := nearestSeedPoint(sus, svs, spts, q)
	_, _, iters, _ := refineSurfaceParam(s, q, su, sv, surfaceInvertMaxIter)
	if iters >= surfaceInvertMaxIter {
		t.Errorf("ParamAt refinement ran the full %d iterations — early exit not working", surfaceInvertMaxIter)
	}
}

// TestParamNearRecoversParameters checks ParamNear (the mesher/SSI seed-march entry point)
// recovers the parameters of a known surface point from a nearby seed, perpendicular.
func TestParamNearRecoversParameters(t *testing.T) {
	t.Parallel()
	s := skewedPatch(t)
	want := s.PointAt(0.42, 0.58)
	u, v := s.ParamNear(want, 0.4, 0.6)
	if d := s.PointAt(u, v).DistanceTo(want); float64(d) > 1e-7 {
		t.Errorf("ParamNear foot %v is %g from the target, want coincident", s.PointAt(u, v), d)
	}
	if r := perpCosine(s, want, u, v); r > 1e-7 {
		t.Errorf("ParamNear foot not perpendicular: residual %g", r)
	}
}
