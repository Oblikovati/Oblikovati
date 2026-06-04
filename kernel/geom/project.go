// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// projectMaxIter and projectTol bound the point-projection iteration: enough steps to
// converge from the ParamAt seed, and a residual tolerance on the perpendicularity
// condition (tangent · (foot − p) ≈ 0).
const (
	projectMaxIter = 32
	projectTol     = 1e-12
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
	for i := 0; i < projectMaxIter; i++ {
		f := s.PointAt(u, v)
		du, dv := s.DerivativesAt(u, v)
		r := f.VectorTo(p) // p − f: the residual we drive perpendicular to the tangents
		gu, gv := float64(du.Dot(r)), float64(dv.Dot(r))
		if stdmath.Abs(gu) < projectTol && stdmath.Abs(gv) < projectTol {
			break
		}
		ddu, ddv, ok := gaussNewtonStep(du, dv, gu, gv)
		if !ok {
			break // degenerate tangent frame (pole/apex): keep the seed
		}
		u, v = clampToSurface(s, u+ddu, v+ddv)
	}
	return u, v, s.PointAt(u, v)
}

// gaussNewtonStep solves the 2×2 normal-equation system [[a,b],[b,c]]·Δ = g for the
// projection step, where a=du·du, b=du·dv, c=dv·dv. ok is false at a degenerate frame.
func gaussNewtonStep(du, dv math.Vector3, gu, gv float64) (ddu, ddv float64, ok bool) {
	a, b, c := float64(du.Dot(du)), float64(du.Dot(dv)), float64(dv.Dot(dv))
	det := a*c - b*b
	if stdmath.Abs(det) < 1e-18 {
		return 0, 0, false
	}
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
