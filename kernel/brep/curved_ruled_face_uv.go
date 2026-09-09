// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

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
	loopFrame
	frame    geom.RuledFrame
	wrapping bool // some kept boundary loop wraps the azimuth (set by wrappingSolidFaces)
	// keepAt is the shared keep test (coincidentKeepAt): the ON/ON table where a face of the other
	// operand lies on this same surface, the membership oracle everywhere else. nil falls back to the
	// oracle alone, which is what every caller outside the mixed dispatch supplies.
	keepAt func(math.Point3) bool
}

// seamOverrun: a band is open in v, so the seam runs a full window past each end and bounds nothing
// there — EXCEPT past an apex, where the surface folds onto its other nappe and the overrun would land
// on real geometry. A face bounded by its apex gets no overrun at all, exactly as a sphere's chart gets
// none past a pole (loopFrameHost, ADR-0062).
func (c *ruledFaceUV) seamOverrun() float64 {
	if _, atApex := c.boundedByApex(); atApex {
		return 0
	}
	return c.band.vMax - c.band.vMin
}

// boundedByApex reports whether one end of the wall's window is the cone apex, and which. The apex is
// where the surface RADIUS vanishes, so that is what is measured — a radius is a length, so the weld
// tolerance is its class — rather than comparing the axial parameter to zero, which would be an exact
// float compare on a value the frame's own edges produced.
func (c *ruledFaceUV) boundedByApex() (float64, bool) {
	if c.frame.RadSlope == 0 {
		return 0, false // a cylinder has no apex
	}
	for _, v := range []float64{c.band.vMin, c.band.vMax} {
		if stdmath.Abs(c.frame.Radius(v)) <= c.res.Weld() {
			return v, true
		}
	}
	return 0, false
}

// apexSegments closes the parameter rectangle at the APEX. Like a sphere's pole segment it is a
// DEGENERATE edge — the whole azimuth at the apex is one point in space, so it bounds no geometry and
// welds to nothing. It exists so the frame's even-odd containment can read a patch that runs to the
// apex as inside, which the patch's own rim cannot say (ADR-0062).
func (c *ruledFaceUV) apexSegments() []uvSeg {
	v, atApex := c.boundedByApex()
	if !atApex {
		return nil
	}
	apex := c.point3(0, v)
	return []uvSeg{{
		a: math.P2(0, math.Scalar(v)), b: math.P2(2*stdmath.Pi, math.Scalar(v)),
		curve: geom.NewLineSegment(apex, apex), tA: 0, tB: 1, kind: segSeam,
	}}
}

// vWindow is the wall's axial window (loopFrameHost).
func (c *ruledFaceUV) vWindow() (float64, float64) { return c.band.vMin, c.band.vMax }

// seamCurve is the artificial boundary closing the periodic strip: for a ruled wall the RULING at the
// placed azimuth, bounded to the band AND its overrun so every crossing the frame can have lies on the
// seam itself. Bounding it to the ruling's own unit span instead would put real crossings off the end
// of it, and a crossing is only admitted where it lies on the seam (loopFrameHost).
func (c *ruledFaceUV) seamCurve() geom.Curve3 {
	pad := c.seamOverrun()
	return geom.NewLineSegment(c.point3(0, c.band.vMin-pad), c.point3(0, c.band.vMax+pad))
}

var _ uvSide = (*ruledFaceUV)(nil)

// newRuledFaceUV frames a wall for a solid-membership trim under op (isB marks it the boolean's B).
func newRuledFaceUV(f curvedFace, rs ruledSide, op Op, isB bool, inside func(math.Point3) bool) *ruledFaceUV {
	c := newRuledUVFrame(rs.frame.Base, rs.frame.Axis, rs.frame.Ref, rs.frame.RadSlope, rs.frame.RadConst, rs.band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	out := &ruledFaceUV{ruledUV: c, frame: rs.frame}
	out.loopFrame = loopFrame{host: out, face: f, res: geom.ResolutionForSize(rs.size())}
	return out
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
	// Running along a frame edge is the same contact said without reference to a plane, and it is the
	// half that matters when the imprint is not planar in form: a chamfer wedge's cone crosses the
	// shaft's wall exactly at the wedge's own lower rim, and the crossing comes back as a ruled arc,
	// which has no section plane for the test below to compare. The arrangement then carried the rim
	// TWICE, a rounding apart, and zig-zagged between them — a ninety-fragment boundary that welded to
	// nothing (ADR-0061 stage 4).
	if sectionOnFaceBoundary(imp, c.face, c.res) {
		return true
	}
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			if _, coincident := geom.SectionCrossingCandidates(c.face.surface, e.curve, imp, c.res); coincident {
				return true
			}
		}
	}
	return false
}

// placeSeams moves the azimuth seam clear of the imprint's exact azimuth extent AND of every frame
// vertex and ruling edge, so it crosses the frame only through the interior of a smooth section edge,
// and an imprint only where it must and transversally (uvSide, curved_seam_place.go).
func (c *ruledFaceUV) placeSeams(imprint []geom.Curve3) {
	c.seamU = 0
	c.seamU = c.exactSeamAzimuth(imprint, ringChartU)
}

// assembleSegments emits the frame loops, the imprint and the seam as one tagged segment set, every
// shared incidence a common vertex (uvSide).
func (c *ruledFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	c.imprint = imprint
	seamHits := c.solveSeamCrossings(imprint)
	c.frameSegs = c.frameSegments(seamHits)
	segs := append([]uvSeg{}, c.frameSegs...)
	segs = append(segs, c.imprintSegments(imprint, seamHits)...)
	segs = append(segs, c.seamSegments(seamHits)...)
	return append(segs, c.apexSegments()...)
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

// finalizeLoops has nothing of its own to do: the degenerate edges a cone's apex leaves are dropped for
// every chart at the one place the faces are built (trimByImprint, curved_uv_side.go) (uvSide).
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
			return c.frameContains(uv) && c.keptAt(float64(uv.X), float64(uv.Y))
		}
	}
}

// keptAt is one cell's keep verdict: the shared test when the mixed dispatch supplied one, else the
// chart's own membership reading.
func (c *ruledFaceUV) keptAt(u, v float64) bool {
	if c.keepAt != nil {
		return c.keepAt(c.point3(u, v))
	}
	return c.keptBySolid(u, v)
}

// vClosed: a ruled wall's axial window is bounded, not periodic (loopFrameHost).
func (c *ruledFaceUV) vClosed() bool { return false }

// tubeSeamCurve: v is a bounded window here, so there is no tube seam (loopFrameHost).
func (c *ruledFaceUV) tubeSeamCurve() (geom.Curve3, bool) { return nil, false }
