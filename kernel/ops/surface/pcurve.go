// SPDX-License-Identifier: GPL-2.0-only

package surface

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// marchUV builds a face boundary's pcurve in the surface's (u,v): the first point is projected
// with a full grid search, then each subsequent point is projected SEEDED from the previous
// point's (u,v) (geom.BSplineSurface.ParamNear). Seeding keeps the curve on one smooth branch, so
// it does not self-intersect where independent ParamAt would — adjacent points snapping to
// different local minima near a near-fold (the imported edges sit ~mm off the surface, ADR-0030).
// A non-self-intersecting boundary is the prerequisite for a non-folding interior triangulation
// (M24 F02) and a reliable point-in-trim test. The 3D positions are unchanged — this only assigns
// each point a smooth (u,v).
func marchUV(s geom.BSplineSurface, loop []math.Point3) []math.Point2 {
	out := make([]math.Point2, len(loop))
	if len(loop) == 0 {
		return out
	}
	pu, pv := s.ParamAt(loop[0])
	out[0] = math.P2(math.Scalar(pu), math.Scalar(pv))
	for i := 1; i < len(loop); i++ {
		pu, pv = s.ParamNear(loop[i], pu, pv)
		out[i] = math.P2(math.Scalar(pu), math.Scalar(pv))
	}
	return out
}
