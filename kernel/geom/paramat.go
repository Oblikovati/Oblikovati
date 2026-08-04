// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// ParamAt inversions for the analytic surfaces are closed-form: each undoes its
// PointAt by projecting p onto the surface's frame. Angular parameters come back
// in [0, 2π) (wrap2pi) and match the forward radial(u) = cos u·Ref + sin u·binormal
// convention via u = atan2(p·binormal, p·Ref). The NURBS surface has no closed form,
// so it falls back to a coarse grid seed plus projected Gauss–Newton steps.

// ParamAt returns (u, v) = ((p−Origin)·UAxis, (p−Origin)·VAxis).
func (p Plane) ParamAt(q math.Point3) (u, v float64) {
	d := p.Origin.VectorTo(q)
	return d.Dot(p.UAxis.AsVector()), d.Dot(p.VAxis.AsVector())
}

// ParamAt returns the angle around the axis and the signed distance along it.
func (c Cylinder) ParamAt(q math.Point3) (u, v float64) {
	d := c.Origin.VectorTo(q)
	v = d.Dot(c.AxisDir.AsVector())
	r := d.Sub(c.AxisDir.AsVector().Scale(v))
	return wrap2pi(stdmath.Atan2(r.Dot(c.binormal), r.Dot(c.Ref.AsVector()))), v
}

// ParamAt returns the angle around the axis and the distance from the apex along it.
func (c Cone) ParamAt(q math.Point3) (u, v float64) {
	d := c.Apex.VectorTo(q)
	v = d.Dot(c.AxisDir.AsVector())
	r := d.Sub(c.AxisDir.AsVector().Scale(v))
	return wrap2pi(stdmath.Atan2(r.Dot(c.binormal), r.Dot(c.Ref.AsVector()))), v
}

// ParamAt returns the longitude u ∈ [0, 2π) and latitude v ∈ [−π/2, π/2] of the
// closest point on the sphere (the radial direction from the center to q).
func (s Sphere) ParamAt(q math.Point3) (u, v float64) {
	d := unitOrZero(s.Center.VectorTo(q))
	return wrap2pi(stdmath.Atan2(d.Y, d.X)), stdmath.Asin(math.Clamp(d.Z, -1, 1))
}

// ParamAt returns the around-axis angle u and around-tube angle v, both in [0, 2π).
func (t Torus) ParamAt(q math.Point3) (u, v float64) {
	d := t.Center.VectorTo(q)
	axial := d.Dot(t.AxisDir.AsVector())
	planar := d.Sub(t.AxisDir.AsVector().Scale(axial))
	u = wrap2pi(stdmath.Atan2(planar.Dot(t.binormal), planar.Dot(t.Ref.AsVector())))
	v = wrap2pi(stdmath.Atan2(axial, planar.Length()-t.MajorRadius)) // ρ = Minor·cos v, axial = Minor·sin v
	return u, v
}

// surfaceInvertMaxIter / surfaceNearMaxIter cap the Gauss–Newton point inversion: a fresh
// grid-seeded ParamAt may start farther out than a march-seeded ParamNear, so it gets a
// larger budget — but both exit early on convergence (#1401), so the cap is only hit on a
// genuinely degenerate parameterisation.
const (
	surfaceInvertMaxIter = 40
	surfaceNearMaxIter   = 24
)

// ParamAt projects q onto the NURBS surface numerically: a knot-span-aware seed
// (knotSpanSeedParams — one basin per span, #1608) then the shared damped Gauss–Newton
// point inversion (Piegl & Tiller §6.1), with an early convergence exit — and finally a
// retry from the opposite bound when the foot came back pinned on one (retryAcrossPinnedBound).
func (s BSplineSurface) ParamAt(q math.Point3) (u, v float64) {
	us := knotSpanSeedParams(s.UKnots, s.UDegree)
	vs := knotSpanSeedParams(s.VKnots, s.VDegree)
	u, v = nearestSeed(s, q, us, vs)
	u, v, _, _ = refineSurfaceParam(s, q, u, v, surfaceInvertMaxIter)
	return retryAcrossPinnedBound(s, q, u, v)
}

