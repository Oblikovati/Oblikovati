// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Knot-structure-aware seeding for NURBS point/surface inversion (Oblikovati#1608).
// Newton point inversion (Piegl & Tiller §6.1) is only LOCALLY convergent: it walks to
// the foot in whose basin the seed already lies. A seed grid with fewer nodes than the
// surface has knot spans skips whole spans, so on a high-span import (STEP surfaces
// routinely carry dozens of spans) the polish locks onto the wrong foot and reports the
// wrong signed distance. Seeding PER KNOT SPAN fixes this: candidate feet cluster near
// knots, and within one span the surface is a single polynomial of degree p that can
// turn at most p−1 times, so p+1 samples inside every span resolve its bending and no
// span is ever skipped. This mirrors OCCT's Extrema_GenExtPS (samples per knot interval)
// and the Greville-abscissa density Piegl & Tiller recommend for span-aware sampling.

// knotSpanSeedParams returns point-inversion seed parameters over a knot vector's clamped
// domain, placing degree+1 samples inside every distinct knot span. A surface with S
// spans is thus seeded S·(degree+1)+1 times in that direction — dense where it can bend,
// never skipping a span the way the retired fixed 16×16 / 24 grid did (#1608).
func knotSpanSeedParams(knots []float64, degree int) []float64 {
	breaks := domainBreakpoints(knots, degree)
	per := degree + 1
	out := []float64{breaks[0]}
	for i := 0; i+1 < len(breaks); i++ {
		a, b := breaks[i], breaks[i+1]
		for k := 1; k <= per; k++ {
			out = append(out, a+(b-a)*float64(k)/float64(per))
		}
	}
	return out
}

// domainBreakpoints returns the distinct knot values that bound the spans of the clamped
// domain [knots[degree], knots[len−1−degree]], both ends included and sorted ascending.
func domainBreakpoints(knots []float64, degree int) []float64 {
	lo, hi := knots[degree], knots[len(knots)-1-degree]
	out := []float64{lo}
	for _, k := range knots {
		if k > lo+knotEps && k < hi-knotEps && !containsKnot(out, k) {
			out = append(out, k)
		}
	}
	out = append(out, hi)
	sortFloats(out)
	return out
}

// nearestSeed returns the (u, v) from the us×vs seed lattice whose surface point is
// closest to q — the Gauss–Newton starting guess for point inversion.
func nearestSeed(s Surface, q math.Point3, us, vs []float64) (float64, float64) {
	bu, bv, bd := us[0], vs[0], stdmath.Inf(1)
	for _, u := range us {
		for _, v := range vs {
			if d := s.PointAt(u, v).DistanceSquaredTo(q); d < bd {
				bu, bv, bd = u, v, d
			}
		}
	}
	return bu, bv
}

// minSeedSpacing returns the smallest positive gap between consecutive sorted seed params
// — the natural cluster-merge radius for distinguishing multiple feet in one neighbourhood.
func minSeedSpacing(params []float64) float64 {
	gap := stdmath.Inf(1)
	for i := 1; i < len(params); i++ {
		if d := params[i] - params[i-1]; d > 0 && d < gap {
			gap = d
		}
	}
	if stdmath.IsInf(gap, 1) {
		return 1
	}
	return gap
}
