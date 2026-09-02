// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Phase 2 of the ruled-side (u,v) arrangement (#2212): frame the band and clip the imprint
// into it.
//
// The band is the side's own trim in parameter space. Clipping the sampled imprint to it, and
// choosing where to put the azimuth seam (the widest imprint-free gap, so the seam does not cut
// through the region of interest), is what makes the subdivision that follows well posed. The
// material predicate that phase 3 classifies cells with is decided here too, because it is a
// property of the half-space being cut, not of any one cell.

// bandFrameSegments returns the four straight (u,v) edges that bound the parameter rectangle for the
// arrangement: the bottom rim (v=vMin) and top rim (v=vMax) — each a single horizontal segment in (u,v),
// tagged to its rim circle — and the two seam verticals (u=0 and u=2π), tagged to the shared seam ruling.
// The rims are genuine surface boundaries; the seam verticals are artificial and dissolve where the kept
// region wraps them (resolved when cells are merged across the seam).
func (c ruledUV) bandFrameSegments() []uvSeg {
	twoPi := 2 * stdmath.Pi
	seam := geom.NewLineSegment(c.point3(0, c.band.vMin), c.point3(0, c.band.vMax))
	return []uvSeg{
		{a: math.P2(0, c.band.vMin), b: math.P2(twoPi, c.band.vMin), curve: c.band.bottomCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMax), b: math.P2(twoPi, c.band.vMax), curve: c.band.topCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMin), b: math.P2(0, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(twoPi, c.band.vMin), b: math.P2(twoPi, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
	}
}

// assembleBandSegments builds the full tagged (u,v) segment set the arrangement subdivides: every imprint
// segment (seam-split so none spans the azimuth discontinuity) plus the rim+seam frame that closes the
// parameter rectangle. Degenerate segments (both endpoints welded by the seam split) are dropped.
func (c ruledUV) assembleBandSegments(imprint []uvSeg) []uvSeg {
	return append(c.clipImprintToBand(imprint), c.bandFrameSegments()...)
}

// clipImprintToBand seam-splits every imprint segment so none spans the azimuth discontinuity, clips each to
// the band's axial range (a tilted cut's ellipse can rise past the rim — sampling that out-of-band part would
// inject a spurious arc; clipping lands the imprint exactly on the rim where it crosses), and drops segments
// welded to zero length. Extracted from assembleBandSegments so the already-cut side (cutCylinderUV) reuses
// the identical imprint clip while supplying its own prior-boundary frame instead of a top rim (#1732).
func (c ruledUV) clipImprintToBand(imprint []uvSeg) []uvSeg {
	out := make([]uvSeg, 0, len(imprint)+4)
	for _, s := range imprint {
		for _, split := range splitSeamCrossing(s) {
			for _, clipped := range c.clipSegToVBand(split) {
				if clipped.a.DistanceTo(clipped.b) > arrTol {
					out = append(out, clipped)
				}
			}
		}
	}
	return out
}

// clipSegToVBand clips a (u,v) imprint segment to the axial band [vMin,vMax], returning the in-band part or
// nothing when the segment lies wholly outside on one side. Crucially, an endpoint that leaves the band is
// snapped to the EXACT curve parameter where the imprint crosses the rim (not a linear interpolation of the
// sampled segment), so the rim-crossing 3D point equals plane∩rim and welds with the cap's matching arc —
// the section ellipse is plane∩cylinder, so its v=rim crossing is exactly plane∩rim (#1405).
func (c ruledUV) clipSegToVBand(s uvSeg) []uvSeg {
	vMin, vMax := c.band.vMin, c.band.vMax
	a, b := float64(s.a.Y), float64(s.b.Y)
	if (a < vMin && b < vMin) || (a > vMax && b > vMax) {
		return nil
	}
	sa, ta := c.clipEndToBand(s, true)
	sb, tb := c.clipEndToBand(s, false)
	if sa.DistanceTo(sb) <= arrTol {
		return nil
	}
	return []uvSeg{{a: sa, b: sb, curve: s.curve, tA: ta, tB: tb, kind: s.kind}}
}

// clipEndToBand returns one endpoint of a segment clipped to the band: the endpoint unchanged when already
// in [vMin,vMax], else the (u,v) and curve parameter where the imprint curve exactly reaches the nearer rim
// (refined on the curve, between this end's parameter and the other's).
func (c ruledUV) clipEndToBand(s uvSeg, isA bool) (math.Point2, float64) {
	p, v, t, tOther := s.a, float64(s.a.Y), s.tA, s.tB
	if !isA {
		p, v, t, tOther = s.b, float64(s.b.Y), s.tB, s.tA
	}
	if v >= c.band.vMin && v <= c.band.vMax {
		return p, t
	}
	vLim := c.band.vMin
	if v > c.band.vMax {
		vLim = c.band.vMax
	}
	tc := c.refineCurveV(s.curve, t, tOther, vLim)
	return c.paramOf(s.curve.PointAt(tc)), tc
}

// refineCurveV bisects the imprint curve parameter between tOut (outside the band) and tIn (inside) to the
// point where the curve's axial coordinate v equals vLim — the exact rim crossing.
func (c ruledUV) refineCurveV(curve geom.Curve3, tOut, tIn, vLim float64) float64 {
	out := tOut
	for range 50 {
		tm := (tOut + tIn) / 2
		if (c.curveV(curve, out)-vLim <= 0) == (c.curveV(curve, tm)-vLim <= 0) {
			tOut = tm
		} else {
			tIn = tm
		}
	}
	return (tOut + tIn) / 2
}

// curveV is the axial coordinate v of a 3D imprint curve at parameter t.
func (c ruledUV) curveV(curve geom.Curve3, t float64) float64 {
	return float64(c.paramOf(curve.PointAt(t)).Y)
}

// materialPredicate reports whether a (u,v) point of the band is on the KEPT side of the imprint. For a
// plane cut it is the half-space membership g(u,v) = a(u)+v·b(u) < 0 (the same {g<0} the analytic walk
// keeps); a general curved∩curved imprint supplies its own inside/outside test. The arrangement is built
// once and classified through this predicate, so the two cases share the whole pipeline (#1405).
type materialPredicate func(uv math.Point2) bool

// halfSpaceMaterial is the plane-cut predicate: a band point is kept where the signed distance g(u,v) is
// negative, exactly the {g<0} region the single-valued walk traced as a v-interval. It is passed to
// trimByImprint as a builder (ruledUV.halfSpaceMaterial) so it is bound to the seam-shifted frame.
func (c ruledUV) halfSpaceMaterial() materialPredicate {
	return func(uv math.Point2) bool { return c.aU(uv.X)+uv.Y*c.bU(uv.X) < 0 }
}

// chooseSeamU returns an azimuth for the arrangement's artificial seam that is clear of the imprint: the
// midpoint of the widest gap between the imprint's azimuths (the wrap gap included). Placing the seam there
// keeps a section arm from grazing it, which would otherwise collapse the (u,v) arrangement (#1405). With
// no imprint, or one that covers every azimuth, it returns 0 (the default seam).
func (c ruledUV) chooseSeamU(imprint []geom.Curve3) float64 {
	if c.hasSeamHint {
		return c.seamHint // the recogniser pins the seam (the pinched imprint has no clear gap to find)
	}
	var us []float64
	for _, cv := range imprint {
		for _, s := range c.sampleImprintUV(cv) { // c.seamU is still 0 here, so these are absolute azimuths
			us = append(us, float64(s.a.X))
		}
	}
	return widestGapMid(us)
}

// widestGapMid returns the midpoint of the widest circular gap (the wrap gap included) between the given
// angles in [0, 2π) — the clearest place to put an artificial periodic seam. Empty (or a single value)
// returns 0, the default seam. Shared by the ruled azimuth seam and both torus seams (#1406).
func widestGapMid(angles []float64) float64 {
	if len(angles) == 0 {
		return 0
	}
	sort.Float64s(angles)
	twoPi := 2 * stdmath.Pi
	bestGap, bestMid := angles[0]+twoPi-angles[len(angles)-1], (angles[len(angles)-1]+angles[0]+twoPi)/2
	for i := 1; i < len(angles); i++ {
		if g := angles[i] - angles[i-1]; g > bestGap {
			bestGap, bestMid = g, (angles[i]+angles[i-1])/2
		}
	}
	for bestMid >= twoPi {
		bestMid -= twoPi
	}
	return bestMid
}
