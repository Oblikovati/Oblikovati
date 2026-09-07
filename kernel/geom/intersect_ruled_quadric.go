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
// then a WINDOW — the ruling meets the quadric over part of the sweep and misses it outside, so the
// two arcs join at their turning points into one closed loop. The wrap form is tried first, on either
// base in turn, because a rod crossing a wall is a full wrap on the ROD's chart even when it is a
// window on the wall's, and the wrap has no singular point to carry. A pair that is a window on BOTH
// charts — a ball crossing a rod off its axis is the smallest example — is the window form's own case
// and comes back as [RuledQuadricLoop] (ADR-0061 stage 5).

// intersectRuledQuadric returns the exact intersection curves of a straight-ruled surface and an
// implicit quadric, or handled=false when neither role assignment is well-conditioned. Both role
// assignments are tried because the same crossing is a full azimuth wrap on one operand's chart and a
// folded window on the other's; the wrap is the form with no singular point.
func intersectRuledQuadric(a, b Surface, res Resolution) ([]Curve3, bool) {
	if curves, ok := RuledQuadricSection(a, b, res); ok {
		return curves, true
	}
	if curves, ok := RuledQuadricSection(b, a, res); ok {
		return curves, true
	}
	// Neither chart carries the section as a full wrap, so it folds on both: the window form owns it.
	if curves, ok := RuledQuadricWindows(a, b, res); ok {
		return curves, true
	}
	return RuledQuadricWindows(b, a, res)
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

// RuledQuadricWindows returns base∩other as closed WINDOW loops on base's chart — the folded form of
// the same closed section, for the pairs whose ruling meets the quadric over part of the azimuth only
// (ADR-0061 stage 5). An empty result with ok=true means the ruling misses the quadric everywhere, so
// the surfaces are known not to cross; ok=false is the same refusal [RuledQuadricSection] makes — base
// is not straight-ruled and periodic, other has no quadric form, or a window is too ill-conditioned to
// name (its branches never separate past the stitch resolution, or a fold is a double root rather than
// a turning point).
//
//	loops, ok := geom.RuledQuadricWindows(rodCylinder, ball, geom.ResolutionForBox(box))
func RuledQuadricWindows(base, other Surface, res Resolution) ([]Curve3, bool) {
	implicit, ok := other.(ImplicitQuadric)
	if !ok || !isFullAzimuth(base) {
		return nil, false
	}
	quad := implicit.QuadricForm()
	discs, ok := sampleRuledDiscriminants(base, quad, res)
	if !ok {
		return nil, false
	}
	spans, ok := discriminantWindows(base, quad, discs)
	if !ok {
		return nil, false // no fold on this chart: the section is a full wrap, which RuledQuadricSection owns
	}
	if len(spans) == 0 {
		return nil, true // the ruling misses the quadric at every azimuth: no crossing, and that is an answer
	}
	loops := make([]Curve3, 0, len(spans))
	for _, w := range spans {
		loop := RuledQuadricLoop{Base: base, Quad: quad, U0: w[0], U1: w[1]}
		if !ruledQuadricWindowConditioning(loop, res) {
			return nil, false
		}
		loops = append(loops, loop)
	}
	return loops, true
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

// sampleRuledDiscriminants reads the ruling quadratic's discriminant across the base's whole azimuth,
// refusing the same two things the wrap form's gate refuses before it looks at any root: a base that is
// not affine in its ruling parameter (it has no straight ruling to substitute), and a ruling that runs
// along the quadric (one root escapes and the solve cancels). Only the SEPARATION test differs between
// the two forms, because a window's branches are meant to meet at its ends.
func sampleRuledDiscriminants(base Surface, quad Quadric, res Resolution) ([]float64, bool) {
	mNorm := quad.M.Norm()
	if mNorm <= 0 {
		return nil, false // a degenerate (planar) quadric: the plane∩ruled conics are their own closed form
	}
	out := make([]float64, ruledQuadricAzimuthProbes)
	for i := range out {
		r := straightRulingAt(base, twoPi*float64(i)/ruledQuadricAzimuthProbes)
		if r.SecondDiffScale > res.Weld() {
			return nil, false
		}
		co := quad.alongRuling(r)
		if stdmath.Abs(co.a) < ruledQuadricSkewFloor*mNorm*float64(r.Dir.LengthSquared()) {
			return nil, false
		}
		out[i] = co.discriminant()
	}
	return out, true
}

// discriminantWindows returns the maximal azimuth spans over which the ruling meets the quadric, each
// bounded by the exact fold azimuths where the discriminant crosses zero. A span that wraps the seam is
// returned as [U0, U0+width] with U1 past 2π, which the base's periodic chart evaluates unchanged.
//
// ok=false is "not this form's case": the discriminant never changes sign, so either the ruling meets
// the quadric everywhere — a full wrap, which [RuledQuadricSection] owns and which must not be dressed
// up as a loop whose two folds are the same azimuth — or the count of rises and falls disagrees, which
// cannot happen on a circle and so is a numerical answer nobody should build on. An empty span list
// with ok=true is the honest "they do not meet".
func discriminantWindows(base Surface, quad Quadric, discs []float64) ([][2]float64, bool) {
	n := len(discs)
	step := twoPi / float64(n)
	var rises, falls []float64
	for i, d := range discs {
		prev := discs[(i+n-1)%n]
		switch {
		case prev <= 0 && d > 0:
			rises = append(rises, foldAzimuth(base, quad, float64(i-1)*step, float64(i)*step))
		case prev > 0 && d <= 0:
			falls = append(falls, foldAzimuth(base, quad, float64(i-1)*step, float64(i)*step))
		}
	}
	if len(rises) == 0 && len(falls) == 0 {
		return nil, allNonPositive(discs) // no fold: a full wrap (refuse) or no crossing at all (an answer)
	}
	if len(rises) != len(falls) {
		return nil, false
	}
	out := make([][2]float64, 0, len(rises))
	for _, u0 := range rises {
		out = append(out, [2]float64{u0, nextAbove(falls, u0)})
	}
	return out, true
}

// allNonPositive reports that the ruling misses the quadric at every probe.
func allNonPositive(discs []float64) bool {
	for _, d := range discs {
		if d > 0 {
			return false
		}
	}
	return true
}

// nextAbove returns the first fall azimuth strictly after u0, wrapped by a period when the window
// straddles the seam — so a window is always [U0, U1] with U0 < U1.
func nextAbove(falls []float64, u0 float64) float64 {
	best := stdmath.Inf(1)
	for _, f := range falls {
		u := f
		if u <= u0 {
			u += twoPi
		}
		best = stdmath.Min(best, u)
	}
	return best
}

// foldAzimuth refines a bracketed sign change of the discriminant to the fold itself. The bracket comes
// from the probe sweep, so this is a bisection on a smooth trigonometric function with one root in it —
// no derivative, unconditionally convergent.
//
// It returns the endpoint on the NON-POSITIVE side, never the midpoint. That side is where the two
// roots have already merged, so [ruledQuadricCoeffs.foldRoot] answers the double root −b/2a there and
// the loop's two halves start and end at the SAME point — the closure is exact rather than 2√Δ/|a|
// wide, and √Δ of a discriminant bisected to rounding is still a hundred thousand times the rounding.
func foldAzimuth(base Surface, quad Quadric, lo, hi float64) float64 {
	discAt := func(u float64) float64 {
		return quad.alongRuling(straightRulingAt(base, u)).discriminant()
	}
	loPositive := discAt(lo) > 0
	for range foldBisectionSteps {
		mid := (lo + hi) / 2
		if (discAt(mid) > 0) == loPositive {
			lo = mid
			continue
		}
		hi = mid
	}
	if loPositive {
		return hi
	}
	return lo
}

// foldBisectionSteps halves the probe bracket to the fold. The bracket is 2π/720 wide, so 60 halvings
// take it below the double's own resolution — the fold is then exact to rounding, which is what the
// window's two ends have to be for the loop to close on itself.
const foldBisectionSteps = 60

// ruledQuadricWindowConditioning certifies one window before a loop is built on it: its branches must
// separate, somewhere inside, by more than the stitch resolution — otherwise the whole loop is a
// grazing sliver two faces could not be told apart across — and each fold must be a simple root of the
// discriminant, which is what makes it a turning point rather than a tangency the window closes on.
//
// It is deliberately the mirror of the wrap form's gate: that one reads the MINIMUM separation across
// the sweep, because a wrap has no fold and its two branches must never meet; this one reads the
// MAXIMUM inside the window, because a window's branches meet at both ends by construction.
func ruledQuadricWindowConditioning(l RuledQuadricLoop, res Resolution) bool {
	if l.coeffsAt(l.U0).discriminantSlope() == 0 || l.coeffsAt(l.U1).discriminantSlope() == 0 {
		return false // a double root: the window closes on a tangency, not on a turning point
	}
	widest := 0.0
	for i := 1; i < ruledQuadricWindowProbes; i++ {
		u := l.U0 + (l.U1-l.U0)*float64(i)/ruledQuadricWindowProbes
		widest = stdmath.Max(widest, l.coeffsAt(u).separation())
	}
	return widest > res.Stitch()
}

// ruledQuadricWindowProbes samples a window's interior for its widest branch separation. The separation
// has one interior maximum for every ruled/quadric pair (it is √Δ/|a| with Δ a low-order trigonometric
// polynomial vanishing at both ends), so a coarse sweep finds it.
const ruledQuadricWindowProbes = 64
