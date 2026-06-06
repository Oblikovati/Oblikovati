// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati/math"
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
	return wrap2pi(stdmath.Atan2(d.Y, d.X)), stdmath.Asin(clampUnit(d.Z))
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

// ParamAt projects q onto the NURBS surface numerically: a coarse grid seed then
// projected Gauss–Newton steps (Piegl & Tiller A6, simplified), staying in domain.
func (s BSplineSurface) ParamAt(q math.Point3) (u, v float64) {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	u, v = surfaceGridSeed(s, q, uLo, uHi, vLo, vHi)
	for i := 0; i < 40; i++ {
		u, v = surfaceProjectStep(s, q, u, v, uLo, uHi, vLo, vHi)
	}
	return u, v
}

// surfaceGridSeed returns the sampled (u, v) over a coarse grid whose point is
// nearest q — the starting guess for the Gauss–Newton refinement.
func surfaceGridSeed(s Surface, q math.Point3, uLo, uHi, vLo, vHi float64) (float64, float64) {
	const n = 16
	bu, bv, bd := uLo, vLo, stdmath.Inf(1)
	for i := 0; i <= n; i++ {
		u := uLo + (uHi-uLo)*float64(i)/n
		for j := 0; j <= n; j++ {
			v := vLo + (vHi-vLo)*float64(j)/n
			if d := s.PointAt(u, v).DistanceSquaredTo(q); d < bd {
				bu, bv, bd = u, v, d
			}
		}
	}
	return bu, bv
}

// surfaceProjectStep advances (u, v) toward q by projecting the residual onto each
// partial derivative, clamped to the domain.
func surfaceProjectStep(s Surface, q math.Point3, u, v, uLo, uHi, vLo, vHi float64) (float64, float64) {
	du, dv := s.DerivativesAt(u, v)
	res := s.PointAt(u, v).VectorTo(q)
	return clampTo(u+projectParam(res, du), uLo, uHi), clampTo(v+projectParam(res, dv), vLo, vHi)
}

// projectParam returns res·d / |d|² — the parameter step reducing the residual
// along d (zero when d is degenerate).
func projectParam(res, d math.Vector3) float64 {
	denom := d.LengthSquared()
	if denom < math.DefaultTolerance {
		return 0
	}
	return res.Dot(d) / denom
}
