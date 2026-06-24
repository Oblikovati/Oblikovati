// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// IntersectSurfaceSurface returns the intersection curve(s) of two surfaces as polylines, every point
// of which lies on BOTH surfaces to tolerance. It uses a predictor–corrector continuation tracer
// (traceIntersectionCurves, Oblikovati#1319): it marches the joint zero of the two surfaces, so it
// finds closed interior loops, keeps saddle branches connected, and reports a tangential contact as a
// point — the known failure modes of the old fixed-grid marching-squares contour, which survives only
// as a fallback for any configuration the continuation seeding misses. grid bounds the base parameter
// window — supply an explicit window for an unbounded base direction (a plane/cylinder/cone); a fully
// bounded base (sphere/torus) may pass the zero SurfaceGrid for defaults.
//
// Example — a sphere cut by the z=0 plane yields its equator circle:
//
//	sp, _ := geom.NewSphere(math.P3(0,0,0), 5)
//	pl, _ := geom.NewPlane(math.P3(0,0,0), math.V3(0,0,1))
//	loops := geom.IntersectSurfaceSurface(sp, pl, geom.SurfaceGrid{}) // one loop on z=0
func IntersectSurfaceSurface(base, other Surface, grid SurfaceGrid) [][]math.Point3 {
	if curves := traceIntersectionCurves(base, other, grid); len(curves) > 0 {
		return curves
	}
	field := func(u, v float64) float64 {
		return SignedDistanceToSurface(other, base.PointAt(u, v))
	}
	return traceZeroOnSurface(base, field, grid)
}
