// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// SpiricArc is a bounded segment of the SPIRIC OF PERSEUS — the quartic curve a plane NOT
// perpendicular to a torus axis cuts from the torus (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375).
// On the torus surface P(u,v) (u around the axis, v around the tube) the cut plane is the level set
//
//	g(u,v) = (R + r·cos v)·M·cos(u − Phi) + C·r·sin v − K = 0
//
// where the coefficients come from the plane's unit normal m and origin o relative to the torus:
// C = m·axis, M = |m − C·axis| (the in-plane magnitude), Phi = atan2(m·binormal, m·Ref) (the azimuth
// the in-plane normal points), K = m·(o − Center). Solving g=0 for u gives the two roots
//
//	u(v) = Phi ± arccos( w(v) ),  w(v) = (K − C·r·sin v) / (M·(R + r·cos v))
//
// so the section is single-valued in u on each Branch (+1 / −1). A SpiricArc fixes the torus, the
// coefficients, the branch, and a tube-angle range [V0, V1]; the parameter t∈[0,1] maps to
// v = V0 + t·(V1−V0), and the point is evaluated ON the torus at (u(v), v) — hence exactly on both
// the torus and the cut plane (unlike a numeric tracer's polyline, the edge stays analytic).
//
// It is the edge curve a torus's oblique half-space cut stores (as a Circle is the edge a
// perpendicular cut stores); the two branches over the same [V0, V1] meet at the oval's v-extremes
// (where w=±1, both roots collapse to u=Phi) to close one spiric loop.
type SpiricArc struct {
	Torus   Torus
	Phi     float64 // azimuth the in-plane cut normal points (u(v) = Phi ± arccos w)
	M, K, C float64 // section coefficients (see the type doc)
	Branch  float64 // +1 or −1: which arccos root this arc follows
	V0, V1  float64 // tube-angle range; t∈[0,1] maps to v = V0 + t·(V1−V0)
}

// TorusSectionCoeffs returns the spiric coefficients (Phi, M, K, C) for the torus cut by plane pl,
// derived from the plane's unit normal m and origin o relative to the torus frame:
//
//	C = m·axis,  M = |m − C·axis|,  Phi = atan2(m·binormal, m·Ref),  K = m·(o − Center)
//
// so g(u,v) = (R + r·cos v)·M·cos(u − Phi) + C·r·sin v − K. Both the section solver and a torus's
// oblique half-space cut build their SpiricArc edges from these, keeping the section math in one place.
func TorusSectionCoeffs(t Torus, pl Plane) (phi, m, k, c float64) {
	mv := unitVec3(pl.Normal())
	axis := t.AxisDir.AsVector()
	binormal := axis.Cross(t.Ref.AsVector())
	c = float64(mv.Dot(axis))
	perp := mv.Sub(axis.Scale(math.Scalar(c)))
	m = float64(perp.Length())
	phi = stdmath.Atan2(float64(mv.Dot(binormal)), float64(mv.Dot(t.Ref.AsVector())))
	k = float64(t.Center.VectorTo(pl.Origin).Dot(mv))
	return phi, m, k, c
}

// uOfV returns the azimuth u on this branch at tube angle v: Phi + Branch·arccos(w(v)). The arccos
// argument is clamped to [−1,1] so the oval's v-extremes (where w grazes ±1, the two branches meet)
// stay finite rather than producing NaN from floating-point overshoot.
func (s SpiricArc) uOfV(v float64) float64 {
	cv, sv := cosSin(v)
	denom := s.M * (s.Torus.MajorRadius + s.Torus.MinorRadius*cv)
	w := (s.K - s.C*s.Torus.MinorRadius*sv) / denom
	return s.Phi + s.Branch*stdmath.Acos(clampUnit(w))
}

// PointAt returns the point at parameter t∈[0,1], evaluated on the torus at (u(v), v).
func (s SpiricArc) PointAt(t float64) math.Point3 {
	v := s.V0 + t*(s.V1-s.V0)
	return s.Torus.PointAt(s.uOfV(v), v)
}

// TangentAt returns dP/dt by the chain rule: (V1−V0)·(du/dv·∂P/∂u + ∂P/∂v). Near the oval's
// v-extremes du/dv diverges (the parameterization has a vertical tangent there); the chord-and-angle
// edge sampler drives off PointAt alone, so a large-but-finite tangent there is harmless.
func (s SpiricArc) TangentAt(t float64) math.Vector3 {
	v := s.V0 + t*(s.V1-s.V0)
	u := s.uOfV(v)
	du, dv := s.Torus.DerivativesAt(u, v)
	return du.Scale(math.Scalar(s.dUdV(v))).Add(dv).Scale(math.Scalar(s.V1 - s.V0))
}

// dUdV returns du/dv = Branch·(−w′/√(1−w²)) where w(v) = (K − C·r·sin v)/(M·(R + r·cos v)). The
// denominator is clamped away from zero at the v-extremes so the tangent stays finite.
func (s SpiricArc) dUdV(v float64) float64 {
	cv, sv := cosSin(v)
	R, r := s.Torus.MajorRadius, s.Torus.MinorRadius
	den := s.M * (R + r*cv)
	w := (s.K - s.C*r*sv) / den
	// w′ = [(−C·r·cos v)·(R+r·cos v) − (K − C·r·sin v)·(−r·sin v)] / (M·(R+r·cos v)²)
	num := (-s.C*r*cv)*(R+r*cv) + (s.K-s.C*r*sv)*r*sv
	wPrime := num / (s.M * (R + r*cv) * (R + r*cv))
	root := stdmath.Sqrt(stdmath.Max(1-w*w, 1e-12))
	return s.Branch * (-wPrime / root)
}

// Domain returns [0, 1].
func (s SpiricArc) Domain() (lo, hi float64) { return 0, 1 }
