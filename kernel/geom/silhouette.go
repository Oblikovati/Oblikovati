// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati/math"

// Silhouette traces the silhouette (contour generator) of a surface for a given view
// direction as polylines: the locus where the surface normal is perpendicular to the view
// (n · viewDir = 0), the apparent outline under a parallel projection along viewDir
// (PBI-244). grid bounds the parameter window as in [IntersectSurfaceSurface].
//
// Example — a sphere's silhouette for any view is a great circle:
//
//	sp, _ := geom.NewSphere(math.P3(0,0,0), 5)
//	loops := geom.Silhouette(sp, math.V3(0,0,1), geom.SurfaceGrid{}) // the z=0 equator
func Silhouette(s Surface, viewDir math.Vector3, grid SurfaceGrid) [][]math.Point3 {
	field := func(u, v float64) float64 {
		return float64(s.NormalAt(u, v).Dot(viewDir))
	}
	return traceZeroOnSurface(s, field, grid)
}
