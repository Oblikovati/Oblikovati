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
//
// A face that is not planar is coplanar with nothing. The kind is checked FIRST because a cylinder's
// NormalAt(0,0) is a perfectly good unit vector that can pass the parallel test, and the plane was
// then taken from it — the mixed boolean's coplanar cover screens every face of the other operand,
// cylinders included, and panicked in the type assertion on the slotted screw's cross-hole (#3459).
func coplanar(a, b curvedFace) bool {
	pa, aok := planeOf(a)
	pb, bok := planeOf(b)
	if !aok || !bok {
		return false
	}
	if stdmath.Abs(faceNormal(a).Dot(faceNormal(b))) < 1-1e-7 { // tol:angular — parallel-normals cosine
		return false // not parallel
	}
	return stdmath.Abs(pb.Origin.VectorTo(pa.Origin).Dot(faceNormal(b))) < 1e-7 // tol:calibrated — coplanar gap (see arrange2d arrTol)
}

// faceEdges3D returns all of a face's loop edges as 3D segments — imprinted onto a coplanar
// face so the shared-overlap region splits off as its own sub-face.
func faceEdges3D(f curvedFace) [][2]math.Point3 {
	var segs [][2]math.Point3
	for _, ring := range planarRings(f) {
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
func coplanarOverlapSegments(f curvedFace, segs [][2]math.Point3) [][2]math.Point3 {
	var out [][2]math.Point3
	for _, s := range segs {
		out = append(out, clipSegmentToMaterial(f, s)...)
	}
	return interiorSegments(f, out)
}

// clipSegmentToMaterial splits a segment (coplanar with f) at its crossings with f's loops and
// returns the sub-segments whose midpoint lies in f's material (inside the outer loop, outside
// every hole).
func clipSegmentToMaterial(f curvedFace, s [2]math.Point3) [][2]math.Point3 {
	a2, b2 := to2D(facePlane(f), s[0]), to2D(facePlane(f), s[1])
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
		p := to3D(facePlane(f), a2.TranslateBy(ab.Scale(ts[i])))
		q := to3D(facePlane(f), a2.TranslateBy(ab.Scale(ts[i+1])))
		out = append(out, [2]math.Point3{p, q})
	}
	return out
}

// loopCrossingParams returns the sorted parameters along seg (0 and 1 included) where it
// crosses any of f's loop edges — the cut points that split seg into material/non-material runs.
func loopCrossingParams(f curvedFace, seg geom.LineSegment2d) []float64 {
	ts := []float64{0, 1}
	for _, ring := range planarRings(f) {
		n := len(ring)
		for i := range n {
			c, d := to2D(facePlane(f), ring[i]), to2D(facePlane(f), ring[(i+1)%n])
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
//
// degenerate is true when ip lies within band of the BOUNDARY of a coplanar face of the other solid.
// That is exactly the condition under which a volumetric query at ip cannot be answered: every ray
// from ip pierces that face at t≈0 and grazes its boundary (rayGrazes → nearFaceBoundary), so no
// direction is clean, and the winding fallback zeroes the same face by design. Away from the
// boundary the plain query is sound, which matters — the two-sided probe costs two whole-body casts
// instead of one, and this is the boolean's hot loop (#3459).
func coplanarCover(f curvedFace, ip math.Point3, others []curvedFace, band float64) (covered, sameNormal, degenerate bool) {
	for _, o := range others {
		if !coplanar(f, o) {
			continue
		}
		if !degenerate && nearFaceBoundary(o, ip, band) {
			degenerate = true
		}
		if pointInFace2D(to2D(facePlane(o), ip), o) {
			return true, faceNormal(f).Dot(faceNormal(o)) > 0, degenerate
		}
	}
	return false, false, degenerate
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
