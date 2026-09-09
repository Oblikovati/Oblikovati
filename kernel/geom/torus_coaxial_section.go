// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The COAXIAL family of the torus bucket, solved in closed form (ADR-0066, Oblikovati#3514).
//
// When the other surface's constraint on the torus carries no azimuth dependence at all, the section is
// not two azimuths but WHOLE TUBE CIRCLES, at the tube angles where the constraint's level term
// vanishes. Which tube angles those are, and how many there are, is a TOPOLOGICAL question — it is the
// number of curves the section has — and it is answered here by solving, never by sampling.
//
// It used to be answered by scanning the level at 720 tube angles and bisecting each sign change 60
// times. That is the shape ADR-0065's review found fatal one bucket over: a topological question
// decided from a fixed grid is wrong whenever the feature is narrower than the grid, and no amount of
// bisection afterwards recovers a crossing the scan stepped over. The level is a low-degree
// trigonometric polynomial in the tube angle for both co-forms, so there is nothing to sample.
//
// THE QUADRIC. Level(v) = constant(v) + ρ(v)²(m₁₁+m₂₂)/2, with ρ = R + r·cos v and the station offset
// W₀(v) = w + r·sin v·â. Expanding W₀·MW₀ + 2G·W₀ + K in s = r·sin v leaves a term in s, a term in s²
// and a constant; ρ² contributes a term in cos v and one in cos²v. So Level is a degree-two
// trigonometric polynomial in v — the same shape [torusSecondHarmonic] carries in the azimuth, read by
// the same exact quartic solver.
//
// THE TORUS. Its level factors, and each factor is a FIRST harmonic. With d the axial offset between
// the two centres, a₀(v) = d + r·sin v and ρ(v) = R + r·cos v,
//
//	Level = ((ρ − R_b)² + a₀² − r_b²) · ((ρ + R_b)² + a₀² − r_b²)
//
// and in each factor the r²cos²v and r²sin²v collapse to the constant r², leaving
//
//	F±(v) = (R ± R_b)² + d² + r² − r_b² + 2(R ± R_b)·r·cos v + 2d·r·sin v
//
// which is level + reach·cos(v − phase): an ARCCOS, exact, two roots or none. The two factors are the
// two halves of the other torus's meridian circle — the near one and the one its tube reaches past its
// own axis — so solving both is what makes this right for a spindle torus as well as a ring.

// torusCoaxialCircles returns the tube circles where a co-form COAXIAL with the torus meets it: one
// circle per tube angle at which the level term CROSSES zero. A tube angle the level only grazes is a
// tangency, one contact rather than two crossings, and carries no section.
//
// ok=false is a co-form whose level this cannot solve, which no member of [TorusCoForm] is — it is the
// refusal a third member would have to earn.
func torusCoaxialCircles(t Torus, co TorusCoForm) ([]Curve3, bool) {
	factors := co.coaxialLevelFactors(t)
	if len(factors) == 0 {
		return nil, false
	}
	var out []Curve3
	for _, v := range crossingTubeAngles(factors) {
		out = append(out, torusStationCircle(t, v))
	}
	return out, true
}

// crossingTubeAngles is every tube angle at which some factor vanishes and the LEVEL changes sign
// there, in one ascending order with coincident angles collapsed.
//
// A root is kept on its own factor's slope. The level is the product of the factors, so at a simple
// root of one factor the level crosses exactly when the others do not vanish too — and when two do, the
// angle is a double root of the level, which is a tangency and drops out either way. Reading the slope
// on the factor rather than on the product is what keeps the test exact: a product of two polynomials
// evaluated near a common root is the difference of two nearly equal numbers.
func crossingTubeAngles(factors []torusSecondHarmonic) []float64 {
	var out []float64
	for _, f := range factors {
		for _, v := range f.azimuths() {
			if stdmath.Abs(f.slopeAt(v)) > torusLevelCrossingTol*f.scale() {
				out = append(out, v)
			}
		}
	}
	return sortedDedupedAngles(out)
}

