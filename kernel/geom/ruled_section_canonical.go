// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Canonicalising a ruled-quadric section (ADR-0061 stage 4).
//
// The one general intersector builds every ruled∩quadric section as a RuledQuadricArc. Some of those
// sections ARE circles — a sphere cut by a cylinder through its centre, a bore through a plate — and
// delivering one as a general ruled arc loses everything downstream that reads the curve's KIND: the
// per-face oracle's band walk, the tessellator's conformal rim stations, the provenance name of a bore
// rim, every conic clip. Recognising the section is not a type-pair fast path beside the intersector;
// it asks the CURVE what it is, after the general path has built it.

// ruledSectionCircleProbes is how many azimuths the circle recognition certifies against. The branch is
// sampled over its full sweep, so a section that departs from the circle anywhere is caught.
const ruledSectionCircleProbes = 24

// canonicalSection presents a full-sweep section branch in its most specific exact form: a branch whose
// every point lies on ONE circle comes back as that Circle, anything else as itself.
func canonicalSection(a RuledQuadricArc, res Resolution) Curve3 {
	pts := make([]math.Point3, ruledSectionCircleProbes)
	for i := range pts {
		pts[i] = a.PointAt(float64(i) / ruledSectionCircleProbes)
	}
	if c, ok := circleThrough(pts, res.Weld()); ok {
		return c
	}
	return a
}

// circleThrough returns the circle every sample lies on, to within tol. The samples are evenly spaced
// over a full turn of the base azimuth, so their centroid IS the circle's centre when they lie on one —
// and when they do not, the certification below rejects it.
func circleThrough(pts []math.Point3, tol float64) (Circle, bool) {
	if len(pts) < 3 {
		return Circle{}, false
	}
	center := pointCentroid(pts)
	normal, ok := canonicalPlaneNormal(pts, center)
	if !ok {
		return Circle{}, false
	}
	radius := float64(center.DistanceTo(pts[0]))
	if radius <= tol {
		return Circle{}, false
	}
	for _, p := range pts {
		if stdmath.Abs(float64(center.DistanceTo(p))-radius) > tol {
			return Circle{}, false
		}
		if stdmath.Abs(float64(center.VectorTo(p).Dot(normal.AsVector()))) > tol {
			return Circle{}, false
		}
	}
	c, err := NewCircle(center, normal.AsVector(), radius)
	return c, err == nil
}

// pointCentroid is the average of the samples.
func pointCentroid(pts []math.Point3) math.Point3 {
	var x, y, z float64
	for _, p := range pts {
		x, y, z = x+float64(p.X), y+float64(p.Y), z+float64(p.Z)
	}
	n := float64(len(pts))
	return math.P3(math.Scalar(x/n), math.Scalar(y/n), math.Scalar(z/n))
}

// canonicalPlaneNormal is the samples' plane normal with a SIGN fixed by a total order on its
// components, so the two operands of a boolean — which build the same section from opposite bases —
// derive the same circle, seam included, rather than two circles that differ by a reflection.
func canonicalPlaneNormal(pts []math.Point3, center math.Point3) (math.UnitVector3, bool) {
	n, err := math.UnitVector3FromVector(center.VectorTo(pts[0]).Cross(center.VectorTo(pts[len(pts)/4])))
	if err != nil {
		return math.UnitVector3{}, false
	}
	v := n.AsVector()
	for _, c := range []math.Scalar{v.X, v.Y, v.Z} {
		if c > 0 {
			return n, true
		}
		if c < 0 {
			u, err := math.UnitVector3FromVector(v.Scale(-1))
			return u, err == nil
		}
	}
	return math.UnitVector3{}, false
}
