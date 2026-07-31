// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// CanalFootLocusRail interpolates a canal loft's foot-contact locus — one EXACT rolling-ball
// contact per station — into the same B-spline row LoftCanalStations lofts that side's boundary
// from: chord-length parameters over the station CENTRES (P&T §9.2.1), the loft's own fitDegree,
// averaged knots, and global interpolation THROUGH the contacts (never a smoothing fit, so every
// solved contact stays on the rail). Built from the same (centers, feet) station rows the loft
// consumes, the returned curve's control points and knots are arithmetic-identical to the loft's
// foot pole-row (assembleCanalLoft's interpEuclideanRow), so the rail IS the lofted surface's
// foot-side boundary isoparm — a boundary edge carrying it lies ON the patch to evaluation noise,
// where the degree-1 polyline through the rail NODES it replaces sagged off the patch by the chord
// sagitta (simple/S9: 7.927e-04 of the bounding diagonal; railb-locus-report.md).
//
//	rail, err := geom.CanalFootLocusRail(rc.Centers, rc.FeetB)
func CanalFootLocusRail(centers, feet []math.Point3) (BSplineCurve, error) {
	if len(centers) != len(feet) {
		return BSplineCurve{}, fmt.Errorf(
			"CanalFootLocusRail: %d centres vs %d feet; need one contact per station", len(centers), len(feet))
	}
	vParams, err := alphaParams(coords3(centers), 1) // the loft's chord-length spine parametrization
	if err != nil {
		return BSplineCurve{}, fmt.Errorf("CanalFootLocusRail: station chord-length params: %w", err)
	}
	vDeg, err := fitDegree(len(centers))
	if err != nil {
		return BSplineCurve{}, fmt.Errorf("CanalFootLocusRail: %w", err)
	}
	ctrl, knots, err := fitInterpolationAt(coords3(feet), vDeg, vParams)
	if err != nil {
		return BSplineCurve{}, fmt.Errorf("CanalFootLocusRail: foot-locus interpolation: %w", err)
	}
	return NewBSplineCurveUniformWeights(vDeg, points3(ctrl), knots)
}
