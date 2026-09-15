// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The IMPLICIT side of the torus section (ADR-0066, Oblikovati#3514).
//
// The torus bucket runs the intersector's parametric×implicit substitution the other way round from
// the ruled bucket: the TORUS supplies the parametrisation and the other surface supplies an implicit
// equation, which the reduction restricts to one tube circle at a time. What that reduction needs of
// the other surface is therefore not a TYPE and not even a quadric — it is an implicit form whose
// restriction to a circle is a trigonometric polynomial the station solver can read.
//
// Two forms are in that class, and there is a reason there are exactly two rather than a list that
// grows: [Quadric], and a second torus's own quartic. torus_torus_harmonic.go carries the argument —
// both restrict to the SAME degree-two trigonometric polynomial, because a circle meets a torus in
// four finite points, not eight, the other four having been spent at the circular points at infinity.
// A form that restricted to a richer polynomial would need a richer station solver; neither of these
// does.

// TorusCoForm is the implicit form of the surface a torus is sectioned AGAINST. It is sealed to this
// package — its methods are unexported — because implementing it is a claim about the DEGREE of the
// restriction, and the reduction's whole lane structure rests on that claim being true.
type TorusCoForm interface {
	// stationOn reduces the form's constraint to one tube circle of t, at tube angle v.
	stationOn(t Torus, v float64) torusStation
	// incidence is a function zero exactly on the form's OWN surface — the second of the two
	// conditions a section curve satisfies (curve_incidence.go). It is the form's best-conditioned
	// such function, which is not always the polynomial the reduction substitutes into: a torus's
	// quartic grows as the fourth power of distance, so a torus answers its signed distance instead.
	incidence() func(math.Point3) float64
	// distanceTo is the point's distance from the form's OWN surface, in model units. It is what the
	// section's post-condition gates on: near a fold the implicit residual falls off quadratically in
	// the azimuth error while the position error does not, so only a length can see a branch that has
	// wandered off the surface there.
	distanceTo(p math.Point3) float64
	// stationValueLipschitz bounds |∂f/∂v| — how fast the value of the form's OWN implicit polynomial,
	// read on the chart torus, can change with the TUBE ANGLE — everywhere on the chart at once. It is
	// what lets a sampled sweep reason about the gaps BETWEEN its samples instead of only at them, and
	// every form can state it because ∂f/∂v is ∇F(P)·∂P/∂v: a bound on the form's own gradient over the
	// chart's reach, times |∂P/∂v| = the chart's minor radius exactly. Each form answers from the
	// representation that carries its gradient (Oblikovati/Oblikovati#3515).
	stationValueLipschitz(chart Torus) float64
	// stationSlopeLipschitz bounds |∂(∂f/∂u)/∂v| — how fast the station polynomial's SLOPE in the
	// azimuth can change with the tube angle — everywhere on the chart at once. Where
	// stationValueLipschitz lets a sweep reason about a root between its samples, this lets it reason
	// about an EXTREMUM between them, which is what "the station carries the same number of extrema at
	// every tube angle" needs (torus_section_lane.go). ∂²f/∂u∂v is Puᵀ·∇²F·Pv + ∇F·Puv, so a form
	// answers it from a bound on its own HESSIAN alongside the gradient bound it already states.
	stationSlopeLipschitz(chart Torus) float64
	// coaxialLevelFactors returns the tube-angle polynomials whose roots are exactly the tube angles at
	// which this form meets a COAXIAL chart torus. They are solved, never sampled: how many circles the
	// section has is a topological question, and torus_coaxial_section.go carries why each form's level
	// is a low-degree trigonometric polynomial in the tube angle.
	coaxialLevelFactors(chart Torus) []torusSecondHarmonic
	// chartTorus returns the torus this form is, when it is one. It is how the role classification
	// finds a chart without asserting on a geometry kind a second time.
	chartTorus() (Torus, bool)
	// sectionFamily classifies the form against a torus: which of the three reductions applies, over
	// the WHOLE turn. Each form answers from the representation that carries the property — a quadric
	// from its tensor, a torus from its stations (intersect_torus_section.go).
	sectionFamily(t Torus) torusSectionFamily
}

