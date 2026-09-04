// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// sphereFaceUV is the (u,v) chart of a SPHERE face framed by its own loops — the same loop frame
// ruledFaceUV uses, over the sphere's longitude/latitude instead of a ruled wall's azimuth/axial
// distance (ADR-0060, ADR-0061 stage 3). It is what lets the mixed boolean split a sphere, which until
// now had no chart at all and passed whole or declined.
//
// A sphere brings one thing a ruled wall does not: POLES. Longitude is undefined at v = ±π/2, so a cap
// containing a pole has a boundary — its section circle — that does not enclose the pole in (u,v), and
// an even-odd frame containment would read the cap's own interior as outside. OCCT closes that with a
// DEGENERATE edge, a boundary in parameter space that is a single point in 3-D, and this does the same:
// the pole segments below run the full longitude at v = ±π/2, bounding no real geometry and welding to
// nothing, exactly as the longitude seam does.
type sphereFaceUV struct {
	loopFrame
	sphere      geom.Sphere
	seamU       float64
	op          Op
	isB         bool
	insideOther func(math.Point3) bool
	wrapping    bool // some kept boundary loop wraps the longitude (set by wrappingSolidFaces)
}

var (
	_ uvSide        = (*sphereFaceUV)(nil)
	_ loopFrameHost = (*sphereFaceUV)(nil)
)

// sphereFaceOf recognises a face the sphere chart can frame: a sphere surface whose every loop closes,
// so the frame's even-odd containment has a boundary to read. A BOUNDARY-LESS sphere qualifies too —
// the face is then the whole surface, which the frame reports directly rather than reading a boundary
// that is not there. ok=false for anything else.
func sphereFaceOf(f curvedFace) (geom.Sphere, bool) {
	s, ok := geom.SphereOf(f.surface)
	if !ok {
		return geom.Sphere{}, false
	}
	if f.outerless {
		return geom.Sphere{}, false // the complement of its rings: not a case this chart reads yet
	}
	res := geom.ResolutionForBox(faceLoopBox(f))
	for _, l := range f.loops {
		if !loopChainCloses(l, res) {
			return geom.Sphere{}, false
		}
	}
	return s, true
}

// newSphereFaceUV frames a sphere face for a solid-membership trim under op (isB marks it the
// boolean's B).
func newSphereFaceUV(f curvedFace, s geom.Sphere, op Op, isB bool, inside func(math.Point3) bool) *sphereFaceUV {
	c := &sphereFaceUV{sphere: s, op: op, isB: isB, insideOther: inside}
	c.loopFrame = loopFrame{host: c, face: f, res: geom.ResolutionForSize(2 * s.Radius)}
	return c
}

// paramOf inverts a point on the sphere to seam-relative (longitude, latitude) (uvSide).
func (c *sphereFaceUV) paramOf(p math.Point3) math.Point2 {
	d := c.sphere.Center.VectorTo(p)
	v := stdmath.Asin(stdmath.Max(-1, stdmath.Min(1, float64(d.Z)/c.sphere.Radius)))
	u := stdmath.Atan2(float64(d.Y), float64(d.X)) - c.seamU
	for u < 0 {
		u += 2 * stdmath.Pi
	}
	return math.P2(u, v)
}

// point3 inverts the parameterisation (loopFrameHost).
func (c *sphereFaceUV) point3(u, v float64) math.Point3 { return c.sphere.PointAt(u+c.seamU, v) }

// vWindow is the sphere's full latitude range: the poles bound it (loopFrameHost).
func (c *sphereFaceUV) vWindow() (float64, float64) { return -stdmath.Pi / 2, stdmath.Pi / 2 }

// seamOverrun is ZERO: latitude is closed by the poles, and a seam running past one maps onto the
// antipodal side of the sphere rather than beyond the geometry (loopFrameHost).
func (c *sphereFaceUV) seamOverrun() float64 { return 0 }

// seamCurve is the MERIDIAN at the placed longitude — the artificial boundary closing the periodic
// strip. It is a half great circle from pole to pole, not a straight ruling, which is why the framing
// takes a Curve3 (loopFrameHost).
func (c *sphereFaceUV) seamCurve() geom.Curve3 {
	normal := math.V3(-stdmath.Sin(c.seamU), stdmath.Cos(c.seamU), 0) // the meridian plane's normal
	ref := math.V3(stdmath.Cos(c.seamU), stdmath.Sin(c.seamU), 0)
	arc, err := geom.NewArc3d(c.sphere.Center, normal, ref, c.sphere.Radius, -stdmath.Pi/2, stdmath.Pi)
	if err != nil {
		return geom.NewLineSegment(c.point3(0, -stdmath.Pi/2), c.point3(0, stdmath.Pi/2))
	}
	return arc
}

