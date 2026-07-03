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

// seedMinSamples floors the per-direction seed count so a LOW-span surface (a single Bézier
// patch) is still sampled as densely as the retired fixed grid; a high-span surface exceeds
// it naturally at one sample per span. This is the issue's "grid = max(16, spans+1)" (#1608).
const seedMinSamples = 16

// knotSpanSeedParams returns point-inversion seed parameters over a knot vector's clamped
// domain, placing at least one sample inside every distinct knot span so no span is skipped,
// and enough per span that the total reaches seedMinSamples on a low-span surface. A surface
// with S spans is seeded ≈max(16, S)+1 times — the issue's span-aware grid (#1608), Θ(S) not
// the retired fixed grid, so the cost matches the old 16/24 grid on ordinary surfaces.
func knotSpanSeedParams(knots []float64, degree int) []float64 {
	breaks := domainBreakpoints(knots, degree)
	spans := len(breaks) - 1
	per := (seedMinSamples + spans - 1) / spans // ceil(min/spans): ≥1, larger only when spans<16
	out := []float64{breaks[0]}
	for i := 0; i < spans; i++ {
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

// ssiSpanCounter reports a surface's distinct-knot-span counts so the SSI seeder can start its
// quadtree at ≈one coarse cell per span (OCCT Extrema_GenExtPS samples per knot interval, #1608).
// The retired fixed 8×8 coarse grid put several knot spans in one cell on a high-span surface, so
// many parallel intersection curves aliased to a single two-edge crossing and all but one were
// silently dropped. A span-sized coarse cell holds at most a few crossings, so the transversal
// shortcut is sound again without forcing the whole domain to refine. No knot structure ⇒ (0,0).
type ssiSpanCounter interface {
	knotSpanCounts() (uSpans, vSpans int)
}

// knotSpanCounts returns the number of distinct knot spans in each parameter direction.
func (s BSplineSurface) knotSpanCounts() (int, int) {
	return len(domainBreakpoints(s.UKnots, s.UDegree)) - 1, len(domainBreakpoints(s.VKnots, s.VDegree)) - 1
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
