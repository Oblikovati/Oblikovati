// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

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

// multiFace is FALSE for a torus: its kept boundary loops belong to ONE face, not to several grouped by
// containment. On a closed surface two disjoint loops can bound a single region — a plane through the
// hole cuts both walls, and the part that survives is the ring between the two ovals, neither of which
// contains the other in (u,v). Grouping by containment split that into two faces, each carrying one oval
// (uvSide).
func (c *torusFaceUV) multiFace() bool { return false }

// wrapsAllU: the rim-orientation flip is a band's, not a closed surface's (uvSide).
func (c *torusFaceUV) wrapsAllU() bool { return false }

// wrappingSolidFaces emits a kept region that WRAPS the azimuth: a perpendicular cut leaves a band
// bounded by TWO section circles, each turning the whole way round, which the contractible emission
// cannot file (uvSide).
func (c *torusFaceUV) wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool) {
	return c.wrappingComponents(c, kept, segs, surface, f)
}

// placeSeams puts BOTH seams clear of the imprint's exact extent in that coordinate and of the frame's
// own, so a contractible patch lands clear of both artificial seams (uvSide, curved_seam_place.go).
func (c *torusFaceUV) placeSeams(imprint []geom.Curve3) {
	c.seamU, c.seamV = 0, 0
	c.seamU, c.seamV = c.exactSeamAzimuth(imprint, ringChartU), c.exactSeamAzimuth(imprint, ringChartV)
}

// assembleSegments samples the face's own loops, the imprint and the four artificial seams closing the
// doubly-periodic rectangle (uvSide).
func (c *torusFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	c.imprint = imprint
	seamHits := c.solveSeamCrossings(imprint)
	c.frameSegs = c.frameSegments(seamHits)
	segs := append([]uvSeg{}, c.frameSegs...)
	segs = append(segs, c.imprintSegments(imprint, seamHits)...)
	return append(segs, c.rectangleSeams(seamHits)...)
}

// rectangleSeams closes the doubly-periodic parameter rectangle on all four sides, each side split at
// every incidence solved on it so the seam shares those vertices with the curves that cross it. The
// torus is closed across each, so none bounds real geometry: a kept region wrapping one folds to its
// reverse twin, and an all-seam loop is dropped.
func (c *torusFaceUV) rectangleSeams(seamHits []frameCrossing) []uvSeg {
	uSeam, vSeam := c.seamCurve(), c.azimuthCircle()
	us, vs := []float64{0, twoPi}, []float64{0, twoPi}
	for _, cr := range seamHits {
		at := c.paramOf(c.seamHitPoint(cr))
		if cr.tube {
			us = append(us, float64(at.X))
		} else {
			vs = append(vs, float64(at.Y))
		}
	}
	var out []uvSeg
	for _, side := range []struct {
		along []float64
		onV   float64
		vSide bool
		curve geom.Curve3
	}{{us, 0, true, vSeam}, {us, twoPi, true, vSeam}, {vs, 0, false, uSeam}, {vs, twoPi, false, uSeam}} {
		sort.Float64s(side.along)
		for i := 1; i < len(side.along); i++ {
			if side.along[i]-side.along[i-1] <= arrTol {
				continue
			}
			a, b := math.P2(side.along[i-1], side.onV), math.P2(side.along[i], side.onV)
			if !side.vSide {
				a, b = math.P2(side.onV, side.along[i-1]), math.P2(side.onV, side.along[i])
			}
			out = append(out, uvSeg{a: a, b: b, curve: side.curve, tA: 0, tB: 1, kind: segSeam})
		}
	}
	return out
}

// seamHitPoint is the 3D point of a seam incidence, on whichever curve carries it.
func (c *torusFaceUV) seamHitPoint(cr frameCrossing) math.Point3 {
	if cr.loop == seamIncidence {
		return c.imprint[cr.edge].PointAt(cr.tEdge)
	}
	return c.face.loops[cr.loop].edges[cr.edge].curve.PointAt(cr.tEdge)
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
	outerless := closedSurfaceOuterless(loops)
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

// vClosed: the tube angle is a period, so an imprint sampled across it wraps and must split at the
// v-seam (loopFrameHost).
func (c *torusFaceUV) vClosed() bool { return true }

// tubeSeamCurve is the parallel at the placed tube angle — the seam closing the tube period
// (loopFrameHost).
func (c *torusFaceUV) tubeSeamCurve() (geom.Curve3, bool) { return c.azimuthCircle(), true }

// seamOrigin is the surface parameter of the chart's (0,0): a torus chart places both seams
// (uvSide, ADR-0063).
func (c *torusFaceUV) seamOrigin() math.Point2 { return math.P2(c.seamU, c.seamV) }