// torusLevelCrossingTol is how steeply the level must pass through zero to count as CROSSING it rather
// than touching it. It compares a slope with the polynomial's own largest coefficient, both in the
// polynomial's units, so it is dimensionless and carries no model scale — the same pair written in
// metres and in millimetres classifies identically.
//
// It is the SAME relative floor a candidate root is certified against ([torusRootResidualTol]), and
// deliberately so: a root whose residual is admitted at that level is not distinguishable from a
// tangency by a slope any smaller than it.
const torusLevelCrossingTol = torusRootResidualTol

// coaxialLevelFactors returns the ONE polynomial in the tube angle whose roots are the tube angles at
// which a coaxial quadric meets the torus (see the file comment's expansion).
func (q Quadric) coaxialLevelFactors(t Torus) []torusSecondHarmonic {
	axis, _, _ := torusAxisFrame(t)
	w := q.Anchor.VectorTo(t.Center)
	mw, ma := q.M.Apply(w), q.M.Apply(axis)
	// Level(v) = base + linear·sin v + square·sin²v + ρ(v)²·half, with ρ = R + r·cos v.
	base := float64(w.Dot(mw)) + float64(2*float64(q.G.Dot(w))) + q.K
	linear := float64(t.MinorRadius * (float64(2*float64(axis.Dot(mw))) + float64(2*float64(q.G.Dot(axis)))))
	square := float64(float64(t.MinorRadius*t.MinorRadius) * float64(axis.Dot(ma)))
	_, e1, e2 := torusAxisFrame(t)
	m11, m22, _ := inPlaneTensorEntries(q, e1, e2)
	return []torusSecondHarmonic{sumTubeAnglePolys(
		sinSquaredPoly(base, linear, square),
		radiusSquaredPoly(t, float64((m11+m22)/2)),
	)}
}

// coaxialLevelFactors returns the TWO first-harmonic factors of a coaxial torus's level: the near half
// of its meridian circle and the half its tube reaches past its own axis.
func (t Torus) coaxialLevelFactors(chart Torus) []torusSecondHarmonic {
	axis, _, _ := torusAxisFrame(chart)
	d := float64(t.Center.VectorTo(chart.Center).Dot(axis)) // axial offset between the two centres
	r := chart.MinorRadius
	out := make([]torusSecondHarmonic, 0, 2)
	for _, sign := range [2]float64{1, -1} {
		reach := chart.MajorRadius + float64(sign*t.MajorRadius)
		out = append(out, torusSecondHarmonic{
			Level: float64(reach*reach) + float64(d*d) + float64(r*r) - float64(t.MinorRadius*t.MinorRadius),
			Cos1:  float64(2 * reach * r),
			Sin1:  float64(2 * d * r),
		})
	}
	return out
}

// sinSquaredPoly writes base + linear·sin v + square·sin²v as a tube-angle polynomial, folding the
// square through sin²v = (1 − cos 2v)/2.
func sinSquaredPoly(base, linear, square float64) torusSecondHarmonic {
	return torusSecondHarmonic{
		Level: base + float64(square/2),
		Sin1:  linear,
		Cos2:  float64(-square / 2),
	}
}

// radiusSquaredPoly writes weight·ρ(v)² as a tube-angle polynomial, with ρ = R + r·cos v folded through
// cos²v = (1 + cos 2v)/2.
func radiusSquaredPoly(t Torus, weight float64) torusSecondHarmonic {
	rr := float64(t.MinorRadius * t.MinorRadius)
	return torusSecondHarmonic{
		Level: float64(weight * (float64(t.MajorRadius*t.MajorRadius) + float64(rr/2))),
		Cos1:  float64(weight * float64(2*float64(t.MajorRadius*t.MinorRadius))),
		Cos2:  float64(weight * float64(rr/2)),
	}
}

// sumTubeAnglePolys adds two tube-angle polynomials coefficient by coefficient.
func sumTubeAnglePolys(a, b torusSecondHarmonic) torusSecondHarmonic {
	return torusSecondHarmonic{
		Cos2: a.Cos2 + b.Cos2, Sin2: a.Sin2 + b.Sin2,
		Cos1: a.Cos1 + b.Cos1, Sin1: a.Sin1 + b.Sin1,
		Level: a.Level + b.Level,
	}
}
