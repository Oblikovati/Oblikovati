// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// torusFaceUV is the (u,v) chart of a TORUS face framed by its own loops — the same loop frame a ruled
// wall and a sphere use, over azimuth and tube angle (ADR-0060, ADR-0061 stage 3). Until now a torus
// had no chart in the mixed boolean: it went to the pass-through bucket, so any tool that met it
// declined, and the only way to cut one was the half-space pipeline ADR-0062 deletes.
//
// A torus is DOUBLY periodic and has no poles, which makes it the simplest of the three charts in one
// way and the hardest in another: both parameter directions wrap, so the frame closes the rectangle on
// all four sides with artificial seams. None of them bounds real geometry — the surface is closed
// across each — so a boundary run on one is either folded to its reverse twin where the kept region
// wraps it, or dropped as an all-seam loop.
type torusFaceUV struct {
	loopFrame
	torus        geom.Torus
	seamU, seamV float64
	op           Op
	isB          bool
	insideOther  func(math.Point3) bool
}

var (
	_ uvSide        = (*torusFaceUV)(nil)
	_ loopFrameHost = (*torusFaceUV)(nil)
)

// torusFaceOf recognises a face the torus chart can frame: a torus surface whose every loop closes, so
// the frame's even-odd containment has a boundary to read. A BOUNDARY-LESS torus qualifies too — the
// face is then the whole surface. ok=false for anything else.
func torusFaceOf(f curvedFace) (geom.Torus, bool) {
	t, ok := geom.TorusOf(f.surface)
	if !ok || f.outerless {
		return geom.Torus{}, false
	}
	res := geom.ResolutionForBox(faceLoopBox(f))
	for _, l := range f.loops {
		if !loopChainCloses(l, res) {
			return geom.Torus{}, false
		}
	}
	return t, true
}

// newTorusFaceUV frames a torus face for a solid-membership trim under op (isB marks it the boolean's B).
func newTorusFaceUV(f curvedFace, t geom.Torus, op Op, isB bool, inside func(math.Point3) bool) *torusFaceUV {
	c := &torusFaceUV{torus: t, op: op, isB: isB, insideOther: inside}
	c.loopFrame = loopFrame{host: c, face: f, res: geom.ResolutionForSize(2 * (t.MajorRadius + t.MinorRadius))}
	return c
}

// paramOf inverts a point on the torus to seam-relative (azimuth, tube angle) (uvSide).
func (c *torusFaceUV) paramOf(p math.Point3) math.Point2 {
	u, v := c.torus.ParamAt(p)
	return math.P2(wrapAngle(u-c.seamU), wrapAngle(v-c.seamV))
}

// point3 inverts the parameterisation (loopFrameHost).
func (c *torusFaceUV) point3(u, v float64) math.Point3 {
	return c.torus.PointAt(u+c.seamU, v+c.seamV)
}

// vWindow is the tube's full turn: both directions wrap, so the frame closes the whole rectangle
// (loopFrameHost).
func (c *torusFaceUV) vWindow() (float64, float64) { return 0, 2 * stdmath.Pi }

// seamOverrun is ZERO: both directions are closed, so a seam past the period maps back onto the torus
// (loopFrameHost).
func (c *torusFaceUV) seamOverrun() float64 { return 0 }

// seamCurve is the TUBE CIRCLE at the placed azimuth — the artificial boundary closing the strip in u.
// It is a real circle on the torus, which is what makes its crossings with the frame solvable in closed
// form like any other section (loopFrameHost).
func (c *torusFaceUV) seamCurve() geom.Curve3 {
	// the tube centre at this azimuth: the major-radius point on the torus's own plane
	radial := unit(c.torus.Center.VectorTo(c.torus.PointAt(c.seamU, stdmath.Pi/2)).Sub(
		c.torus.AxisDir.AsVector().Scale(math.Scalar(c.torus.MinorRadius))))
	centre := c.torus.Center.TranslateBy(radial.Scale(math.Scalar(c.torus.MajorRadius)))
	normal := c.torus.AxisDir.AsVector().Cross(radial)
	circle, err := geom.NewCircle(centre, normal, c.torus.MinorRadius)
	if err != nil {
		return geom.NewLineSegment(c.point3(0, 0), c.point3(0, stdmath.Pi))
	}
	return circle
}

// uPeriodic: the azimuth wraps (uvSide).
func (c *torusFaceUV) uPeriodic() bool { return true }

// vPeriodic: the tube angle wraps too (uvSide).
func (c *torusFaceUV) vPeriodic() bool { return true }

// multiFace: a torus's kept region may be several patches (uvSide).
func (c *torusFaceUV) multiFace() bool { return true }

// wrapsAllU: the rim-orientation flip is a band's, not a closed surface's (uvSide).
func (c *torusFaceUV) wrapsAllU() bool { return false }

