// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// planeUV is a bounded, NON-periodic planar seat face expressed as a uvSide, so a partial curved-on-planar
// boolean (a drilled hole / boss base whose imprint conic CLIPS the face boundary) runs through the one
// shared (u,v)-arrangement trimmer instead of triangle-soup CSG (#1591, ADR-0049). The chart is the plane's
// own (u,v) = to2D(plane, ·) — an isometry, no azimuth seam, no rim circles. The frame is the face POLYGON
// (segPolygon edges) rather than rims+seam, and the periodic machinery is inert: placeSeams a no-op,
// u/vPeriodic and wrapsAllU false. The material predicate keeps face-interior cells that are outside the tool.
type planeUV struct {
	plane  geom.Plane
	seatUV [][]math.Point2        // seat face boundary in (u,v), outer loop first (even-odd containment)
	seat3D [][]math.Point3        // the same loops in 3D, so each polygon edge re-emits as its exact segment
	inTool func(math.Point3) bool // reports whether a 3D point is inside the tool solid (removed for a drill)
}

// planeUV satisfies uvSide: a non-periodic, polygon-framed plane (#1591).
var _ uvSide = (*planeUV)(nil)

// paramOf inverts a seat-plane point to its (u,v) chart coordinates (uvSide) — a plain orthonormal projection,
// no seam offset (the plane is non-periodic).
func (c *planeUV) paramOf(p math.Point3) math.Point2 { return to2D(c.plane, p) }

// placeSeams is a no-op: a bounded plane has no artificial seam to move clear of the imprint (uvSide).
func (c *planeUV) placeSeams(_ []geom.Curve3) {}

// vPeriodic reports that a plane's v does not wrap (uvSide).
func (c *planeUV) vPeriodic() bool { return false }

// uPeriodic reports that a plane's u does not wrap — u is a real world distance, so the boundary welder must
// NOT fold u≈2π onto u=0 (uvSide, ADR-0049 D-c).
func (c *planeUV) uPeriodic() bool { return false }

// wrapsAllU reports that a plane never wraps the azimuth — there is none (uvSide).
func (c *planeUV) wrapsAllU() bool { return false }

// multiFace reports that a plane cut may leave the seat DISCONNECTED (a slot severing the face into two
// pieces), so the boundary loops are grouped into separate faces by containment (uvSide).
func (c *planeUV) multiFace() bool { return true }

// wrappingSolidFaces never applies to a plane (there is no wrapping tube band); it defers to the standard
// contractible-outer emission (uvSide).
func (c *planeUV) wrappingSolidFaces(_ []Face2D, _ []uvSeg, _ geom.Surface, _ curvedFace) ([]curvedFace, bool) {
	return nil, false
}

// finalizeLoops is identity: a plane has no apex-pole rim or torus hole to re-mark (uvSide).
func (c *planeUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return loops }

// emitRun re-emits one boundary run as an exact loopEdge (uvSide). Both the imprint conic (segImprint) and a
// polygon boundary edge (segPolygon) are analytic curves re-emitted over their recovered parameter span, so
// emitImprintRun serves both — the shared re-emitter that makes the seat arc and the tool wall weld exactly.
func (c *planeUV) emitRun(run []recoveredEdge) (loopEdge, bool) { return emitImprintRun(run) }

// orientLoops applies the plane's winding convention (uvSide): every kept loop's edges run forward, and the
// imprint (conic) sub-arcs are reversed into the lid — the shared cut boundary the tool wall reuses so the
// two faces weld. A plane is open in both u and v, so the kept face always has an outer loop (outerless=false).
func (c *planeUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
		lid = append(lid, reverseEdgeChain(e.section)...)
	}
	return faceLoops, lid, false
}

// assembleSegments samples the imprint conic(s) into the tagged (u,v) segment set and appends the face
// polygon as the non-periodic frame (uvSide). The arrangement subdivides both, splitting each at their exact
// crossings; keptCells then classifies cells by the material predicate.
func (c *planeUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	out := make([]uvSeg, 0, len(imprint)*imprintSampleCount+len(c.seat3D)*4)
	for _, cv := range imprint {
		out = append(out, c.sampleConicUV(cv)...)
	}
	return append(out, c.frameSegments()...)
}

// sampleConicUV samples one imprint conic into (u,v) segments carrying the source curve + its parameters, so
// a boundary run along it re-emits as the exact analytic arc (not the sampled chord).
func (c *planeUV) sampleConicUV(cv geom.Curve3) []uvSeg {
	lo, hi := cv.Domain()
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevP, prevT := to2D(c.plane, cv.PointAt(lo)), lo
	for i := 1; i <= imprintSampleCount; i++ {
		t := lo + (hi-lo)*float64(i)/imprintSampleCount
		p := to2D(c.plane, cv.PointAt(t))
		segs = append(segs, uvSeg{a: prevP, b: p, curve: cv, tA: prevT, tB: t, kind: segImprint})
		prevP, prevT = p, t
	}
	return segs
}

// frameSegments emits the seat face boundary as segPolygon edges — the plane's non-periodic frame. Each edge
// carries its own straight segment as the source curve so a kept boundary run re-emits it exactly.
func (c *planeUV) frameSegments() []uvSeg {
	var out []uvSeg
	for _, ring := range c.seat3D {
		for i, n := 0, len(ring); i < n; i++ {
			a3, b3 := ring[i], ring[(i+1)%n]
			seg := geom.NewLineSegment(a3, b3)
			out = append(out, uvSeg{a: to2D(c.plane, a3), b: to2D(c.plane, b3), curve: seg, tA: 0, tB: 1, kind: segPolygon})
		}
	}
	return out
}

// planeMaterial builds the planeUV material predicate: keep a cell whose deep-interior (u,v) point lies inside
// the seat face polygon AND outside the tool solid (a drill removes the tool; the seat retains the rest). A
// closure (not a bound value) so it reads the receiver live, matching the ruled/torus material builders.
func planeMaterial(c *planeUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			return pointInUVLoops(uv, c.seatUV) && !c.inTool(to3D(c.plane, uv))
		}
	}
}

// pointInUVLoops reports whether q is inside a face given as (u,v) loops (outer first, then holes): inside the
// outer loop and outside every hole — the even-odd containment the planar boolean uses (pointInPolygon2D).
func pointInUVLoops(q math.Point2, loops [][]math.Point2) bool {
	if len(loops) == 0 || !pointInPolygon2D(q, loops[0]) {
		return false
	}
	for _, hole := range loops[1:] {
		if pointInPolygon2D(q, hole) {
			return false
		}
	}
	return true
}
