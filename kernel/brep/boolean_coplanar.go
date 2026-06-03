// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
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
	if stdmath.Abs(a.normal.Dot(b.normal)) < 1-1e-7 {
		return false // not parallel
	}
	return stdmath.Abs(b.plane.Origin.VectorTo(a.plane.Origin).Dot(b.normal)) < 1e-7
}

// faceEdges3D returns all of a face's loop edges as 3D segments — imprinted onto a coplanar
// face so the shared-overlap region splits off as its own sub-face.
func faceEdges3D(f planarFace) [][2]math.Point3 {
	var segs [][2]math.Point3
	for _, ring := range f.loops {
		n := len(ring)
		for i := 0; i < n; i++ {
			segs = append(segs, [2]math.Point3{ring[i], ring[(i+1)%n]})
		}
	}
	return segs
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
