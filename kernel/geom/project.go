// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// projectMaxIter bounds the point-projection iteration (enough steps to converge from the
// ParamAt seed); projectTol is the point-coincidence tolerance (|foot − p| ≈ 0, q on the
// surface); zeroCosTol is the RELATIVE zero-cosine tolerance — |Sᵤ·r|/(|Sᵤ||r|), the cosine
// of the angle from perpendicular — the scale-invariant convergence test (Piegl & Tiller
// §6.1), so a tiny or huge patch stops at the same geometric accuracy (#1401).
const (
	projectMaxIter = 32
	projectTol     = 1e-12
	zeroCosTol     = 1e-10
)

// ClosestPointOnSurface returns the parameters (u, v) and the foot point on s nearest to
// p — the foot of the perpendicular from p. It seeds from the surface's ParamAt inverse
// (exact for plane/cylinder/sphere) and refines with clamped Gauss–Newton steps (Piegl &
// Tiller A6), so the foot is perpendicular for every surface including the cone and torus
// where ParamAt is only approximate.
//
// Example — projecting a point onto a sphere lands it on the sphere at radius R:
//
//	s, _ := geom.NewSphere(math.P3(0,0,0), 5)
//	_, _, foot := geom.ClosestPointOnSurface(s, math.P3(10, 0, 0)) // foot ≈ (5,0,0)
func ClosestPointOnSurface(s Surface, p math.Point3) (u, v float64, foot math.Point3) {
	u, v = s.ParamAt(p)
	u, v = clampToSurface(s, u, v)
	u, v, _, _ = refineSurfaceParam(s, p, u, v, projectMaxIter)
	return u, v, s.PointAt(u, v)
}

// refineSurfaceParam drives (u, v) to the foot of the perpendicular from q on s by damped
// 2×2 Gauss–Newton on the first-fundamental-form system [[Su·Su, Su·Sv],[Su·Sv, Sv·Sv]]
// (Newton point inversion, Piegl & Tiller §6.1), stopping EARLY once the perpendicularity
// residual (Su·r, Sv·r) is within projectTol. It is the one point-inversion every surface
// shares — analytic [ClosestPointOnSurface] and the NURBS ParamAt/ParamNear and offset hot
// paths (Oblikovati/Oblikovati#1401). The per-axis projection it replaced ignored the Su·Sv
// cross-term and never tested convergence, so it crawled on a skewed/folding
// parameterisation and burned every iteration even after converging. Returns the refined
// (u, v), the iterations actually run (< maxIter once converged), and whether it gave up at a
// DEGENERATE FRAME.
//
// That last return distinguishes the loop's two failure exits, which are not equivalent. At a
// numerical floor (!moved) the line search tried and found nothing better: the foot is a genuine
// local minimum of the distance, so returning it is an answer. At a degenerate frame (!ok) no step
// direction exists at all, so the loop returns the SEED IT WAS HANDED, untouched — not a computed
// foot. Only the latter can propagate: a caller that marches (see [BSplineSurface.ParamNear])
// re-seeds the next point from this result, so one degenerate frame pins every subsequent point to
// the same parameter (#2020).
func refineSurfaceParam(s Surface, q math.Point3, u, v float64, maxIter int) (float64, float64, int, bool) {
	i := 0
	for ; i < maxIter; i++ {
		f := s.PointAt(u, v)
		du, dv := s.DerivativesAt(u, v)
		r := f.VectorTo(q) // q − f: the residual we drive perpendicular to the tangents
		gu, gv := float64(du.Dot(r)), float64(dv.Dot(r))
		if surfaceFootConverged(du, dv, r, gu, gv) {
			break
		}
		ddu, ddv, ok := gaussNewtonStep(du, dv, gu, gv)
		if !ok {
			return u, v, i, true // degenerate frame (pole/apex): no step exists — this is the SEED, not a foot
		}
		nu, nv, moved := lineSearchToward(s, q, u, v, ddu, ddv, float64(r.LengthSquared()))
		if !moved {
			break // no step reduces the distance — a local minimum / numerical floor
		}
		u, v = nu, nv
	}
	return u, v, i, false
}

