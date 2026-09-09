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

// torusChartPrecedes orders two tori for the CHART role, FATTER tube first.
//
// The station polynomial is the CO-FORM's quartic restricted to the chart's tube circle, and the one
// thing the lane structure asks is that its extremum tracks stay separable across the whole turn
// (torusLaneAnchors). What decides that is the co-form, not the chart: a thin tube is nearly a wire, so
// its quartic cuts a circle in a simple, stable root structure, while a fat one cuts a structure whose
// extremum COUNT changes over the turn — and a count that changes makes "which branch pair" a guess.
// So the thinner torus belongs on the implicit side, which puts the fatter one on the chart.
//
// The rule was written the other way round first, on the reasoning that the thinner CHART moves its
// stations least. That was wrong, and measuring is what said so. Over meeting random ring-torus pairs
// (torus_torus_test.go's TestTheFatterTubeIsTheBetterChart, three seeds): fat-chart builds 608 / 678 /
// 651 of 1800 / 1775 / 1797, thin-chart 543 fewer on the first seed alone. The fat rule wins on every
// seed and loses on none. An assignment by tube aspect and one by absolute minor radius are within
// noise of each other (607 / 663 / 646); the ASPECT is kept because it is dimensionless, so the same
// two tori in metres and in millimetres take the same chart.
//
// Everything after the aspect is a TIE-BREAK, and it is exhaustive on purpose: two copies of one ring
// is the commonest torus pair there is, and a rule that left their order to the caller would let the
// same model produce two different sets of section bytes. The keys are compared exactly.
func torusChartPrecedes(x, y Torus) bool {
	return compareTorusChartKeys(torusChartKey(x), torusChartKey(y)) < 0
}

// torusTubeAspect is −r/R: the fraction of its own radius the tube circle's radius varies by over the
// turn, NEGATED so that the ascending chart order puts the fatter tube first.
func torusTubeAspect(t Torus) float64 { return -float64(t.MinorRadius / t.MajorRadius) }

// torusChartKey is the chart order's key: the tube aspect that decides it, then the radii, the centre
// and the axis as the exhaustive tie-break.
func torusChartKey(t Torus) [9]float64 {
	a := t.AxisDir.AsVector()
	return [9]float64{
		torusTubeAspect(t), t.MajorRadius, t.MinorRadius,
		float64(t.Center.X), float64(t.Center.Y), float64(t.Center.Z),
		float64(a.X), float64(a.Y), float64(a.Z),
	}
}

// compareTorusChartKeys compares two chart keys lexicographically on their exact bits: −1, 0 or +1.
func compareTorusChartKeys(x, y [9]float64) int {
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
