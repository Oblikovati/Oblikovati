// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sort"

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
	return s.Phi + s.Branch*stdmath.Acos(spiricCosineAtLimit(w))
}

// spiricCosineAtLimit resolves w for the arccos, snapping it to ±1 where it is within rounding of them.
//
// |w| = 1 is the oval's v-EXTREME: the two branches of one oval meet there, and it is the only place
// they do. Arccos is infinitely steep at its ends, so |w| short of 1 by δ gives an angle √(2δ) away
// from the limit — half the mantissa is lost, and δ of half an ulp puts the two branches 3·10⁻⁸ apart
// in azimuth. Each then samples the shared point to a different place, the two arcs of the oval fail to
// weld into one loop, and the arrangement sees an open chain that divides nothing: a torus cut by a
// plane between its tube radii kept the WHOLE face on both sides (ADR-0062).
//
// A |w| that exceeds 1 by rounding is not a solution at all, and one short of it by rounding IS the
// limit — the arc's domain is by construction the interval where |w| ≤ 1. So both are answered the same
// way, and the two branches then evaluate one azimuth, not two.
func spiricCosineAtLimit(w float64) float64 {
	// tol:calibrated — a few ulps of 1, the rounding w itself carries; arccos amplifies it by a square
	// root, so the snap must happen BEFORE the call, not be absorbed after it.
	const limitUlps = 8 * 2.220446049250313e-16
	if stdmath.Abs(w) >= 1-limitUlps {
		return stdmath.Copysign(1, w)
	}
	return math.Clamp(w, -1, 1)
}

// UAt returns the azimuth u on this branch at tube angle v — the spiric section's single-valued u(v).
// A two-oval band (a plane that cuts both tube walls) is meshed by sweeping v over the full tube period
// and filling u between the two branches' UAt, so the tessellator needs this without inverting PointAt.
func (s SpiricArc) UAt(v float64) float64 { return s.uOfV(v) }

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

// TorusPlaneSection returns the curves a plane cuts from a torus when the section is a SPIRIC — the
// quartic of Perseus — rather than the two concentric circles a perpendicular plane gives. It is the
// analytic answer for every other plane, so the section solver need not decline them (ADR-0061 stage 3).
//
// The section is single-valued in the tube angle on each branch: u(v) = Φ ± arccos w(v), with
// w(v) = (K − C·r·sin v) / (M·(R + r·cos v)) (see [TorusSectionCoeffs]). Where |w| < 1 both branches
// exist; where |w| = 1 they meet, and the curve turns. So the section is read off the v-set on which the
// plane reaches the tube:
//
//   - |w| ≤ 1 for EVERY v — the plane passes through the hole and cuts both walls — gives two curves,
//     each a branch wrapping the whole tube period;
//   - otherwise the set is a union of v-intervals, and each interval closes into one oval: the +1 branch
//     out and the −1 branch back.
//
// ok=false when the coefficients are degenerate (a cut normal that is purely axial, where M = 0 and the
// azimuth is not single-valued in the tube angle).
func TorusPlaneSection(t Torus, pl Plane) ([]Curve3, bool) {
	phi, m, k, c := TorusSectionCoeffs(t, pl)
	if m <= 0 {
		return nil, false // the cut normal has no radial part: u is not single-valued in v
	}
	arc := func(branch, v0, v1 float64) Curve3 {
		return SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: branch, V0: v0, V1: v1}
	}
	spans, whole := spiricTubeSpans(t, m, k, c)
	if whole {
		return []Curve3{arc(1, -stdmath.Pi, stdmath.Pi), arc(-1, -stdmath.Pi, stdmath.Pi)}, true
	}
	out := make([]Curve3, 0, 2*len(spans))
	for _, sp := range spans {
		out = appendRealArc(out, arc(1, sp[0], sp[1]), t)
		out = appendRealArc(out, arc(-1, sp[1], sp[0]), t)
	}
	return out, len(out) > 0
}