var (
	_ TorusCoForm = Quadric{}
	_ TorusCoForm = Torus{}
)

// incidence is the quadric's own F, which is zero on its surface and well scaled next to the model.
func (q Quadric) incidence() func(math.Point3) float64 { return q.ValueAt }

// distanceTo is the quadric's FIRST-ORDER distance |F| / |∇F|, with ∇F = 2(MW + G). F vanishes on the
// surface and its gradient is the surface normal scaled by how fast F climbs, so the ratio is the
// Newton step to the surface — exact in the limit and, for the sub-weld displacements this gate reads,
// exact to their own square. A point where the gradient vanishes is on the quadric's own singular locus
// (a cone's apex); it answers an infinite distance there, which refuses rather than admits.
func (q Quadric) distanceTo(p math.Point3) float64 {
	w := q.Anchor.VectorTo(p)
	grad := float64(2 * float64(q.M.Apply(w).Add(q.G).Length()))
	if grad == 0 {
		return stdmath.Inf(1)
	}
	return float64(stdmath.Abs(q.ValueAt(p)) / grad)
}

// stationValueLipschitz is exact arithmetic on the two factors, not an estimate. ∂f/∂v is ∇Q(P)·∂P/∂v
// with ∇Q(X) = 2(M W + G), and |∂P/∂v| is the chart's minor radius exactly (∂P/∂v = r(−sin v·e(u) +
// cos v·â), an orthogonal pair scaled by r). |W| is at most the distance from the quadric's anchor to the
// chart's centre plus the chart's own reach, R + r. The tensor norm is Frobenius, which dominates the
// spectral norm, so the product is an upper bound on every direction M can act in.
//
// The bound also covers the value AT an extremum track, which is what the wrap sweep reads: by the
// envelope theorem dA/dv = ∂f/∂v there, because ∂f/∂u vanishes at an extremum by definition.
func (q Quadric) stationValueLipschitz(chart Torus) float64 {
	reach := float64(q.Anchor.VectorTo(chart.Center).Length()) + chart.MajorRadius + chart.MinorRadius
	slope := float64(q.M.Norm()*reach) + float64(q.G.Length())
	return float64(2 * slope * chart.MinorRadius)
}

// stationSlopeLipschitz for a quadric. ∇²Q is the CONSTANT 2M, so the mixed term is
// 2‖M‖·|Pu|·|Pv| with |Pu| = ρ(v) ≤ R + r and |Pv| = r exactly, and the second term is the same
// gradient bound [Quadric.stationValueLipschitz] already forms, against |Puv| = |ρ'(v)| ≤ r — which is
// that function's own value. The tensor norm is Frobenius, which dominates the spectral norm.
func (q Quadric) stationSlopeLipschitz(chart Torus) float64 {
	curve := float64(2 * float64(q.M.Norm()) * (chart.MajorRadius + chart.MinorRadius) * chart.MinorRadius)
	return curve + q.stationValueLipschitz(chart)
}

// chartTorus reports that a quadric is not a torus, so it can only ever take the implicit role.
func (q Quadric) chartTorus() (Torus, bool) { return Torus{}, false }

// torusCoFormOf returns the implicit form s contributes to a torus section: its quartic when s is a
// torus, its quadric when s has one, and ok=false when the surface has neither (a B-spline, an offset,
// a threaded cylinder). It is the ONE place the torus bucket asks what a surface is.
func torusCoFormOf(s Surface) (TorusCoForm, bool) {
	if t, ok := s.(Torus); ok {
		return t, true
	}
	if q, ok := s.(ImplicitQuadric); ok {
		return q.QuadricForm(), true
	}
	return nil, false
}

