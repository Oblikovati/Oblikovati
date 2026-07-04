// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Disjoint gate for the partial-rim second cut (#1732). The first shippable sub-family is the DISJOINT case:
// the new SSI imprint lies entirely in the still-full part of the band, clear of the prior notch, so the
// slice-1 builder reuses without solving 3-curve arrangement vertices. disjointFromPrior decides ship vs
// decline; every non-disjoint config (imprint crossing the notch, tangent to it, enclosing it, wrapping the
// whole azimuth) declines to CSG, which the caller records observably.
//
// The test is deliberately small because belowPrior over the imprint samples already subsumes crossing AND
// notch-containment AND notch-enclosure: an imprint whose every point survived the first cut cannot cross into
// the removed notch, lie inside it, or surround it (the notch is above the surviving boundary). What remains is
// a 3D near-approach margin (a tangent imprint stays technically below but must decline before the arrangement
// welds two within-tol boundaries) and the contractible/seam-clear precondition.

// disjointMarginRel scales the near-approach margin to the rim radius, floored well above the weld grid so a
// grazing imprint declines before the arrangement can merge it with the prior boundary (#1732).
const disjointMarginRel = 1e-3

// minWrapGap: the new imprint must leave at least this azimuthal gap (radians); a loop covering the whole
// circle is non-contractible, so (u,v) containment is undefined and it is out of the disjoint sub-family.
const minWrapGap = 0.05

// priorSampleCount samples each prior edge for the 3D near-approach test — dense enough that the chord error is
// far below the margin.
const priorSampleCount = 64

// disjointFromPrior reports whether the new imprint is disjoint from the surviving prior boundary — the ship
// condition for the partial-rim second cut. It assumes placeSeams has run (u is seam-relative).
func (c *cutCylinderUV) disjointFromPrior(imprint []geom.Curve3) bool {
	impUV := c.imprintUV(imprint)
	if len(impUV) == 0 || c.imprintWrapsAzimuth(impUV) {
		return false
	}
	prior := c.priorUVSegments()
	for _, s := range impUV {
		if !belowPrior(prior, s.a) || !belowPrior(prior, s.b) {
			return false // the imprint enters the removed notch (crosses it, lies in it, or surrounds it)
		}
	}
	return c.clearOfPrior(imprint)
}

// imprintUV samples the new imprint into seam-relative (u,v) segments.
func (c *cutCylinderUV) imprintUV(imprint []geom.Curve3) []uvSeg {
	var out []uvSeg
	for _, cv := range imprint {
		out = append(out, c.sampleImprintUV(cv)...)
	}
	return out
}

// imprintWrapsAzimuth reports whether the imprint covers the whole circle with no gap >= minWrapGap — a
// non-contractible belt cut, outside the disjoint sub-family.
func (c *cutCylinderUV) imprintWrapsAzimuth(impUV []uvSeg) bool {
	us := make([]float64, 0, len(impUV))
	for _, s := range impUV {
		us = append(us, float64(s.a.X))
	}
	return widestGap(us) < minWrapGap
}

// clearOfPrior reports whether every sampled new-imprint point is at least the model-scaled margin (3D) from
// the prior boundary — so a tangent or near-coincident imprint declines before the arrangement welds the two.
func (c *cutCylinderUV) clearOfPrior(imprint []geom.Curve3) bool {
	prior3D := c.priorSegments3D()
	margin := stdmath.Max(disjointMarginRel*c.band.rBot, 1e3*planarStitchGrid)
	for _, cv := range imprint {
		for i := 0; i <= imprintSampleCount; i++ {
			if minDistToSegs3D(cv.PointAt(float64(i)/imprintSampleCount), prior3D) < margin {
				return false
			}
		}
	}
	return true
}

// priorSegments3D samples the prior loop into 3D segments for the near-approach test.
func (c *cutCylinderUV) priorSegments3D() [][2]math.Point3 {
	var out [][2]math.Point3
	for _, e := range c.prior.edges {
		prev := e.curve.PointAt(e.t0)
		for i := 1; i <= priorSampleCount; i++ {
			p := e.curve.PointAt(e.t0 + (e.t1-e.t0)*float64(i)/float64(priorSampleCount))
			out = append(out, [2]math.Point3{prev, p})
			prev = p
		}
	}
	return out
}

// minDistToSegs3D is the minimum distance from p to any of the 3D segments.
func minDistToSegs3D(p math.Point3, segs [][2]math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, s := range segs {
		if d := distPointSegment(p, s[0], s[1]); d < best {
			best = d
		}
	}
	return best
}

// widestGap returns the largest azimuthal gap (radians) between consecutive angles around the circle.
func widestGap(angles []float64) float64 {
	if len(angles) < 2 {
		return 2 * stdmath.Pi
	}
	sort.Float64s(angles)
	twoPi := 2 * stdmath.Pi
	gap := angles[0] + twoPi - angles[len(angles)-1]
	for i := 1; i < len(angles); i++ {
		if g := angles[i] - angles[i-1]; g > gap {
			gap = g
		}
	}
	return gap
}
