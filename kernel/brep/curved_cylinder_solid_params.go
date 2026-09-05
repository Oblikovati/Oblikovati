// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cylinder half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). Trims a closed analytic
// cylinder (one cylindrical side + two planar caps) by a plane PERPENDICULAR to its axis: the kept
// axial band is itself a shorter cylinder, exact and watertight. An oblique plane (an elliptical
// section, a wedge result) needs the general curved arrangement and returns ErrUnsupportedHalfSpace
// so the caller keeps the CSG fallback.

// CylinderParams recovers a bare cylinder's surface, base-cap centre and height from a body, for callers
// (the curved subtract) that need the axis frame before building. ok=false unless the body is exactly one
// cylindrical side plus two planar caps. It mirrors cylinderSolidParams, exported for kernel/ops.
func CylinderParams(body *topo.Body) (cyl geom.Cylinder, base math.Point3, height float64, ok bool) {
	return cylinderSolidParams(facesOfAny(body))
}

// cylinderSolidParams recovers a closed cylinder's geometry from its flattened faces: the side's
// geom.Cylinder, the base centre (axially lowest cap), and the height. ok=false unless the body is
// exactly one cylindrical side plus two planar caps (a bare SolidCylinder), the only shape this
// perpendicular-cut path handles.
func cylinderSolidParams(faces []curvedFace) (cyl geom.Cylinder, base math.Point3, height float64, ok bool) {
	var caps []geom.Plane
	found := false
	for _, f := range faces {
		switch s := f.surface.(type) {
		case geom.Cylinder:
			cyl, found = s, true
		case geom.Plane:
			caps = append(caps, s)
		default:
			return geom.Cylinder{}, math.Point3{}, 0, false
		}
	}
	if !found || len(caps) != 2 {
		return geom.Cylinder{}, math.Point3{}, 0, false
	}
	base, height = cylinderExtent(cyl, caps)
	return cyl, base, height, true
}

// cylinderExtent returns the axially-lowest cap centre and the cylinder height, projecting both cap
// origins onto the axis from the side's origin (so it is robust to which cap geom.Cylinder.Origin
// sits on).
func cylinderExtent(cyl geom.Cylinder, caps []geom.Plane) (base math.Point3, height float64) {
	axis := cyl.AxisDir.AsVector()
	s0 := float64(cyl.Origin.VectorTo(caps[0].Origin).Dot(axis))
	s1 := float64(cyl.Origin.VectorTo(caps[1].Origin).Dot(axis))
	lo, hi := stdmath.Min(s0, s1), stdmath.Max(s0, s1)
	return cyl.Origin.TranslateBy(axis.Scale(math.Scalar(lo))), hi - lo
}
