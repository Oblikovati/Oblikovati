// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

// 3D spline sampling mirrors the 2D approach (curvesample_spline.go): a spline stores only
// its defining points, and a representative smooth polyline through them is produced with
// a uniform Catmull–Rom (passes through the points, natural for closed loops). Exact
// approximating NURBS fidelity is a kernel B-rep-curve concern, deferred; this polyline
// only has to follow the curve closely enough for paths/region detection and faceting.

// catmullRom3DOpen samples an open 3D point chain, duplicating the endpoints as their own
// phantom neighbours so the curve starts/ends exactly on the first/last point.
func catmullRom3DOpen(pts []math.Point3) []math.Point3 {
	n := len(pts)
	out := []math.Point3{}
	for i := 0; i < n-1; i++ {
		out = append(out, spanSamples3D(pts[clampIndex(i-1, n-1)], pts[i], pts[i+1], pts[clampIndex(i+2, n-1)])...)
	}
	return append(out, pts[n-1])
}

// catmullRom3DClosed samples a closed 3D point chain, wrapping neighbours around the loop
// (no duplicate closing vertex).
func catmullRom3DClosed(pts []math.Point3) []math.Point3 {
	n := len(pts)
	out := []math.Point3{}
	for i := 0; i < n; i++ {
		out = append(out, spanSamples3D(pts[(i-1+n)%n], pts[i], pts[(i+1)%n], pts[(i+2)%n])...)
	}
	return out
}

// spanSamples3D returns the per-span sample points of the p1→p2 span (with neighbours
// p0,p3), using the same resolution as the 2D sampler.
func spanSamples3D(p0, p1, p2, p3 math.Point3) []math.Point3 {
	out := make([]math.Point3, splineSamplesPerSpan)
	for j := 0; j < splineSamplesPerSpan; j++ {
		out[j] = catmullRom3DPoint(p0, p1, p2, p3, float64(j)/float64(splineSamplesPerSpan))
	}
	return out
}

// catmullRom3DPoint evaluates the uniform Catmull–Rom spline of the p1→p2 span at local
// parameter t∈[0,1].
func catmullRom3DPoint(p0, p1, p2, p3 math.Point3, t float64) math.Point3 {
	t2, t3 := t*t, t*t*t
	return math.P3(
		catmullRom1D(p0.X, p1.X, p2.X, p3.X, math.Scalar(t), math.Scalar(t2), math.Scalar(t3)),
		catmullRom1D(p0.Y, p1.Y, p2.Y, p3.Y, math.Scalar(t), math.Scalar(t2), math.Scalar(t3)),
		catmullRom1D(p0.Z, p1.Z, p2.Z, p3.Z, math.Scalar(t), math.Scalar(t2), math.Scalar(t3)),
	)
}

// catmullRom1D is the per-coordinate uniform Catmull–Rom basis blend.
func catmullRom1D(a, b, c, d, t, t2, t3 math.Scalar) math.Scalar {
	return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
}

// sampleChain3D samples a 3D point chain as open or closed; a chain shorter than three
// points is just its control polygon.
func sampleChain3D(pts []math.Point3, closed bool) []math.Point3 {
	if len(pts) < 3 {
		return pts
	}
	if closed {
		return catmullRom3DClosed(pts)
	}
	return catmullRom3DOpen(pts)
}
