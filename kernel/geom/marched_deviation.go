// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// A marched intersection polyline is a CHORD approximation of the true intersection curve: the tracer
// lands every VERTEX on both surfaces, but the straight chord between two vertices bows off the curve.
// That bow is why a marched body's boundary description is slightly inconsistent — its outward vector
// area no longer cancels to zero, and its measured volume misses the exact one — so the marcher must
// REPORT the bow as a measured output instead of leaving a consumer to assume exactness (ground rule:
// "achieved tolerance is a measured output of an operation, stored on the entity"; #3489).

// MarchedDeviation measures the achieved deviation of a marched polyline from the true a∩b intersection
// curve, in model units. A point lies on the intersection curve if and only if it lies on BOTH surfaces,
// so each chord is probed at its midpoint — the chord's farthest excursion from the curve it subtends —
// and scored by the LARGER of that midpoint's two surface distances. The result is the worst chord's
// score: an estimate of how far this polyline's boundary sits off the exact one. It is 0 for a
// degenerate input (fewer than two points), and 0 only to the extent the chords really do lie on both
// surfaces.
//
// Example — a coarse march of two crossing unit cylinders reports its own chord bow:
//
//	loops := geom.IntersectSurfaceSurface(cylZ, cylX, geom.SurfaceGrid{})
//	dev := geom.MarchedDeviation(cylZ, cylX, loops[0]) // ~1e-4 for a few-hundred-point saddle loop
func MarchedDeviation(a, b Surface, pts []math.Point3) float64 {
	worst := 0.0
	for i := 0; i+1 < len(pts); i++ {
		mid := pts[i].Midpoint(pts[i+1])
		worst = stdmath.Max(worst, stdmath.Max(distanceToSurface(a, mid), distanceToSurface(b, mid)))
	}
	return worst
}

// marchedCurvesDeviation is [MarchedDeviation] over a whole traced curve set — the single achieved
// tolerance of one surface-intersection result, which is as exact as its WORST curve.
func marchedCurvesDeviation(a, b Surface, curves [][]math.Point3) float64 {
	worst := 0.0
	for _, pts := range curves {
		worst = stdmath.Max(worst, MarchedDeviation(a, b, pts))
	}
	return worst
}

// distanceToSurface is the unsigned distance from p to its closest point on s. It reads the foot of the
// perpendicular directly rather than [SignedDistanceToSurface]'s normal projection, because a deviation
// is a magnitude: which side of the surface the chord bows to carries no information here.
func distanceToSurface(s Surface, p math.Point3) float64 {
	_, _, foot := ClosestPointOnSurface(s, p)
	return float64(foot.DistanceTo(p))
}

// CurveDeviation returns the achieved tolerance a curve carries: the marched chord deviation of a
// [Polyline] (by value or by pointer — the curved boolean carries imprint loops by identity so its
// run-merge can compare them with `==` — and through a ReverseCurve3 wrapper), and 0 for every analytic
// curve, which describes its geometry EXACTLY. It is the one place outside the intersector that reads a
// curve's achieved tolerance, so consumers in topo/brep never type-assert on a geometry kind themselves.
//
// Example:
//
//	tol := geom.CurveDeviation(edge.Geometry()) // 0 for a Circle, >0 for a marched intersection edge
func CurveDeviation(c Curve3) float64 {
	switch pl := InnerCurve(c).(type) {
	case Polyline:
		return pl.Deviation
	case *Polyline:
		return pl.Deviation
	}
	return 0
}
