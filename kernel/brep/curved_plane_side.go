// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Which side of a plane, and when two points on a section are one point. These outlive the analytic
// half-space pipeline ADR-0061 deleted (they were the shared helpers of its general stage) and are used
// by the ruled-face boundary walk, the drill and the coaxial builders.

// faceSample returns a point on the face (a boundary vertex, or a surface point for a boundary-less
// face) whose plane side reports the whole face's side when the plane does not cross it.
func faceSample(f curvedFace) math.Point3 {
	if len(f.loops) > 0 && len(f.loops[0].edges) > 0 {
		return f.loops[0].edges[0].start()
	}
	return f.surface.PointAt(0, 0)
}

// signedDistance returns n·(p − plane.Origin): negative on the kept side.
func signedDistance(p math.Point3, plane geom.Plane, n math.Vector3) float64 {
	return float64(plane.Origin.VectorTo(p).Dot(n))
}

// samePoint reports whether two points coincide within the model-relative weld tolerance (#1399).
// Section/trace endpoints carry more accumulated round-off than an exact vertex weld, so they merge at
// the looser on-line tolerance res.Plane() (1e-7 at unit scale) rather than res.Weld(); deriving it from
// the body's extent keeps loop-closure and arc-chaining watertight on a km-scale part.
func samePoint(a, b math.Point3, res geom.Resolution) bool {
	return float64(a.DistanceTo(b)) < res.Plane()
}
