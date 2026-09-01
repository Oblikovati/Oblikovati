// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Network surface body (M36-F10): fit each U- and V-direction polyline to a B-spline, build the
// Gordon network surface that interpolates the grid (geom.NetworkSurface), and wrap it as a
// single-face surface body. The curves must approximately intersect at a regular grid.

// NetworkSurfaceBody builds a surface body from the U- and V-direction curve polylines. Each polyline
// is fitted to a B-spline; the result interpolates the curve grid. It errors when a polyline is too
// short to fit or the curves do not form a usable grid.
func NetworkSurfaceBody(uPolylines, vPolylines [][]math.Point3) (*topo.Body, error) {
	uCurves, err := fitCurves(uPolylines, "u")
	if err != nil {
		return nil, err
	}
	vCurves, err := fitCurves(vPolylines, "v")
	if err != nil {
		return nil, err
	}
	surf, err := geom.NetworkSurface(uCurves, vCurves)
	if err != nil {
		return nil, err
	}
	return retopo.FullDomainBody(surf, "network"), nil
}

// fitCurves fits each polyline to a B-spline curve, erroring on a degenerate (too-short) input.
func fitCurves(polylines [][]math.Point3, dir string) ([]geom.BSplineCurve, error) {
	out := make([]geom.BSplineCurve, len(polylines))
	for i, pts := range polylines {
		c, err := geom.NewFittedBSplineCurve(pts)
		if err != nil {
			return nil, fmt.Errorf("ops.NetworkSurfaceBody: %s-curve %d: %w", dir, i, err)
		}
		out[i] = c
	}
	return out, nil
}
