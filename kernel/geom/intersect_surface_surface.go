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
	return TraceSurfaceIntersection(base, other, grid).Curves
}

// SurfaceIntersection carries the intersection curves plus their provenance: ViaFallback reports that
// the continuation tracer found no curve and the fixed-grid marching-squares contour supplied the
// result. Fallback curves are contour-quality — sub-grid loops lost, tangencies invisible — so a
// caller holding a diag recorder must surface that degradation instead of shipping it silently
// (#1597; vacuously false when there is no intersection at all).
type SurfaceIntersection struct {
	Curves      [][]math.Point3
	ViaFallback bool
}

// TraceSurfaceIntersection is IntersectSurfaceSurface with provenance, for callers that need to see —
// not just receive — a degraded fallback result.
func TraceSurfaceIntersection(base, other Surface, grid SurfaceGrid) SurfaceIntersection {
	if curves := traceIntersectionCurves(base, other, grid); len(curves) > 0 {
		return SurfaceIntersection{Curves: curves}
	}
	field := func(u, v float64) float64 {
		return SignedDistanceToSurface(other, base.PointAt(u, v))
	}
	curves := traceZeroOnSurface(base, field, grid)
	return SurfaceIntersection{Curves: curves, ViaFallback: len(curves) > 0}
}
