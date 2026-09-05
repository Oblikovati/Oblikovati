// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// cutCylinderUV is the (u,v) model of an ALREADY-CUT cylinder side under a SECOND curved boolean (#1732). It
// embeds the bare-side ruledUV for the geometry frame, seam, point3, re-emission and orientation, and adds the
// surviving prior-trim loop. The second cut's arrangement subdivides by {new imprint, bottom rim, seam, PRIOR
// loop} — the prior loop replaces the removed top rim as constraint edges — and a cell is kept iff it is BELOW
// the prior boundary (survived the first cut) AND selected by the second cut. Only the segment assembly and
// the material predicate differ from ruledUV; the bare-side path is untouched (its golden is byte-identical),
// so the two compose cleanly.
type cutCylinderUV struct {
	ruledUV
	prior priorTrimLoop
}

var _ uvSide = (*cutCylinderUV)(nil)

// cutEdgeVSpanRel: an edge spanning more than this fraction of the band height is a first-cut section boundary
// (v-varying), not a near-flat rim arc the seam may safely cross.
const cutEdgeVSpanRel = 1e-3

// newCutCylinderUVSolid frames an already-cut cylinder side cut by another SOLID's imprint: the ruledUV solid
// frame over the recovered band, plus the prior loop the arrangement composes as constraint edges (#1732).
func newCutCylinderUVSolid(cyl geom.Cylinder, band coneSideBand_, prior priorTrimLoop, op Op, isB bool, inside func(math.Point3) bool) cutCylinderUV {
	return cutCylinderUV{ruledUV: newCylinderUVSolid(cyl, band, op, isB, inside), prior: prior}
}

// placeSeams moves the artificial azimuth seam clear of BOTH the new imprint and the prior loop's CUT boundary
// (its v-varying edges — the first cut's section conic), so the seam crosses only the smooth surviving rim arc,
// never a notch vertex or the new imprint (uvSide, #1732). The rim arc is a legitimate seam crossing, exactly
// as the bare side's top rim is.
func (c *cutCylinderUV) placeSeams(imprint []geom.Curve3) {
	c.seamU = c.chooseSeamU(append(append([]geom.Curve3{}, imprint...), c.priorCutCurves()...))
}

// priorCutCurves returns the prior loop's v-varying edges (the first cut's section boundary), excluding the
// near-constant-v rim arcs the seam may safely cross.
func (c *cutCylinderUV) priorCutCurves() []geom.Curve3 {
	var out []geom.Curve3
	for _, e := range c.prior.edges {
		if c.edgeVSpan(e) > cutEdgeVSpanRel*c.band.vMax {
			out = append(out, e.curve)
		}
	}
	return out
}

// edgeVSpan is the axial (v) extent an edge covers — near zero for a rim arc, large for a section conic.
func (c *cutCylinderUV) edgeVSpan(e loopEdge) float64 {
	lo, hi := priorAxialRange(priorTrimLoop{edges: []loopEdge{e}}, c.band.bottom, c.axis)
	return hi - lo
}

// assembleSegments builds the second cut's tagged (u,v) segment set: the new imprint (clipped to the band) +
// the prior loop as constraint edges + the bottom rim + the seam — but NO top rim (the prior loop is the top
// boundary now). uvSide override; the bare ruledUV assembly is unchanged (#1732).
func (c *cutCylinderUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	var imp []uvSeg
	for _, cv := range imprint {
		imp = append(imp, c.sampleImprintUV(cv)...)
	}
	segs := c.clipImprintToBand(imp)
	segs = append(segs, c.priorUVSegments()...)
	return append(segs, c.cutFrameSegments()...)
}

// priorUVSegments samples the prior loop into tagged (u,v) segments, seam-split so none spans the azimuth
// discontinuity and dropped where welded to zero length. They re-emit to their own analytic curves so the kept
// face's top boundary follows the prior loop exactly. This same segment set is the polyline belowPrior tests.
func (c *cutCylinderUV) priorUVSegments() []uvSeg {
	var out []uvSeg
	for _, e := range c.prior.edges {
		for _, s := range c.sampleRange(e.curve, e.t0, e.t1) {
			for _, split := range splitSeamCrossing(s) {
				if split.a.DistanceTo(split.b) > arrTol {
					out = append(out, split)
				}
			}
		}
	}
	return out
}

// cutFrameSegments is bandFrameSegments MINUS the top rim: a cut side has no full top circle, so only the
// bottom rim (v=0, the surviving anchor) and the two seam verticals close the rectangle; the prior loop is the
// top boundary (#1732).
func (c *cutCylinderUV) cutFrameSegments() []uvSeg {
	twoPi := 2 * stdmath.Pi
	seam := geom.NewLineSegment(c.point3(0, c.band.vMin), c.point3(0, c.band.vMax))
	return []uvSeg{
		{a: math.P2(0, c.band.vMin), b: math.P2(twoPi, c.band.vMin), curve: c.band.bottomCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMin), b: math.P2(0, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(twoPi, c.band.vMin), b: math.P2(twoPi, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
	}
}

// cutCylinderMaterial composes the survival predicate (below the prior boundary — the cell outlived the first
// cut) with the second cut's own material test (solid membership or plane half-space), read live off the
// seam-shifted receiver after placeSeams (#1732).
func cutCylinderMaterial(c *cutCylinderUV) func() materialPredicate {
	return func() materialPredicate {
		poly := c.priorUVSegments()
		base := c.baseMaterial()
		return func(uv math.Point2) bool { return belowPrior(poly, uv) && base(uv) }
	}
}

// baseMaterial returns the second cut's own predicate: solid membership in solidMode, else the plane half-space.
func (c *cutCylinderUV) baseMaterial() materialPredicate {
	if c.solidMode {
		return func(uv math.Point2) bool { return c.keptBySolid(uv.X, uv.Y) }
	}
	return c.halfSpaceMaterial()
}

// belowPrior reports whether a (u,v) point survived the first cut: it lies BELOW the prior top boundary. An
// upward v-ray from the point crosses the prior loop an odd number of times iff the point is under it — the
// prior loop is the side's top boundary spanning all azimuth, so the ray exits the band exactly through it.
// This is the disjoint sub-family's survival test; a prior loop that is a detached interior lens (not the top
// boundary) is out of scope and gated out upstream (#1732).
func belowPrior(poly []uvSeg, uv math.Point2) bool {
	return priorRayCrossings(poly, float64(uv.X), float64(uv.Y))%2 == 1
}

// priorRayCrossings counts prior-loop segments an upward v-ray at azimuth qu, from height qv, passes through.
// Half-open in u ([lo,hi)) so a shared vertex between two segments is counted once.
func priorRayCrossings(poly []uvSeg, qu, qv float64) int {
	n := 0
	for _, s := range poly {
		lo, hi := float64(s.a.X), float64(s.b.X)
		ylo, yhi := float64(s.a.Y), float64(s.b.Y)
		if lo > hi {
			lo, hi, ylo, yhi = hi, lo, yhi, ylo
		}
		if lo == hi || qu < lo || qu >= hi {
			continue
		}
		if ylo+(yhi-ylo)*(qu-lo)/(hi-lo) > qv {
			n++
		}
	}
	return n
}