// wrappingSolidFaces: a torus is emitted by the ordinary contractible path (uvSide).
func (c *torusFaceUV) wrappingSolidFaces(_ []Face2D, _ []uvSeg, _ geom.Surface, _ curvedFace) ([]curvedFace, []loopEdge, bool) {
	return nil, nil, false
}

// placeSeams puts BOTH seams in the widest gap of the imprint's and the frame's own crossings, so a
// contractible patch lands clear of both artificial seams (uvSide).
func (c *torusFaceUV) placeSeams(imprint []geom.Curve3) {
	c.seamU, c.seamV = 0, 0
	var us, vs []float64
	sample := func(p math.Point3) {
		u, v := c.torus.ParamAt(p)
		us, vs = append(us, u), append(vs, v)
	}
	for _, cv := range imprint {
		lo, hi := cv.Domain()
		for i := 0; i <= imprintSampleCount; i++ {
			sample(cv.PointAt(lo + (hi-lo)*float64(i)/imprintSampleCount))
		}
	}
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			sample(e.start())
			sample(e.end())
		}
	}
	c.seamU, c.seamV = widestGapMid(us), widestGapMid(vs)
}

// assembleSegments samples the face's own loops, the imprint and the four artificial seams closing the
// doubly-periodic rectangle (uvSide).
func (c *torusFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	c.imprint = imprint
	seamHits := c.solveSeamCrossings(imprint)
	c.frameSegs = c.frameSegments(seamHits)
	segs := append([]uvSeg{}, c.frameSegs...)
	segs = append(segs, c.imprintSegments(imprint, seamHits)...)
	return append(segs, c.rectangleSeams()...)
}

// rectangleSeams closes the doubly-periodic parameter rectangle on all four sides. The torus is closed
// across each, so none bounds real geometry: a kept region wrapping one folds to its reverse twin, and
// an all-seam loop is dropped.
func (c *torusFaceUV) rectangleSeams() []uvSeg {
	twoPi := 2 * stdmath.Pi
	uSeam, vSeam := c.seamCurve(), c.azimuthCircle()
	return []uvSeg{
		{a: math.P2(0, 0), b: math.P2(twoPi, 0), curve: vSeam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(0, twoPi), b: math.P2(twoPi, twoPi), curve: vSeam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(0, 0), b: math.P2(0, twoPi), curve: uSeam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(twoPi, 0), b: math.P2(twoPi, twoPi), curve: uSeam, tA: 0, tB: 1, kind: segSeam},
	}
}

// azimuthCircle is the circle at the placed tube angle — the artificial boundary closing the strip in
// v, and a real circle on the torus.
func (c *torusFaceUV) azimuthCircle() geom.Curve3 {
	r := c.torus.MajorRadius + c.torus.MinorRadius*stdmath.Cos(c.seamV)
	centre := c.torus.Center.TranslateBy(c.torus.AxisDir.AsVector().Scale(math.Scalar(c.torus.MinorRadius * stdmath.Sin(c.seamV))))
	circle, err := geom.NewCircle(centre, c.torus.AxisDir.AsVector(), r)
	if err != nil {
		return geom.NewLineSegment(c.point3(0, 0), c.point3(stdmath.Pi, 0))
	}
	return circle
}

// emitRun re-emits a boundary run as the exact curve it lies on. A surviving SEAM run is a topology the
// trim does not model — the seams bound nothing real, so a kept boundary that follows one means the
// region wrapped a period in a way the fold did not resolve — and declines rather than inventing an
// edge (uvSide).
func (c *torusFaceUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	if run[0].kind == segSeam {
		return loopEdge{}, false
	}
	return emitImprintRun(run)
}

// orientLoops keeps the arrangement's material-on-the-left winding and reports whether the kept face is
// OUTERLESS: on a closed surface a single CW loop bounds a dropped island, so the face is the
// complement of its rings and has no outer loop (uvSide).
func (c *torusFaceUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	outerless := len(loops) == 1 && loops[0].area < 0
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
		lid = append(lid, reverseEdgeChain(e.section)...)
	}
	return faceLoops, lid, outerless
}

// finalizeLoops: a torus has no degenerate pole to drop (uvSide).
func (c *torusFaceUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return loops }

// frameContains reports whether a seam-relative (u,v) point lies inside the face's frame. A
// BOUNDARY-LESS torus is its whole surface, so every point of the rectangle is inside it.
func (c *torusFaceUV) frameContains(uv math.Point2) bool {
	if len(c.face.loops) == 0 {
		return true
	}
	return priorRayCrossings(c.frameSegs, float64(uv.X), float64(uv.Y))%2 == 1
}

// torusFaceMaterial is the trim's material predicate: inside the frame AND kept by the boolean's keep
// table over the other operand's membership.
func torusFaceMaterial(c *torusFaceUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			if !c.frameContains(uv) {
				return false
			}
			return keep(c.op, c.isB, c.insideOther(c.point3(float64(uv.X), float64(uv.Y))))
		}
	}
}
