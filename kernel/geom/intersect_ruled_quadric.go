// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The ruled∩quadric bucket of [IntersectSurfacesAnalytic]. Given a straight-ruled base (the
// parametric side) and a quadric (the implicit side), the section is the root set of ONE quadratic in
// the ruling parameter, exactly as ruled_quadric_arc.go derives. What is left is a TOPOLOGY question:
// which connected pieces those roots form over the base's periodic azimuth. Where the discriminant
// stays strictly positive across the whole sweep, the answer is unambiguous — two closed loops, one
// per ordered root, each single-valued in u and regular everywhere — and each is exactly one
// [RuledQuadricArc]. Where the discriminant reaches zero the two roots meet at a FOLD: the section is
// then a window (two arcs joined at their turning points) or a pinched pair, whose u-parametrisation
// has a square-root singularity at the fold. That is refused here, with the same base tried the other
// way round first, because a rod crossing a wall is a full wrap on the ROD's chart even when it is a
// window on the wall's.

// intersectRuledQuadric returns the exact intersection curves of a straight-ruled surface and an
// implicit quadric, or handled=false when neither role assignment is well-conditioned. Both role
// assignments are tried because the same crossing is a full azimuth wrap on one operand's chart and a
// folded window on the other's; the wrap is the form with no singular point.
func intersectRuledQuadric(a, b Surface, res Resolution) ([]Curve3, bool) {
	if curves, ok := RuledQuadricSection(a, b, res); ok {
		return curves, true
	}
	return RuledQuadricSection(b, a, res)
}

// RuledQuadricSection returns the two exact full-azimuth section loops of base∩other, evaluated on
// BASE's chart — the ruled∩quadric closed form of [IntersectSurfacesAnalytic], exposed so a consumer
// that must know WHICH chart the arcs live on (the curved boolean's imprint, which clips them to that
// base's own marching window) can choose the role itself. ok is false when base is not straight-ruled
// and periodic, other has no implicit quadric form, or the pair fails the conditioning gate.
//
//	arcs, ok := geom.RuledQuadricSection(rodCylinder, wallCylinder, geom.ResolutionForBox(box))
func RuledQuadricSection(base, other Surface, res Resolution) ([]Curve3, bool) {
	implicit, ok := other.(ImplicitQuadric)
	if !ok || !isFullAzimuth(base) {
		return nil, false
	}
	quad := implicit.QuadricForm()
	if !ruledQuadricConditioning(base, quad, res) {
		return nil, false
	}
	return []Curve3{
		RuledQuadricArc{Base: base, Quad: quad, Upper: false, U0: 0, U1: twoPi},
		RuledQuadricArc{Base: base, Quad: quad, Upper: true, U0: 0, U1: twoPi},
	}, true
}

// isFullAzimuth reports whether a surface's u runs the full periodic circle — the domain the two
// section loops wrap, and the only one over which "the discriminant is positive everywhere" states a
// complete topology rather than a local one.
func isFullAzimuth(s Surface) bool {
	lo, hi := s.UDomain()
	return lo == 0 && hi == twoPi
}

// ruledQuadricAzimuthProbes is how many azimuths the conditioning gate samples across the full sweep.
// The coefficients a, b, c are low-order trigonometric polynomials in u for every ruled/quadric pair,
// so this resolves the discriminant's extrema far finer than the branch-separation margin the gate
// then demands.
const ruledQuadricAzimuthProbes = 720

// ruledQuadricSkewFloor is the smallest |a| the gate accepts, as a fraction of ‖M‖·|D|². a = D·(M D)
// measures how transverse the ruling is to the quadric: it is sin² of the angle between a cylinder's
// ruling and the other cylinder's axis, so this floor is a ~1.8° near-parallel guard. Below it one
// root escapes toward infinity and the quadratic is solved by cancellation — the ill-conditioned
// regime the fast path must demote from, per the kernel ground rules.
const ruledQuadricSkewFloor = 1e-3 // tol:conditioning — dimensionless transversality of ruling to quadric

// ruledQuadricBranchSeparation is the smallest branch gap the gate accepts, as a fraction of the
// largest gap over the sweep. The two roots meeting (gap → 0) is the fold that turns two wraps into a
// window or a pinch, so this is the margin that certifies the wrap topology between the probes as well
// as at them. It is a ratio of two lengths, hence model-scale free.
const ruledQuadricBranchSeparation = 0.05 // tol:conditioning — min/max branch gap over the azimuth sweep

// ruledQuadricConditioning reports whether base∩quad is the well-conditioned two-wrap section: base is
// affine in its ruling parameter (the straight-ruling certificate), the ruling stays transverse to the
// quadric, and the two roots stay apart across the whole azimuth. It is a gate on CONDITIONING, not on
// surface type — a torus or a B-spline base fails the affine certificate, and a tangent or
// near-parallel pair of cylinders fails the numeric margins, both landing on the general marcher.
func ruledQuadricConditioning(base Surface, quad Quadric, res Resolution) bool {
	mNorm := quad.M.Norm()
	if mNorm <= 0 {
		return false // a degenerate (planar) quadric: the plane∩ruled conics are their own closed form
	}
	minGap, maxGap := stdmath.Inf(1), 0.0
	for i := range ruledQuadricAzimuthProbes {
		u := twoPi * float64(i) / ruledQuadricAzimuthProbes
		r := straightRulingAt(base, u)
		if r.SecondDiffScale > res.Weld() {
			return false // not affine in v: this surface has no straight ruling to substitute
		}
		co := quad.alongRuling(r)
		if stdmath.Abs(co.a) < ruledQuadricSkewFloor*mNorm*float64(r.Dir.LengthSquared()) {
			return false // the ruling runs (near) along the quadric: one root escapes, the solve cancels
		}
		gap := co.separation()
		minGap, maxGap = stdmath.Min(minGap, gap), stdmath.Max(maxGap, gap)
	}
	return minGap > 0 && minGap >= ruledQuadricBranchSeparation*maxGap
}
