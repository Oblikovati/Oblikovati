// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ruledFaceUV is the (u,v) chart of a ruled wall FRAMED BY ITS OWN LOOPS (ADR-0060) for the mixed
// per-face boolean. The band charts frame a side by two synthetic constant-v rims and a seam; this one
// samples the face's boundary edges into the arrangement with every frame×imprint and frame×seam
// incidence solved in closed form and injected as a shared vertex, exactly as planeFaceUV does for a
// planar face — so an oblique rim, a notch left by an earlier cut, or a partial patch is trimmed by the
// same arrangement as a bare band, and the kept cells are those inside the frame (even-odd over the
// sampled boundary) that the boolean's keep table selects. The embedded ruledUV supplies the surface
// frame, the seam-relative parameterisation and the solid-membership predicate; nothing of its band
// frame is read.
type ruledFaceUV struct {
	ruledUV
	face      curvedFace
	frame     geom.RuledFrame
	res       geom.Resolution
	imprint   []geom.Curve3   // the imprint being trimmed, for seam-incidence lookups
	crossings []frameCrossing // frame×imprint incidences, in curve parameters (seam-independent)
	frameSegs []uvSeg         // the seam-relative sampled frame, filled by assembleSegments
	wrapping  bool            // some kept boundary loop wraps the azimuth (set by wrappingSolidFaces)
}

var _ uvSide = (*ruledFaceUV)(nil)

// newRuledFaceUV frames a wall for a solid-membership trim under op (isB marks it the boolean's B).
func newRuledFaceUV(f curvedFace, rs ruledSide, op Op, isB bool, inside func(math.Point3) bool) *ruledFaceUV {
	c := newRuledUVFrame(rs.frame.Base, rs.frame.Axis, rs.frame.Ref, rs.frame.RadSlope, rs.frame.RadConst, rs.band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return &ruledFaceUV{ruledUV: c, face: f, frame: rs.frame, res: geom.ResolutionForSize(rs.size())}
}

// admits solves every frame×imprint incidence up front and returns the imprint the chart carries: an
// imprint that COINCIDES with a frame edge (two sections in one plane are one conic on the surface) is
// a boundary contact, not a split — a coplanar tool face resting on a rim — and is dropped, exactly as
// the polygonal split drops a segment lying on its own boundary.
func (c *ruledFaceUV) admits(imprint []geom.Curve3) []geom.Curve3 {
	kept := make([]geom.Curve3, 0, len(imprint))
	for _, imp := range imprint {
		if !c.coincidesWithFrame(imp) {
			kept = append(kept, imp)
		}
	}
	c.crossings, _ = c.solveFrameCrossings(kept)
	return kept
}

// coincidesWithFrame reports an imprint section lying in the plane of one of the face's own edges.
func (c *ruledFaceUV) coincidesWithFrame(imp geom.Curve3) bool {
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			if _, coincident := geom.SectionCrossingCandidates(c.face.surface, e.curve, imp); coincident {
				return true
			}
		}
	}
	return false
}

// placeSeams moves the azimuth seam clear of the imprint AND of every frame vertex and ruling edge, so
// it crosses the frame only through the interior of a smooth section edge — where the crossing has a
// closed form (uvSide).
func (c *ruledFaceUV) placeSeams(imprint []geom.Curve3) {
	c.seamU = 0
	var us []float64
	for _, cv := range imprint {
		for _, s := range c.sampleImprintUV(cv) {
			us = append(us, float64(s.a.X))
		}
	}
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			us = append(us, float64(c.paramOf(e.start()).X), float64(c.paramOf(e.end()).X))
		}
	}
	c.seamU = widestGapMid(us)
}

// assembleSegments emits the frame loops, the imprint and the seam as one tagged segment set, every
// shared incidence a common vertex (uvSide).
func (c *ruledFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	c.imprint = imprint
	seamHits := c.solveSeamCrossings(imprint)
	c.frameSegs = c.frameSegments(seamHits)
	segs := append([]uvSeg{}, c.frameSegs...)
	segs = append(segs, c.imprintSegments(imprint, seamHits)...)
	return append(segs, c.seamSegments(seamHits)...)
}

// emitRun re-emits a boundary run: frame and imprint runs as the exact sub-curve they lie on, a seam
// run as the ruling between its ends (uvSide).
func (c *ruledFaceUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	if run[0].kind == segSeam {
		return c.emitSeamRun(run)
	}
	return emitImprintRun(run)
}

// wrapsAllU reports whether the kept region wraps the azimuth — known once the kept boundary loops
// have been classified (uvSide).
func (c *ruledFaceUV) wrapsAllU() bool { return c.wrapping }

// multiFace: the kept region of a wall may be several patches (uvSide).
func (c *ruledFaceUV) multiFace() bool { return true }

// orientLoops keeps the arrangement's material-on-the-left winding; the stitch derives every face
// sense from that winding, so no per-rim source sense is consulted (uvSide, #3504).
func (c *ruledFaceUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
	}
	return faceLoops, nil, false
}

// finalizeLoops: a loop-framed wall has no synthetic apex rim to drop (uvSide).
func (c *ruledFaceUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return loops }

// frameContains reports whether a seam-relative (u,v) point lies inside the face's frame: an upward
// v-ray crosses the sampled boundary an odd number of times.
func (c *ruledFaceUV) frameContains(uv math.Point2) bool {
	return priorRayCrossings(c.frameSegs, float64(uv.X), float64(uv.Y))%2 == 1
}

// ruledFaceMaterial is the trim's material predicate: inside the frame AND kept by the boolean's keep
// table over the other operand's membership. A closure, so it reads the frame after assembleSegments.
func ruledFaceMaterial(c *ruledFaceUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			return c.frameContains(uv) && c.keptBySolid(float64(uv.X), float64(uv.Y))
		}
	}
}
