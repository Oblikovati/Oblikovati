// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The TORUS × TORUS member of the torus section bucket (ADR-0066, Oblikovati#3514) — a fillet meeting
// a fillet, a torus boss on a ring, two linked rings.
//
// ADR-0061 stage 5 recorded this pair as a standing refusal on the ground that "neither side supplies
// an implicit quadric, so the second-harmonic reduction does not apply". That premise is false, and
// the algebra below is why. The second harmonic does not come from the other side being a QUADRIC. It
// comes from the tube circle the chart torus sweeps, and it is there whatever the other side is.
//
// A torus's own implicit form is quartic. Relative to its centre, with W = X − C and â its axis,
//
//	F(X) = (|W|² + R² − r²)² − 4R²(|W|² − (W·â)²)
//
// which is zero exactly on the torus: |W|² + R² − r² = 2R·ρ is the meridian condition
// (ρ − R)² + z² = r² cleared of its square root, with ρ² = |W|² − (W·â)² the squared distance from the
// axis and z = W·â.
//
// Now restrict F to ONE tube circle of the chart torus. A point there is
//
//	P(u) = O(v) + ρ(v)·e(u),   O(v) = C + r·sin v·â,   ρ(v) = R + r·cos v,   e(u) = cos u·ê₁ + sin u·ê₂
//
// and with W₀ = O(v) − C_other the whole azimuth dependence enters through W = W₀ + ρ·e(u). Two terms
// carry it, and BOTH drop a degree on a circle:
//
//	|W|² = |W₀|² + 2ρ·(W₀·e) + ρ²|e|²  = (|W₀|² + ρ²) + 2ρ(b₁·cos u + b₂·sin u)
//
// — the quadratic term is the CONSTANT ρ², because |e| = 1, so |W|² is a FIRST harmonic exactly; and
//
//	(W·â)² = (a₀ + ρ·(n₁·cos u + n₂·sin u))²
//
// — a first harmonic squared, so a second. F is therefore (first)² − 4R²·(first − second), which is a
// SECOND harmonic. Not a fourth: the same five coefficients a quadric produces, read by the same
// station solver, paired into lanes by the same extremum structure, folded by the same window finder.
//
// The geometry says the same thing. Bézout would put a conic against a quartic at eight points; a
// CIRCLE passes through the two circular points at infinity, each of which lies on the absolute conic
// that every torus quartic contains doubly, so four of the eight are spent there and four finite
// intersections remain — a quadric's count. Measured over 20 000 random (chart, other, u, v): the
// coefficients below reproduce F(P(u,v)) to 4.56e-14 of the polynomial's own coefficient scale, against
// the 1e-9 a certified root is allowed; and every certified station root lands on BOTH surfaces to
// 9.23e-14 (torus_torus_test.go).
//
// So there is no new engine here, and no marcher. There are five coefficients, and the existing lane
// machinery does the rest.
//
// The five, with b₁ = W₀·ê₁, b₂ = W₀·ê₂, a₀ = W₀·â, n₁ = ê₁·â, n₂ = ê₂·â, k = R² − r², β = 4R²,
// g₀ = |W₀|² + ρ² + k, g₁ = 2ρb₁, g₂ = 2ρb₂ (all of the OTHER torus's radii, on the CHART's frame):
//
//	Cos2  = (g₁² − g₂²)/2 + β·ρ²·(n₁² − n₂²)/2
//	Sin2  = g₁g₂ + β·ρ²·n₁n₂
//	Cos1  = 2g₀g₁ − 2βρ·(b₁ − a₀n₁)
//	Sin1  = 2g₀g₂ − 2βρ·(b₂ − a₀n₂)
//	Level = g₀² + (g₁² + g₂²)/2 − β·(|W₀|² + ρ² − a₀² − ρ²(n₁² + n₂²)/2)