// torusSectionRoles assigns the two roles of a torus section — the CHART the reduction parametrises
// on, and the implicit form it substitutes into — for a pair at least one side of which is a torus.
//
// It is a CLASSIFICATION and not a try-list: exactly one assignment comes back, and when it declines
// the pair is refused by name rather than retried the other way round. A pair of TORI is the case that
// makes the distinction bite, because both role assignments apply; [torusChartPrecedes] settles it
// with a total order, so the same two surfaces reduce on the same chart whichever order the caller
// hands them over in, and the section's bytes do not depend on the arrangement's face order.
//
//	chart, co, ok := torusSectionRoles(ringFace.Geometry(), rodFace.Geometry())
func torusSectionRoles(a, b Surface) (Torus, TorusCoForm, bool) {
	fa, aOK := torusCoFormOf(a)
	fb, bOK := torusCoFormOf(b)
	if !aOK || !bOK {
		return Torus{}, nil, false
	}
	ca, aIsTorus := fa.chartTorus()
	cb, bIsTorus := fb.chartTorus()
	if aIsTorus && bIsTorus {
		return torusPairRoles(ca, cb)
	}
	if aIsTorus {
		return ca, fb, true
	}
	return cb, fa, bIsTorus
}

// torusPairRoles picks which of two tori carries the chart.
func torusPairRoles(a, b Torus) (Torus, TorusCoForm, bool) {
	if torusChartPrecedes(b, a) {
		return b, a, true
	}
	return a, b, true
}

// torusChartPrecedes orders two tori for the CHART role, the NARROWER tube first.
//
// What the lane structure asks is that the station's extremum tracks stay separable across the whole
// turn (torusLaneAnchors). Every station coefficient is built from the tube circle at tube angle v —
// radius ρ(v) = R + r·cos v, centre offset r·sin v along the axis — so the CHART's own minor radius is
// the amplitude, in model units, by which the station's geometry swings over the turn. A narrow chart's
// stations barely move, its extremum tracks keep their identity, and its branches run the whole turn;
// a wide one's extremum COUNT changes over the turn, and a count that changes makes "which branch pair"
// a guess. R does not enter that amplitude, which is why the key is the minor radius and not the tube
// aspect r/R.
//
// ADR-0066 measured this the other way round and wrote the WIDER-tube rule, on the aspect. The reversal
// is not a disagreement about arithmetic — it is that #3514 measured in a world where a branch pair
// that never folds was a REFUSAL ("the torus section's branch pair never folds"). A narrow chart
// produces exactly that shape, so every one of its wins was declined rather than counted. #3515 carries
// those branches as full-period arcs, and with the refusal gone the same experiment reverses.
//
// Measured at #3515's head over random ring-torus pairs, counting only sections that build AND lie on
// the co-form at 1001 samples per curve (a build that is wrong is worse than a decline, so an objective
// counting builds alone would prefer the assignment that is wrong more often):
//
//	seed        pairs   either assignment   minor radius   tube aspect   ADR-0066's wider-tube rule
//	  9          1500          537              480            477                  387
//	 13          1500          509              456            454                  370
//	 21          1500          519              456            452                  367
//
// The narrow rule wins on every seed and loses on none, and the minor radius edges the aspect on every
// seed while also keeping the "small ring through the hole" corpus row building, which the aspect drops.
// Both are invariant under a uniform change of units — scaling both tori scales both keys, so the ORDER
// is unchanged — so ADR-0066's reason for preferring the aspect does not separate them.
//
// Everything after the swing is a TIE-BREAK, and it is exhaustive on purpose: two copies of one ring
// is the commonest torus pair there is, and a rule that left their order to the caller would let the
// same model produce two different sets of section bytes. The keys are compared exactly.
func torusChartPrecedes(x, y Torus) bool {
	return compareTorusChartKeys(torusChartKey(x), torusChartKey(y)) < 0
}

// torusChartKey is the chart order's key: the tube radius that decides it — the amplitude by which a
// chart's stations swing over the turn — then the major radius, the centre and the axis as the
// exhaustive tie-break.
func torusChartKey(t Torus) [8]float64 {
	a := t.AxisDir.AsVector()
	return [8]float64{
		t.MinorRadius, t.MajorRadius,
		float64(t.Center.X), float64(t.Center.Y), float64(t.Center.Z),
		float64(a.X), float64(a.Y), float64(a.Z),
	}
}

// compareTorusChartKeys compares two chart keys lexicographically on their exact bits: −1, 0 or +1.
func compareTorusChartKeys(x, y [8]float64) int {
	for i := range x {
		if x[i] < y[i] {
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
	}
	return 0
}