// retryAcrossPinnedBound re-runs the inversion seeded at the OPPOSITE parameter bound whenever the
// foot came back pinned exactly on a bound, and keeps whichever candidate is genuinely closer to q.
//
// On a chart that CLOSES — S(ulo,v) = S(uhi,v), an imported barrel wall (#1510) — the seed lattice
// carries two COINCIDENT seeds, ulo and uhi. nearestSeed's tie-break takes ulo, and from there the
// Gauss–Newton step toward a foot that lies just below uhi points OUT of the domain, so
// lineSearchToward's clamp pins it and the inversion returns ulo: a whole-period error, reported as
// the seam when the foot is on the far side of it. Measured on the #1510 cand_radial barrel: every
// point with u in (0.9805, 1) inverted to u=0, up to 0.584 off its own surface, which collapsed four
// distinct rim samples onto one chart node and folded the covering-space periodic mesh.
//
// Seeded at the far bound the same step points INTO the domain, so it converges. This can only
// improve the answer — the candidates are compared by the very distance ParamAt minimises, and ties
// keep the original — and it costs one extra inversion only for a pinned foot. Deliberately NOT
// applied to ParamNear, whose contract is to stay on the seeded branch.
func retryAcrossPinnedBound(s Surface, q math.Point3, u, v float64) (float64, float64) {
	bestU, bestV := u, v
	bestD := float64(s.PointAt(u, v).DistanceSquaredTo(q))
	for _, seed := range oppositeBoundSeeds(s, u, v) {
		cu, cv, _, _ := refineSurfaceParam(s, q, seed[0], seed[1], surfaceInvertMaxIter)
		if d := float64(s.PointAt(cu, cv).DistanceSquaredTo(q)); d < bestD {
			bestU, bestV, bestD = cu, cv, d
		}
	}
	return bestU, bestV
}

// oppositeBoundSeeds returns the mirrored seed for each finite parameter bound (u, v) is pinned on —
// none when the foot is interior, which is the overwhelmingly common case.
func oppositeBoundSeeds(s Surface, u, v float64) [][2]float64 {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	var out [][2]float64
	if m, ok := mirroredBound(u, uLo, uHi); ok {
		out = append(out, [2]float64{m, v})
	}
	if m, ok := mirroredBound(v, vLo, vHi); ok {
		out = append(out, [2]float64{u, m})
	}
	return out
}

// mirroredBound returns the far bound when x sits exactly on one of a finite [lo, hi] — exact
// equality, because clampFinite is what puts it there.
func mirroredBound(x, lo, hi float64) (float64, bool) {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || lo >= hi {
		return 0, false
	}
	if x == lo {
		return hi, true
	}
	if x == hi {
		return lo, true
	}
	return 0, false
}

// ParamNear projects q onto the NURBS surface starting from the seed (u0, v0) instead of a fresh
// grid search. Marching along an edge with each point seeded from the previous one keeps the (u,v)
// on a single smooth branch — a pcurve free of the jitter independent ParamAt calls produce where
// the surface nearly folds (different points snapping to different local minima). It is the
// foundation for the tolerant NURBS mesher (ADR-0030, M24 F01): a smooth, non-self-intersecting
// boundary is the prerequisite for a non-folding interior triangulation and a reliable
// point-in-trim test.
//
// A march that died at a DEGENERATE FRAME is not on a branch — it is the seed handed in, untouched —
// so it falls back to the fresh inversion (preferFreshInversion). This does not weaken the
// stay-on-one-branch contract above, and is deliberately narrower than "did not converge": a march
// that stopped at a numerical floor DID find a local minimum of the distance and keeps it, because
// re-inverting those perturbs a good foot for nothing (measured across the OCCT blend corpus: 2666
// floor-stops on three cases, whose fresh inversions differed by at most 2e-4 and moved three pinned
// meshes). Only the degenerate-frame exit returns an uncomputed seed, and only it propagates.
func (s BSplineSurface) ParamNear(q math.Point3, u0, v0 float64) (u, v float64) {
	u, v = clampToSurface(s, u0, v0)
	u, v, _, degenerate := refineSurfaceParam(s, q, u, v, surfaceNearMaxIter)
	if !degenerate {
		return u, v
	}
	return preferFreshInversion(s, q, u, v)
}

// preferFreshInversion returns whichever of the stalled march foot (u,v) and a fresh grid-seeded
// ParamAt inversion actually lands nearer q. Like retryAcrossPinnedBound it can only improve the
// answer — the two candidates are compared by the very distance the inversion minimises, and a tie
// keeps the march — so it needs no tolerance and cannot depend on rounding.
//
// It exists because a seeded march that dies at a degenerate frame returns its SEED, and
// ProjectCurveToSurface then feeds that seed to the next point, which dies the same way: one stall
// pins the entire remaining pcurve to one parameter value. Measured on TestFilletRunOutToZero's
// run-out cap (macOS/arm64, #2020): the taper's apex has ∂P/∂u == 0 exactly, so 176 consecutive
// boundary samples spanning 2.01 in 3D all inverted to the same (u,v) corner — 175 of the loop's 360
// samples up to 2.01 off their own surface, collapsing the CDT's boundary from 360 edges to 185 and
// cracking the face against its neighbours (177 open mesh edges). A fresh inversion on those same
// points lands them to 1.7e-18.
func preferFreshInversion(s BSplineSurface, q math.Point3, u, v float64) (float64, float64) {
	fu, fv := s.ParamAt(q)
	if s.PointAt(fu, fv).DistanceSquaredTo(q) < s.PointAt(u, v).DistanceSquaredTo(q) {
		return fu, fv
	}
	return u, v
}
