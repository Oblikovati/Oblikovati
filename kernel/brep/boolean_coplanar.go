// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Coplanar (ON/ON) face fragments — where a face of A lies in the same plane as a face of B
// and they overlap as an area, not a line — can't be classified by the plane-plane imprint
// line nor by an inside/outside ray cast (the sample point sits exactly on the other solid's
// boundary). They are handled by set-membership classification instead: split each face by
// the other's coplanar outline, then keep fragments per [coplanarKeep]. See Mantyla,
// "An Introduction to Solid Modeling" §12 (boolean set operations, coincident faces).

// coplanar reports whether two planar faces lie in the same plane (parallel normals and a
// shared point), so their overlap is an area rather than the line an imprint would find.
func coplanar(a, b planarFace) bool {
	if stdmath.Abs(a.normal.Dot(b.normal)) < 1-1e-7 { // tol:angular — parallel-normals cosine
		return false // not parallel
	}
	return stdmath.Abs(b.plane.Origin.VectorTo(a.plane.Origin).Dot(b.normal)) < 1e-7 // tol:calibrated — coplanar gap (see arrange2d arrTol)
}

// faceEdges3D returns all of a face's loop edges as 3D segments — imprinted onto a coplanar
// face so the shared-overlap region splits off as its own sub-face.
func faceEdges3D(f planarFace) [][2]math.Point3 {
	var segs [][2]math.Point3
	for _, ring := range f.loops {
		n := len(ring)
		for i := range n {
			segs = append(segs, [2]math.Point3{ring[i], ring[(i+1)%n]})
		}
	}
	return segs
}

// coplanarOverlapSegments clips the other coplanar face's boundary edges to THIS face's
// material region, keeping only the in-material portions (and dropping any lying on f's own
// boundary). A coplanar union/cut only interacts over the shared overlap; imprinting the
// other outline WHOLE injects segments that fall inside f's holes or beyond its outer loop —
// e.g. a bar whose footprint sits inside a bored face's hole — which corrupt the 2D
// arrangement into spurious sub-faces and a doubled coincident membrane (#860). Clipping each
// edge to f's material (like the non-coplanar imprint already does via faceLineIntervals)
// keeps only the segments that actually split f.
func coplanarOverlapSegments(f planarFace, segs [][2]math.Point3) [][2]math.Point3 {
	var out [][2]math.Point3
	for _, s := range segs {
		out = append(out, clipSegmentToMaterial(f, s)...)
	}
	return interiorSegments(f, out)
}

// clipSegmentToMaterial splits a segment (coplanar with f) at its crossings with f's loops and
// returns the sub-segments whose midpoint lies in f's material (inside the outer loop, outside
// every hole).
func clipSegmentToMaterial(f planarFace, s [2]math.Point3) [][2]math.Point3 {
	a2, b2 := to2D(f.plane, s[0]), to2D(f.plane, s[1])
	ab := a2.VectorTo(b2)
	ts := loopCrossingParams(f, geom.NewLineSegment2d(a2, b2))
	var out [][2]math.Point3
	for i := 0; i+1 < len(ts); i++ {
		if ts[i+1]-ts[i] < arrTol {
			continue
		}
		if !pointInFace2D(a2.TranslateBy(ab.Scale((ts[i]+ts[i+1])/2)), f) {
			continue
		}
		p := to3D(f.plane, a2.TranslateBy(ab.Scale(ts[i])))
		q := to3D(f.plane, a2.TranslateBy(ab.Scale(ts[i+1])))
		out = append(out, [2]math.Point3{p, q})
	}
	return out
}

// loopCrossingParams returns the sorted parameters along seg (0 and 1 included) where it
// crosses any of f's loop edges — the cut points that split seg into material/non-material runs.
func loopCrossingParams(f planarFace, seg geom.LineSegment2d) []float64 {
	ts := []float64{0, 1}
	for _, ring := range f.loops {
		n := len(ring)
		for i := range n {
			c, d := to2D(f.plane, ring[i]), to2D(f.plane, ring[(i+1)%n])
			if _, sp, _, ok := geom.Segment2dIntersection(seg, geom.NewLineSegment2d(c, d), arrTol); ok {
				ts = append(ts, sp)
			}
		}
	}
	sort.Float64s(ts)
	return ts
}

// coplanarCover reports whether sub-face point ip is covered by a coplanar face of the other
// solid, and if so whether that face's normal agrees with f's (shared vs. anti-shared).
func coplanarCover(f planarFace, ip math.Point3, others []planarFace) (covered, sameNormal bool) {
	for _, o := range others {
		if coplanar(f, o) && pointInFace2D(to2D(o.plane, ip), o) {
			return true, f.normal.Dot(o.normal) > 0
		}
	}
	return false, false
}

// coplanarKeep is the selection table for a coplanar overlap fragment. The shared boundary
// is emitted once, as A's copy, so B's coplanar fragments always drop. For A: union and
// intersection keep the same-normal (genuinely shared) fragment, while difference keeps the
// anti-shared (flush-cut) fragment and drops the same-normal one (it becomes interior).
func coplanarKeep(op Op, isB, sameNormal bool) bool {
	if isB {
		return false
	}
	if op == Difference {
		return !sameNormal
	}
	return sameNormal // Union, Intersection
}
