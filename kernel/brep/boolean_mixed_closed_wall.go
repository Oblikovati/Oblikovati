// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// The CLOSED-SURFACE × RULED-WALL crossing (ADR-0061 stage 4, first slice). A ball and a coaxial rod
// meet along a circle that lies on a sphere AND on a cylinder — the simplest curved-versus-curved
// contact there is, and until now the mixed boolean declined it on box overlap alone and handed the
// whole family to the bespoke ball-and-rod recognizers.
//
// The rule is the one the plane×wall pairing already follows: solve the crossing ONCE, in closed form,
// and write the same curves into BOTH sides' imprint lists, so the two charts split on identical
// coordinates and their fragments weld. What is new is only which two buckets are paired.

// pairClosedSurfaceWallImprints imprints every (closed-surface face of p, ruled wall of other) pair
// whose boxes overlap, appending the shared crossing to both lists. ok=false declines the boolean.
func pairClosedSurfaceWallImprints(p, other *facePartition, sphImp, wallImp [][]geom.Curve3) bool {
	faces, boxes := p.closedSurfaces()
	for i, sf := range faces {
		box := inflateBox(boxes[i])
		for k, wf := range other.wall {
			if !box.Intersects(inflateBox(other.wallBox[k])) {
				continue
			}
			curves, ok := closedSurfaceWallImprint(sf, wf)
			if !ok {
				return false
			}
			sphImp[i] = append(sphImp[i], curves...)
			wallImp[k] = append(wallImp[k], curves...)
		}
	}
	return true
}

// closedSurfaceWallImprint is the exact shared imprint of one (closed surface, ruled wall) pair.
//
// SCOPE, and it is deliberately narrow — this is the first slice of stage 4, not the whole of it:
//
//   - the closed surface must be BOUNDARY-LESS (a bare ball or torus), so every crossing is inside its
//     trim by construction and no crossing with one of its own edges has to be solved;
//   - every crossing curve must come back CLOSED, so it is an island on both charts and each side
//     splits by even-odd containment alone;
//   - every crossing must lie strictly INSIDE the wall's band or strictly clear of it. A crossing with
//     the INFINITE ruled surface is not a crossing with the wall: a rod that starts at a ball's centre
//     crosses the sphere in two circles, and only one of them is on the rod. Imprinting the other cut
//     the ball where nothing touches it and the difference came back with the ball's face missing.
//
// Anything else declines, and pairing it is the rest of stage 4.
func closedSurfaceWallImprint(sf, wf curvedFace) ([]geom.Curve3, bool) {
	rs, ok := ruledFaceOf(wf)
	if !ok || len(sf.loops) > 0 {
		return nil, false
	}
	res := geom.ResolutionForSize(rs.size())
	curves, handled := geom.IntersectSurfacesAnalytic(sf.surface, rs.surface, res)
	if !handled {
		return nil, false
	}
	return keepCrossingsOnTheWall(curves, rs, res)
}

// keepCrossingsOnTheWall keeps the crossings that lie on the wall itself and drops the ones the
// infinite surface contributes; ok=false when one straddles a rim or does not close.
func keepCrossingsOnTheWall(curves []geom.Curve3, rs ruledSide, res geom.Resolution) ([]geom.Curve3, bool) {
	var out []geom.Curve3
	for _, cv := range curves {
		if !closedCrossing(cv, res) {
			return nil, false
		}
		switch inside, clear := crossingBandPlacement(cv, rs); {
		case inside:
			out = append(out, cv)
		case clear:
		default:
			return nil, false // straddles a rim: a crossing this slice does not pair
		}
	}
	return out, true
}

// closedCrossing reports a crossing curve that returns to where it started.
func closedCrossing(cv geom.Curve3, res geom.Resolution) bool {
	lo, hi := cv.Domain()
	return float64(cv.PointAt(lo).DistanceTo(cv.PointAt(hi))) <= res.Sew()
}

// crossingBandPlacement classifies a crossing's axial span against the wall's band (bandPlacement).
func crossingBandPlacement(cv geom.Curve3, rs ruledSide) (inside, clear bool) {
	lo, hi := crossingAxialSpan(cv, rs)
	return bandPlacement(lo, hi, rs.band)
}

// crossingAxialSpan is the crossing's extent along the wall's axis, walked on the curve itself. The
// curve is analytic and the question is metric — where does it sit between the rims — so sampling it
// bounds the span without deciding any topology.
func crossingAxialSpan(cv geom.Curve3, rs ruledSide) (lo, hi float64) {
	t0, t1 := cv.Domain()
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for i := 0; i <= crossingSpanSamples; i++ {
		v := bandV(cv.PointAt(t0+(t1-t0)*float64(i)/crossingSpanSamples), rs.axis, rs.band)
		lo, hi = stdmath.Min(lo, v), stdmath.Max(hi, v)
	}
	return lo, hi
}

// crossingSpanSamples walks a crossing to bound its axial extent. It places the curve between two rims,
// nothing finer.
const crossingSpanSamples = 64

// pairWallWallImprints imprints every (wall of p, wall of other) pair whose boxes overlap, appending the
// shared crossing to both lists — the ruled-versus-ruled counterpart of the closed-surface pairing above
// (ADR-0061 stage 4). ok=false declines the boolean.
func pairWallWallImprints(p, other *facePartition, impP, impOther [][]geom.Curve3) bool {
	for i, wf := range p.wall {
		box := inflateBox(p.wallBox[i])
		for k, of := range other.wall {
			// A box overlap alone is not contact, and a pair the separation proof settles must not be
			// asked for a crossing it does not have: an emboss pad riding a constant sagitta clear of a
			// chamfer cone overlaps its box completely (#3459).
			if !box.Intersects(inflateBox(other.wallBox[k])) ||
				geom.SurfacesApart(wf.surface, of.surface, facePairCullPad) {
				continue
			}
			curves, ok := wallWallImprint(wf, of)
			if !ok {
				return false
			}
			impP[i] = append(impP[i], curves...)
			impOther[k] = append(impOther[k], curves...)
		}
	}
	return true
}

// wallWallImprint is the exact shared imprint of one (wall, wall) pair, under the same narrow scope the
// closed-surface pairing takes: every crossing must come back CLOSED, and must lie strictly inside BOTH
// bands or strictly clear of them. Anything else declines.
func wallWallImprint(a, b curvedFace) ([]geom.Curve3, bool) {
	ra, okA := ruledFaceOf(a)
	rb, okB := ruledFaceOf(b)
	if !okA || !okB {
		return nil, false
	}
	res := geom.ResolutionForSize(stdmath.Max(ra.size(), rb.size()))
	curves, handled := geom.IntersectSurfacesAnalytic(ra.surface, rb.surface, res)
	if !handled {
		return nil, false
	}
	kept, ok := keepCrossingsOnTheWall(curves, ra, res)
	if !ok {
		return nil, false
	}
	return keepCrossingsOnTheWall(kept, rb, res)
}
