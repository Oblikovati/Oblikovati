// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Clipping a wall-versus-wall crossing to the wall's band (ADR-0061 stage 4).
//
// A crossing that leaves the wall through a RIM used to decline the whole boolean. It need not: the
// stretch between the rims is real imprint, and the rim circle is already in the face's own frame, so
// the arrangement closes the region the clipped arc opens. That is what the hand-written rim-crossing
// recognizer assembled by hand as an "exit chain" — a wall arc plus a rim arc — and what the general
// pipeline gets for free once the crossing is clipped instead of refused.

// clipCrossingToBand returns the stretches of a crossing that lie between the wall's rims. ok=false when
// a crossing the placement called a straddle turns out to have nothing inside the band, which the two
// answers cannot both be true of.
func clipCrossingToBand(cv geom.Curve3, rs ruledSide) ([]geom.Curve3, bool) {
	spans := bandParamSpans(cv, rs)
	if len(spans) == 0 {
		return nil, false
	}
	out := make([]geom.Curve3, 0, len(spans))
	for _, sp := range spans {
		out = append(out, geom.TrimmedCurve3{Base: cv, Lo: sp[0], Hi: sp[1]})
	}
	return out, true
}

// bandParamSpans brackets every rim crossing on a walk of the curve and bisects each bracket to the
// root, returning the parameter spans whose points lie between the rims.
//
// A span that WRAPS the curve's domain end comes back as two spans meeting at that end. The end of a
// closed analytic curve is an artificial split already — the same one the chart's seam machinery
// resolves — so the extra vertex costs the imprint nothing.
func bandParamSpans(cv geom.Curve3, rs ruledSide) [][2]float64 {
	t0, t1 := cv.Domain()
	depth := func(t float64) float64 { return bandDepth(cv.PointAt(t), rs) }
	var spans [][2]float64
	start, in, prevT := t0, depth(t0) >= 0, t0
	for i := 1; i <= crossingSpanSamples; i++ {
		t := t0 + (t1-t0)*float64(i)/crossingSpanSamples
		if (depth(t) >= 0) == in {
			prevT = t
			continue
		}
		root := bisectRoot(depth, prevT, t)
		if in {
			spans = append(spans, [2]float64{start, root})
		} else {
			start = root
		}
		in, prevT = !in, t
	}
	if in {
		spans = append(spans, [2]float64{start, t1})
	}
	return spans
}

// bandDepth is how far inside the wall's band a point sits: positive between the rims, negative past
// either of them, zero on one. It is the one continuous function whose roots ARE the rim crossings.
func bandDepth(p math.Point3, rs ruledSide) float64 {
	v := bandV(p, rs.axis, rs.band)
	return stdmath.Min(v-rs.band.vMin, rs.band.vMax-v)
}
