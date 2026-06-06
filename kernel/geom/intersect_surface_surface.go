// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati/math"

// IntersectSurfaceSurface traces the intersection curve(s) of two surfaces as polylines on
// the base surface. It marches the zero set of the signed distance from base's points to
// other over base's parameter grid (PBI-244), so every returned point lies on base
// exactly and on other to tolerance. grid bounds the base parameter window — supply an
// explicit window for an unbounded base direction (a plane/cylinder/cone); a fully bounded
// base (sphere/torus) may pass the zero SurfaceGrid for defaults.
//
// Example — a sphere cut by the z=0 plane yields its equator circle:
//
//	sp, _ := geom.NewSphere(math.P3(0,0,0), 5)
//	pl, _ := geom.NewPlane(math.P3(0,0,0), math.V3(0,0,1))
//	loops := geom.IntersectSurfaceSurface(sp, pl, geom.SurfaceGrid{}) // one loop on z=0
func IntersectSurfaceSurface(base, other Surface, grid SurfaceGrid) [][]math.Point3 {
	field := func(u, v float64) float64 {
		return SignedDistanceToSurface(other, base.PointAt(u, v))
	}
	return traceZeroOnSurface(base, field, grid)
}
