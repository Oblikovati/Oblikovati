// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The half-space split of a ruled wall whose boundary the plane crosses in a shape the boundary WALK
// cannot pair (ADR-0060, ADR-0061 stage 2).
//
// loopedSplit walks one loop, pairs its crossings and keeps the runs between them. It needs exactly one
// loop crossed an even number of times; a wall carrying a notch or a hole, or one the plane meets at a
// tangency, has neither, and declined to the CSG fallback. Those are not a different KIND of cut — they
// are the same cut on a face whose boundary is richer than a walk can pair, which is exactly what the
// loop-framed chart exists to trim: it frames the wall by its own loops, however many there are.
//
// This is a classification, not a retry: boundaryWalkPairs decides which of the two takes the face
// before either runs.

// ruledHalfSpaceSplit trims a ruled wall by the cutting plane through the loop-framed chart, returning
// the kept (negative-side) faces and the section arcs that bound the lid. ok=false when the face is not
// a ruled wall the chart can frame, or the chart cannot trim it — the caller then takes its own path,
// which is why there is no error to report: a decline is not a failure of the operation.
func ruledHalfSpaceSplit(f curvedFace, curves []geom.Curve3, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, bool) {
	rs, ok := ruledFaceOf(f)
	if !ok {
		return nil, nil, false
	}
	bounded, ok := boundedSections(curves, rs)
	if !ok {
		return nil, nil, false
	}
	// Difference with isB=false keeps where the predicate is FALSE, so the predicate is "on the positive
	// side": the kept region is the negative half-space, which is what a half-space cut means.
	c := newRuledFaceUV(f, rs, Difference, false, func(p math.Point3) bool {
		return signedDistance(p, plane, n) > 0
	})
	imprint := c.admits(bounded)
	if len(imprint) == 0 {
		return wholeSideOfPlane(f, plane, n), nil, true // every section lay on the frame: untouched
	}
	faces, lid, err := trimByImprint(c, f, rs.surface, imprint, ruledFaceMaterial(c))
	if err != nil {
		return nil, nil, false
	}
	return faces, lid, true
}

// boundaryWalkPairs reports whether the plane crosses the face's boundary in the shape splitLoopByPlane
// pairs: exactly ONE loop, crossed an EVEN number of times. Those are loopedSplit's own preconditions,
// read here so the two paths are chosen by a classification rather than by one failing into the other.
func boundaryWalkPairs(f curvedFace, plane geom.Plane, n math.Vector3, res geom.Resolution) bool {
	if len(f.loops) != 1 {
		return false
	}
	_, crossings := splitLoopByPlane(f.loops[0], plane, n, res)
	return crossings%2 == 0
}

// wholeSideOfPlane keeps or drops a face the plane does not actually divide, by the side its sample
// point falls on.
func wholeSideOfPlane(f curvedFace, plane geom.Plane, n math.Vector3) []curvedFace {
	if signedDistance(faceSample(f), plane, n) <= 0 {
		return []curvedFace{f}
	}
	return nil
}

// boundedSections clips an UNBOUNDED plane section to the wall's own axial window. A plane cutting a
// ruled wall parallel to its axis sections it in a curve that runs to infinity — a ruling on a
// cylinder, a hyperbola arm on a cone — and a chart samples an imprint over its own domain, so an
// infinite domain samples to nothing and the cut reads as no imprint at all (ADR-0062).
//
// ok=false when a section is unbounded and its axial coordinate cannot be bracketed: the chart would
// have no honest interval to sample, so it declines rather than inventing one.
func boundedSections(curves []geom.Curve3, rs ruledSide) ([]geom.Curve3, bool) {
	out := make([]geom.Curve3, 0, len(curves))
	for _, c := range curves {
		lo, hi := c.Domain()
		if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
			out = append(out, c)
			continue
		}
		spans, ok := geom.AxialWindowParams(c, rs.frame.Base, rs.frame.Axis, rs.band.vMin, rs.band.vMax)
		if !ok {
			return nil, false
		}
		for _, sp := range spans {
			out = append(out, geom.TrimmedCurve3{Base: c, Lo: sp[0], Hi: sp[1]})
		}
	}
	return out, true
}