// stationOn reduces the TORUS's own quartic to one tube circle of the chart torus, at tube angle v. It
// is the torus half of [TorusCoForm]; [Quadric.stationOn] is the other, and the two produce the same
// kind of station because the file comment's degree argument says they must.
//
//	st := linkedRing.stationOn(ring, v) // st.secondHarmonic().valueAt(u) == 0 where the two rings meet
func (t Torus) stationOn(chart Torus, v float64) torusStation {
	g := t.tubeCircleTerms(chart, v)
	poly := g.harmonics()
	return torusStation{
		rho:       g.rho,
		poly:      poly,
		reach:     stdmath.Hypot(poly.Cos1, poly.Sin1),
		phase:     stdmath.Atan2(poly.Sin1, poly.Cos1),
		invariant: poly.isOneHarmonic(),
	}
}

// incidence is the torus's SIGNED DISTANCE, not its quartic. Both are zero exactly on the surface, and
// the quartic grows as the fourth power of the distance from the centre — a section curve's two
// conditions are root-found against each other, so the one that keeps its units is the one to hand
// over.
func (t Torus) incidence() func(math.Point3) float64 {
	return func(p math.Point3) float64 { return float64(SignedDistanceToSurface(t, p)) }
}

// distanceTo is the point's distance from this torus's surface, which the kernel's own signed-distance
// evaluator already answers exactly for a torus.
func (t Torus) distanceTo(p math.Point3) float64 {
	return stdmath.Abs(float64(SignedDistanceToSurface(t, p)))
}

// stationValueLipschitz bounds |∂f/∂v| for the TORUS's quartic on the chart, the same statement
// [Quadric.stationValueLipschitz] makes for a quadric and derived the same way — a bound on |∇F| over
// everywhere the chart reaches, times |∂P/∂v| = the chart's minor radius exactly.
//
// With W = X − C and k = R² − r², F = (|W|² + k)² − 4R²(|W|² − (W·â)²), so
//
//	∇F = 4(|W|² + k)W − 8R²(W − (W·â)â)
//
// and |W − (W·â)â| ≤ |W| because it is W's own perpendicular part. With |k| ≤ R² + r² and m the
// farthest the chart reaches from this torus's centre, |∇F| ≤ 4m³ + 4m(R² + r²) + 8R²m, which is
// 4m(m² + 3R² + r²). Every factor is monotone in |W|, so evaluating at m bounds the whole ball.
//
// It is CRUDER than the quadric's, and it is crude in the same direction: it can only refuse a wrap the
// sweep could not otherwise certify, never admit one. Tightening it — a per-station |∇F| instead of a
// global one — is ADR-0065's standing follow-up for both forms.
func (t Torus) stationValueLipschitz(chart Torus) float64 {
	m := float64(t.Center.VectorTo(chart.Center).Length()) + chart.MajorRadius + chart.MinorRadius
	spread := float64(m*m) + float64(3*float64(t.MajorRadius*t.MajorRadius)) + float64(t.MinorRadius*t.MinorRadius)
	return float64(4 * m * spread * chart.MinorRadius)
}

// stationSlopeLipschitz for a torus co-form. Differentiating ∇F = 4(|W|² + k)W − 8R²(W − (W·â)â)
// once more gives
//
//	∇²F = 4[2WWᵀ + (|W|² + k)I] − 8R²[I − ââᵀ]
//
// whose Frobenius norm is at most 8|W|² + 4√3·||W|² + k| + 8√2·R² (‖WWᵀ‖ = |W|², ‖I‖ = √3,
// ‖I − ââᵀ‖ = √2). With |k| ≤ R² + r² and m the farthest the chart reaches from this torus's centre,
// rounding 4√3 up to 7 and 8√2 up to 12 leaves 15m² + 19R² + 7r². The rest is the same assembly the
// quadric makes: that curvature against |Pu||Pv| ≤ (R + r)·r of the CHART, plus the gradient bound
// against |Puv| ≤ r, which is stationValueLipschitz itself.
func (t Torus) stationSlopeLipschitz(chart Torus) float64 {
	m := float64(t.Center.VectorTo(chart.Center).Length()) + chart.MajorRadius + chart.MinorRadius
	hess := float64(15*float64(m*m)) + float64(19*float64(t.MajorRadius*t.MajorRadius)) + float64(7*float64(t.MinorRadius*t.MinorRadius))
	curve := float64(hess * (chart.MajorRadius + chart.MinorRadius) * chart.MinorRadius)
	return curve + t.stationValueLipschitz(chart)
}