// surfaceFootConverged applies the Piegl & Tiller §6.1 stopping tests: point coincidence
// (q lies on the surface, |r| ≈ 0) OR zero cosine (r is perpendicular to both tangents,
// relative to their magnitudes). The relative cosine test is scale-invariant, unlike an
// absolute residual threshold.
func surfaceFootConverged(du, dv math.Vector3, r math.Vector3, gu, gv float64) bool {
	rLen := float64(r.Length())
	if rLen < projectTol {
		return true // q is on the surface
	}
	duLen, dvLen := float64(du.Length()), float64(dv.Length())
	cosU := tangentCosine(gu, duLen, rLen)
	cosV := tangentCosine(gv, dvLen, rLen)
	return cosU < zeroCosTol && cosV < zeroCosTol
}

// tangentCosine returns |tangent·r|/(|tangent||r|) — the cosine of the angle between a
// tangent and the residual, 0 when perpendicular. A degenerate tangent reports 0 (it
// imposes no perpendicularity condition).
func tangentCosine(g, tangentLen, rLen float64) float64 {
	if tangentLen < projectTol {
		return 0
	}
	return stdmath.Abs(g) / (tangentLen * rLen)
}

// lineSearchToward backtracks the Gauss–Newton step (ddu, ddv) until it actually reduces
// the squared distance to q (current d2), halving the step a few times. Backtracking on the
// true objective — not the raw Newton step — keeps the inversion from overshooting and
// diverging on a strongly curved or skewed patch far from the foot (the damping/line-search
// the robust point inversion needs, Hu & Wallner; #1401). moved is false when even a tiny
// step fails to improve.
func lineSearchToward(s Surface, q math.Point3, u, v, ddu, ddv, d2 float64) (nu, nv float64, moved bool) {
	alpha := 1.0
	for range 8 {
		cu, cv := clampToSurface(s, u+alpha*ddu, v+alpha*ddv)
		if float64(s.PointAt(cu, cv).VectorTo(q).LengthSquared()) < d2 {
			return cu, cv, true
		}
		alpha *= 0.5
	}
	return u, v, false
}

// gaussNewtonStep solves the 2×2 normal-equation system [[a,b],[b,c]]·Δ = g for the
// projection step, where a=du·du, b=du·dv, c=dv·dv. ok is false at a degenerate
// frame, judged by the SCALE-INVARIANT [degenerateFirstForm] (parallel/collapsed
// tangents) rather than an absolute cutoff on the length⁴ determinant, which would
// reject a valid small-scale patch and accept a singular huge-scale one (#1402).
func gaussNewtonStep(du, dv math.Vector3, gu, gv float64) (ddu, ddv float64, ok bool) {
	a, b, c := float64(du.Dot(du)), float64(du.Dot(dv)), float64(dv.Dot(dv))
	if degenerateFirstForm(a, b, c) {
		return 0, 0, false
	}
	det := a*c - b*b
	return (c*gu - b*gv) / det, (a*gv - b*gu) / det, true
}

// clampToSurface pins (u, v) into the surface's finite parameter bounds (an unbounded or
// periodic direction — ±Inf — is left as-is; ParamAt already wraps periodic angles).
func clampToSurface(s Surface, u, v float64) (float64, float64) {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	return clampFinite(u, uLo, uHi), clampFinite(v, vLo, vHi)
}

// clampFinite clamps x to [lo, hi], ignoring any infinite bound.
func clampFinite(x, lo, hi float64) float64 {
	if !stdmath.IsInf(lo, 0) && x < lo {
		return lo
	}
	if !stdmath.IsInf(hi, 0) && x > hi {
		return hi
	}
	return x
}

// SignedDistanceToSurface returns the signed distance from p to s along the surface
// normal at the foot of the perpendicular (positive on the normal's side). It is the
// scalar field whose zero set is the surface — the integrand the curve-intersection
// tracer brackets.
func SignedDistanceToSurface(s Surface, p math.Point3) float64 {
	u, v, foot := ClosestPointOnSurface(s, p)
	n := s.NormalAt(u, v)
	return float64(foot.VectorTo(p).Dot(n))
}