// uPeriodic: longitude wraps (uvSide).
func (c *sphereFaceUV) uPeriodic() bool { return true }

// vPeriodic: latitude does not wrap — the poles bound it (uvSide).
func (c *sphereFaceUV) vPeriodic() bool { return false }

// multiFace: a sphere's kept region may be several caps (uvSide).
func (c *sphereFaceUV) multiFace() bool { return true }

// wrapsAllU reports whether the kept region wraps the longitude (uvSide).
func (c *sphereFaceUV) wrapsAllU() bool { return c.wrapping }

// placeSeams puts the longitude seam clear of every imprint and frame longitude, so the seam crosses
// the frame only through the interior of a smooth edge (uvSide).
func (c *sphereFaceUV) placeSeams(imprint []geom.Curve3) {
	c.seamU = 0
	var us []float64
	for _, cv := range imprint {
		lo, hi := cv.Domain()
		for k := 0; k <= sphereSeamProbe; k++ {
			t := lo + (hi-lo)*float64(k)/sphereSeamProbe
			us = append(us, float64(c.paramOf(cv.PointAt(t)).X))
		}
	}
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			us = append(us, float64(c.paramOf(e.start()).X), float64(c.paramOf(e.end()).X))
		}
	}
	c.seamU = widestGapMid(us)
}

// sphereSeamProbe samples an imprint's longitudes when choosing where to put the seam. It only has to
// find a gap, not a boundary, so a coarse walk is enough.
const sphereSeamProbe = 32

// assembleSegments samples the face's own loops, the imprint, the longitude seam and the POLE edges
// into the arrangement's tagged segment set (uvSide).
func (c *sphereFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	c.imprint = imprint
	seamHits := c.solveSeamCrossings(imprint)
	c.frameSegs = c.frameSegments(seamHits)
	segs := append([]uvSeg{}, c.frameSegs...)
	segs = append(segs, c.imprintSegments(imprint, seamHits)...)
	segs = append(segs, c.seamSegments(seamHits)...)
	return append(segs, c.poleSegments()...)
}

// poleSegments closes the parameter rectangle at v = ±π/2. Each is a DEGENERATE edge: the whole
// longitude at a pole is one point in 3-D, so it bounds no geometry and welds to nothing — it exists
// so the frame's even-odd containment can read a cap that contains a pole as inside, which its own
// section circle cannot say.
func (c *sphereFaceUV) poleSegments() []uvSeg {
	var out []uvSeg
	for _, v := range []float64{-stdmath.Pi / 2, stdmath.Pi / 2} {
		pole := c.point3(0, v)
		degenerate := geom.NewLineSegment(pole, pole)
		out = append(out, uvSeg{
			a: math.P2(0, v), b: math.P2(2*stdmath.Pi, v),
			curve: degenerate, tA: 0, tB: 1, kind: segSeam,
		})
	}
	return out
}

// emitRun re-emits a boundary run as the exact curve it lies on; a seam or pole run is artificial and
// re-emits as the meridian arc between its ends (uvSide).
func (c *sphereFaceUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	if run[0].kind == segSeam {
		return c.emitSeamRun(run)
	}
	return emitImprintRun(run)
}

// emitSeamRun re-emits an artificial run — a stretch of the longitude seam, or of a pole — as the
// meridian arc between its ends. A pole run collapses to a point and is emitted as the degenerate edge
// the arrangement carried, which the stitch drops.
func (c *sphereFaceUV) emitSeamRun(run []recoveredEdge) (loopEdge, bool) {
	a, b := run[0].a, run[len(run)-1].b
	pa, pb := c.point3(float64(a.X), float64(a.Y)), c.point3(float64(b.X), float64(b.Y))
	if float64(pa.DistanceTo(pb)) <= c.res.Weld() {
		return loopEdge{curve: geom.NewLineSegment(pa, pb), t0: 0, t1: 1, v0: pa, v1: pb}, true
	}
	arc, ok := meridianArcBetween(c.sphere, pa, pb)
	if !ok {
		return loopEdge{}, false
	}
	return arc, true
}