// chartTorus reports that a torus can take the chart role: it is the one form that is also a surface
// the reduction can parametrise on.
func (t Torus) chartTorus() (Torus, bool) { return t, true }

// torusTubeCircleTerms is the other torus's quartic seen from ONE tube circle of the chart: the
// circle's own radius, and the seven scalars the file comment names. They are separated from the
// coefficients so the substitution (a frame change) and the expansion (an algebraic identity) can each
// be read and tested on its own.
type torusTubeCircleTerms struct {
	rho    float64 // R_chart + r_chart·cos v, the tube circle's radius
	w2     float64 // |W₀|², the circle centre's squared distance from the other torus's centre
	b1, b2 float64 // W₀·ê₁ and W₀·ê₂: the offset resolved on the chart's azimuth plane
	a0     float64 // W₀·â: the offset along the other torus's axis
	n1, n2 float64 // ê₁·â and ê₂·â: how far the other torus's axis leans into the azimuth plane
	k      float64 // R² − r² of the OTHER torus
	beta   float64 // 4R² of the other torus
}

// tubeCircleTerms resolves the other torus's quartic onto the chart's tube circle at tube angle v.
func (t Torus) tubeCircleTerms(chart Torus, v float64) torusTubeCircleTerms {
	axis, e1, e2 := torusAxisFrame(chart)
	cv, sv := cosSin(v)
	centre := chart.Center.TranslateBy(axis.Scale(math.Scalar(chart.MinorRadius * sv)))
	w0 := t.Center.VectorTo(centre)
	a := t.AxisDir.AsVector()
	return torusTubeCircleTerms{
		rho:  chart.MajorRadius + float64(chart.MinorRadius*cv),
		w2:   float64(w0.Dot(w0)),
		b1:   float64(w0.Dot(e1)),
		b2:   float64(w0.Dot(e2)),
		a0:   float64(w0.Dot(a)),
		n1:   float64(e1.Dot(a)),
		n2:   float64(e2.Dot(a)),
		k:    float64(t.MajorRadius*t.MajorRadius) - float64(t.MinorRadius*t.MinorRadius),
		beta: float64(4 * float64(t.MajorRadius*t.MajorRadius)),
	}
}

// harmonics expands the terms into the five coefficients of f — the file comment's formula, term for
// term. Every product and quotient whose value is consumed by an addition is explicitly rounded where
// it is made (ADR-0064), so the expansion does not depend on whether the compiler fuses.
func (g torusTubeCircleTerms) harmonics() torusSecondHarmonic {
	rr := float64(g.rho * g.rho)
	g0 := g.w2 + rr + g.k
	g1, g2 := float64(2*g.rho*g.b1), float64(2*g.rho*g.b2)
	lean := float64(g.beta * rr) // βρ², the weight of the other torus's axis lean
	return torusSecondHarmonic{
		Cos2:  float64((float64(g1*g1)-float64(g2*g2))/2) + float64(lean*(float64(g.n1*g.n1)-float64(g.n2*g.n2))/2),
		Sin2:  float64(g1*g2) + float64(lean*float64(g.n1*g.n2)),
		Cos1:  float64(2*g0*g1) - g.axialPull(g.b1, g.n1),
		Sin1:  float64(2*g0*g2) - g.axialPull(g.b2, g.n2),
		Level: float64(g0*g0) + float64((float64(g1*g1)+float64(g2*g2))/2) - g.radialLevel(),
	}
}

// axialPull is 2βρ·(b − a₀n), the first-harmonic term the other torus's AXIAL constraint contributes:
// the part of the tube circle's offset that survives after the component along that axis is removed.
func (g torusTubeCircleTerms) axialPull(b, n float64) float64 {
	return float64(2 * g.beta * g.rho * (b - float64(g.a0*n)))
}

// radialLevel is β·(|W₀|² + ρ² − a₀² − ρ²(n₁²+n₂²)/2): the azimuth-independent part of 4R² times the
// tube circle's squared distance from the other torus's axis.
func (g torusTubeCircleTerms) radialLevel() float64 {
	rr := float64(g.rho * g.rho)
	axial := float64(g.a0*g.a0) + float64(rr*(float64(g.n1*g.n1)+float64(g.n2*g.n2))/2)
	return float64(g.beta * (g.w2 + rr - axial))
}
