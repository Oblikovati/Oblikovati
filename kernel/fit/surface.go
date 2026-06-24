// SPDX-License-Identifier: GPL-2.0-only

package fit

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Surface fitting (M36-F15): turn a region of a scanned point cloud / mesh into a clean Class-A NURBS
// surface. The region's points are parametrized by projecting them onto their least-squares base
// plane (PCA) — the plane's in-plane axes give each point a (u,v), normalized to [0,1] — and then a
// degree-3 tensor B-spline of a chosen, modest control count is least-squares fitted through them
// (geom.ApproximateSurfaceLS). Few even spans is what makes it Class-A; the achieved deviation is
// reported separately (model/analysis F14). This is the kernel half of reverse-engineering styling
// from scan data (#1291).

// SurfaceToPoints fits a degree×degree B-spline with nu×nv control points to a point-cloud region.
// It errors when the points do not determine a base plane (collinear/too few) or the control count
// is too high for the region (see geom.ApproximateSurfaceLS). Example: SurfaceToPoints(region, 3,
// 5, 5) fits a clean bicubic 5×5 patch.
func SurfaceToPoints(points []math.Point3, degree, nu, nv int) (geom.BSplineSurface, error) {
	plane, err := Plane(points)
	if err != nil {
		return geom.BSplineSurface{}, fmt.Errorf("fit.SurfaceToPoints: base plane: %w", err)
	}
	us, vs := planeParameters(points, plane)
	return geom.ApproximateSurfaceLS(points, us, vs, degree, degree, nu, nv)
}

// planeParameters projects each point onto the base plane's U/V axes and normalizes both coordinates
// to [0,1], giving the surface parameters the least-squares fit interpolates against. A degenerate
// (zero-extent) direction collapses to 0.5 so the parametrization stays well-defined.
func planeParameters(points []math.Point3, plane geom.Plane) (us, vs []float64) {
	us = make([]float64, len(points))
	vs = make([]float64, len(points))
	for i, p := range points {
		d := plane.Origin.VectorTo(p)
		us[i] = float64(d.Dot(plane.UAxis.AsVector()))
		vs[i] = float64(d.Dot(plane.VAxis.AsVector()))
	}
	normalizeUnit(us)
	normalizeUnit(vs)
	return us, vs
}

// normalizeUnit rescales values in place to [0,1] by their min/max; a zero range maps all to 0.5.
func normalizeUnit(vals []float64) {
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		lo, hi = min(lo, v), max(hi, v)
	}
	span := hi - lo
	for i, v := range vals {
		if span == 0 {
			vals[i] = 0.5
			continue
		}
		vals[i] = (v - lo) / span
	}
}
