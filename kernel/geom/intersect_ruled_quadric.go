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
		canonicalSection(RuledQuadricArc{Base: base, Quad: quad, Upper: false, U0: 0, U1: twoPi}, res),
		canonicalSection(RuledQuadricArc{Base: base, Quad: quad, Upper: true, U0: 0, U1: twoPi}, res),
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
// so this brackets every local minimum of the branch gap, which is then refined to rounding.
const ruledQuadricAzimuthProbes = 720

// ruledQuadricSkewFloor is the smallest |a| the gate accepts, as a fraction of ‖M‖·|D|². a = D·(M D)
// measures how transverse the ruling is to the quadric: it is sin² of the angle between a cylinder's
// ruling and the other cylinder's axis, so this floor is a ~1.8° near-parallel guard. Below it one
// root escapes toward infinity and the quadratic is solved by cancellation — the ill-conditioned
// regime the fast path must demote from, per the kernel ground rules.
const ruledQuadricSkewFloor = 1e-3 // tol:conditioning — dimensionless transversality of ruling to quadric

// ruledQuadricConditioning reports whether base∩quad is the well-conditioned two-wrap section: base is
// affine in its ruling parameter (the straight-ruling certificate), the ruling stays transverse to the
// quadric, and the two roots stay APART across the whole azimuth — apart at the modelling resolution,
// which is the certificate that the two branches are two curves the stitch can tell from one another.
// It is a gate on CONDITIONING, not on surface type — a torus or a B-spline base fails the affine
// certificate, a near-parallel pair the transversality floor, and a tangent pair the separation.
//
// The separation is read at its exact minimum: the probes bracket every local minimum of the gap and
// the extremum solver refines each to rounding, so the certificate holds between the probes as well as
// at them. It used to demand a MARGIN instead — the smallest gap at least a twentieth of the largest —
// which refused every near-pinch crossing while its roots were exact to 1e-13: two cylinders whose
// radii differ by a part in a hundred thousand keep their branches 2√(2R·Δr) apart, a thousand welds,
// and the margin was a policy standing in for this measurement. OCCT's cylinder∩cylinder walker
// (IntPatch_ImpImpIntersection, CyCyNoGeometric) parametrises the same section with no separation
// margin at all; what it guards is the fold, which this intersector refuses by base role
// (ADR-0061 stage 4).
func ruledQuadricConditioning(base Surface, quad Quadric, res Resolution) bool {
	mNorm := quad.M.Norm()
	if mNorm <= 0 {
		return false // a degenerate (planar) quadric: the plane∩ruled conics are their own closed form
	}
	gapAt := func(u float64) (float64, bool) {
		r := straightRulingAt(base, u)
		if r.SecondDiffScale > res.Weld() {
			return 0, false // not affine in v: this surface has no straight ruling to substitute
		}
		co := quad.alongRuling(r)
		if stdmath.Abs(co.a) < ruledQuadricSkewFloor*mNorm*float64(r.Dir.LengthSquared()) {
			return 0, false // the ruling runs (near) along the quadric: one root escapes, the solve cancels
		}
		return co.separation(), true
	}
	gaps := make([]float64, ruledQuadricAzimuthProbes)
	for i := range gaps {
		g, ok := gapAt(twoPi * float64(i) / ruledQuadricAzimuthProbes)
		if !ok {
			return false
		}
		gaps[i] = g
	}
	return minimumBranchGap(gaps, gapAt) > res.Stitch()
}

// minimumBranchGap refines every bracketed local minimum of the sampled gap to its exact value and
// returns the smallest, reading the sweep as the circle it is.
func minimumBranchGap(gaps []float64, gapAt func(float64) (float64, bool)) float64 {
	n := len(gaps)
	step := twoPi / float64(n)
	least := stdmath.Inf(1)
	for i, g := range gaps {
		if g > gaps[(i+n-1)%n] || g > gaps[(i+1)%n] {
			continue
		}
		u := ExtremumOnBracket(func(u float64) float64 {
			gap, _ := gapAt(u)
			return gap
		}, float64(i-1)*step, float64(i+1)*step, false)
		if gap, ok := gapAt(u); ok {
			g = stdmath.Min(g, gap)
		}
		least = stdmath.Min(least, g)
	}
	return least
}
