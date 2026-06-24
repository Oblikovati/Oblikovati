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

// cylinderAxisTol is how close |n·axis| must be to 1 for a cut plane to count as perpendicular to
// the cylinder axis (a constant-axial-coordinate cut). Looser than that is an oblique section.
const cylinderAxisTol = 1e-7

// perpendicularToAxis reports whether the cut plane normal is parallel to the cylinder axis (a
// constant-axial-coordinate cut), the only orientation the fast SolidCylinder path handles; an
// axis-parallel or oblique cut routes to the general arrangement instead.
func perpendicularToAxis(n math.Vector3, cyl geom.Cylinder) bool {
	along := float64(n.Dot(cyl.AxisDir.AsVector()))
	return stdmath.Abs(along) >= 1-cylinderAxisTol
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

// cylinderHalfSpace keeps the axial band of the cylinder on the plane's negative side. The plane
// must be perpendicular to the axis; the kept band is rebuilt as a shorter SolidCylinder (exact
// cylinder side + planar caps). A plane clear of the cylinder keeps it whole or empties it.
func cylinderHalfSpace(body *topo.Body, cyl geom.Cylinder, base math.Point3, height float64, plane geom.Plane) (*topo.Body, error) {
	n := unit(plane.Normal())
	axis := cyl.AxisDir.AsVector()
	along := float64(n.Dot(axis))
	if stdmath.Abs(along) < 1-cylinderAxisTol {
		return nil, ErrUnsupportedHalfSpace // oblique cut: elliptical section, deferred
	}
	// Axial coordinate (from base, along +axis) where the cut plane sits, and the kept sub-band.
	cut := float64(base.VectorTo(plane.Origin).Dot(axis))
	lo, hi := keptAxialBand(cut, height, along > 0)
	if hi-lo <= cylinderAxisTol {
		return topo.MergeBodies(topo.NewLineage(topo.Tok("halfspace", "empty", 0)), true), nil
	}
	if hi-lo >= height-cylinderAxisTol {
		return body, nil // plane clears the cylinder on the kept side
	}
	newBase := base.TranslateBy(axis.Scale(math.Scalar(lo)))
	return SolidCylinder(newBase, axis, cyl.Radius, hi-lo)
}

// keptAxialBand returns the [lo, hi] axial interval (within [0, height]) kept on the plane's
// negative side. With n along +axis the negative side is below the cut (toward the base); with n
// along −axis it is above (toward the top).
func keptAxialBand(cut, height float64, nAlongAxis bool) (lo, hi float64) {
	cut = stdmath.Max(0, stdmath.Min(height, cut))
	if nAlongAxis {
		return 0, cut // g = s − cut ≤ 0 ⇒ s ≤ cut
	}
	return cut, height // g = cut − s ≤ 0 ⇒ s ≥ cut
}