// appendRealArc keeps a section arc that spans real length, and drops one that is a POINT.
//
// At the offset where the plane is exactly tangent to the tube, the two boundary roots of w(v) = ±1
// coincide, so one of the spans between them has zero width and BOTH of its branch arcs are the tangency
// point repeated. They are not lobes and they bound nothing: fed to a boolean as imprints they are
// closed curves of zero extent, and everything downstream that samples an imprint — the island walk,
// the arrangement, the meeting solver — sees a ring of identical points. Measured on the axis-parallel
// figure-eight, the section returned FOUR arcs where there are two lobes, the two extra ones collapsed
// onto (0, 3, 0), and the cut came out with one lid instead of two (ADR-0061 stage 2).
//
// The test is the arc's own extent against the tube's weld, not its parameter width: a span is an
// angle, and what disqualifies an arc is bounding no length.
func appendRealArc(out []Curve3, cv Curve3, t Torus) []Curve3 {
	if arcSpansLength(cv, ResolutionForSize(t.MinorRadius).Weld()) {
		return append(out, cv)
	}
	return out
}

// arcSpansLength reports an arc reaching farther than tol from where it starts.
func arcSpansLength(cv Curve3, tol float64) bool {
	lo, hi := cv.Domain()
	start := cv.PointAt(lo)
	for i := 1; i <= arcExtentProbe; i++ {
		if float64(start.DistanceTo(cv.PointAt(lo+(hi-lo)*float64(i)/arcExtentProbe))) > tol {
			return true
		}
	}
	return false
}

// arcExtentProbe samples an arc to see whether it goes anywhere. It bounds the curve, nothing more.
const arcExtentProbe = 8

// spiricTubeSpans is the set of tube angles on which the plane reaches the tube — where |w(v)| ≤ 1 —
// as intervals, or whole=true when that is every angle. The boundaries solve w(v) = ±1, each of which
// is A·cos v + B·sin v = D and so closed form.
func spiricTubeSpans(t Torus, m, k, c float64) (spans [][2]float64, whole bool) {
	r, rr := t.MinorRadius, t.MajorRadius
	roots := append(
		harmonicRoots(m*r, c*r, k-m*rr),
		harmonicRoots(-m*r, c*r, k+m*rr)...)
	inside := func(v float64) bool {
		cv, sv := cosSin(v)
		return stdmath.Abs((k-c*r*sv)/(m*(rr+r*cv))) <= 1
	}
	if len(roots) == 0 {
		return nil, inside(0) // no boundary: the plane reaches the tube at every angle, or at none
	}
	sort.Float64s(roots)
	roots = append(roots, roots[0]+2*stdmath.Pi) // close the period
	for i := 0; i+1 < len(roots); i++ {
		if inside((roots[i] + roots[i+1]) / 2) {
			spans = append(spans, [2]float64{roots[i], roots[i+1]})
		}
	}
	return spans, false
}

// harmonicRoots solves A·cos v + B·sin v = D for v in [−π, π), as the two roots of
// cos(v − atan2(B, A)) = D / √(A²+B²) when that ratio is within reach.
//
// The ratio goes through spiricCosineAtLimit for the same reason w does: |D| = amp is the TANGENCY,
// where the two roots coincide, and arccos is infinitely steep there. A ratio short of 1 by half an ulp
// put the double root's two halves 1.5·10⁻⁸ apart in v — 1.03·10⁻⁷ apart on a tube of radius 2 — so the
// section's two lobes each came back as an arc that does not close on itself, by a hair. Downstream
// that is not a hair: the stitch stores a near-closed edge as an OPEN one and recovers its direction by
// inverting the curve at endpoints 10⁻⁷ apart, which does not round-trip, and one lobe's loop came back
// wound against its own material (ADR-0061).
func harmonicRoots(a, b, d float64) []float64 {
	amp := stdmath.Hypot(a, b)
	if amp == 0 || stdmath.Abs(d) > amp {
		return nil
	}
	base := stdmath.Atan2(b, a)
	off := stdmath.Acos(spiricCosineAtLimit(d / amp))
	return []float64{wrapToPi(base + off), wrapToPi(base - off)}
}

// wrapToPi folds an angle into [−π, π).
func wrapToPi(v float64) float64 {
	for v < -stdmath.Pi {
		v += 2 * stdmath.Pi
	}
	for v >= stdmath.Pi {
		v -= 2 * stdmath.Pi
	}
	return v
}