// meridianArcBetween is the great-circle arc from a to b on the sphere — the seam's own curve, since
// the seam is a meridian. ok=false when the two points are antipodal, where no unique arc exists.
func meridianArcBetween(s geom.Sphere, a, b math.Point3) (loopEdge, bool) {
	da, db := s.Center.VectorTo(a), s.Center.VectorTo(b)
	normal := da.Cross(db)
	if normal.LengthSquared() == 0 {
		return loopEdge{}, false
	}
	sweep := stdmath.Atan2(float64(normal.Length()), float64(da.Dot(db)))
	arc, err := geom.NewArc3d(s.Center, normal, da, s.Radius, 0, sweep)
	if err != nil {
		return loopEdge{}, false
	}
	return loopEdge{curve: arc, t0: 0, t1: 1, v0: a, v1: b}, true
}

// orientLoops keeps the arrangement's material-on-the-left winding and hands back the section arcs
// reversed, as every loop-framed chart does (uvSide).
func (c *sphereFaceUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
		lid = append(lid, reverseEdgeChain(e.section)...)
	}
	return faceLoops, lid, false
}

// finalizeLoops drops the DEGENERATE pole edges. A pole is one point in 3-D, so the parameter-space
// boundary that closes the strip there bounds nothing and must not survive as an edge: left in, it is a
// zero-length edge with a single use, and the body reads as open. This is the counterpart of OCCT's
// degenerate edges, which exist in the face's wire and are skipped by everything that builds geometry
// from it (uvSide).
func (c *sphereFaceUV) finalizeLoops(loops []curvedLoop) []curvedLoop {
	return dropDegenerateEdges(loops, c.res)
}

// dropDegenerateEdges removes the zero-length straight edges a chart's POLE or APEX segment leaves
// behind: a boundary in parameter space that is one point in space. Shared by every chart whose
// parameter rectangle is closed by a singular point of its surface (ADR-0062).
func dropDegenerateEdges(loops []curvedLoop, res geom.Resolution) []curvedLoop {
	out := make([]curvedLoop, 0, len(loops))
	for _, l := range loops {
		edges := make([]loopEdge, 0, len(l.edges))
		for _, e := range l.edges {
			if float64(e.start().DistanceTo(e.end())) <= res.Weld() && geom.IsStraightCurve(e.curve) {
				continue
			}
			edges = append(edges, e)
		}
		if len(edges) > 0 {
			out = append(out, curvedLoop{edges: edges})
		}
	}
	return out
}

// wrappingSolidFaces emits a kept region that WRAPS the longitude. A cap is exactly that: bounded above
// by its section circle, which turns the whole way round, and below by a pole, which is a boundary in
// parameter space and a single point in space (uvSide).
func (c *sphereFaceUV) wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool) {
	faces, lid, ok := c.wrappingComponents(c, kept, segs, surface, f)
	c.wrapping = ok
	return faces, lid, ok
}

// frameContains reports whether a seam-relative (u,v) point lies inside the face's frame: an upward
// v-ray crosses the sampled boundary an odd number of times. A BOUNDARY-LESS sphere is its whole
// surface, so every point is inside it — there is no boundary to count crossings of.
func (c *sphereFaceUV) frameContains(uv math.Point2) bool {
	vMin, vMax := c.vWindow()
	if v := float64(uv.Y); v < vMin || v > vMax {
		return false // beyond a pole: the seam's overrun, which bounds nothing real
	}
	if len(c.face.loops) == 0 {
		return true
	}
	return priorRayCrossings(c.frameSegs, float64(uv.X), float64(uv.Y))%2 == 1
}

// sphereFaceMaterial is the trim's material predicate: inside the frame AND kept by the boolean's keep
// table over the other operand's membership.
func sphereFaceMaterial(c *sphereFaceUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			if !c.frameContains(uv) {
				return false
			}
			return keep(c.op, c.isB, c.insideOther(c.point3(float64(uv.X), float64(uv.Y))))
		}
	}
}

// vClosed: latitude is a bounded window between the poles, not a period (loopFrameHost).
func (c *sphereFaceUV) vClosed() bool { return false }

// seamOrigin is the surface parameter of the chart's (0,0): a sphere chart rotates its longitude
// origin to place the seam; its latitude is the surface's own (uvSide, ADR-0063).
func (c *sphereFaceUV) seamOrigin() math.Point2 { return math.P2(c.seamU, 0) }
