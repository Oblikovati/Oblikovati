// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// SamplePolyline3D returns a model-space polyline approximating a 3D sketch entity —
// the shared shape the viewport overlay draws and the ray picker tests clicks
// against (issue #142: 3D sketch geometry must be visible and pickable for the
// constraint tools). segments bounds the sample count of curved entities; a point
// returns its single position, a line its two endpoints, and an unknown or
// degenerate entity nil.
//
//	pts := sketch.SamplePolyline3D(entity, 64)
func SamplePolyline3D(e Entity, segments int) []math.Point3 {
	if segments < 2 {
		segments = 2
	}
	switch v := e.(type) {
	case *Point3D:
		return []math.Point3{v.Position()}
	case *Line3D:
		return []math.Point3{v.A.Position(), v.B.Position()}
	case *Spline3D:
		return v.Sample()
	case *FixedSpline3D:
		return v.Sample()
	case *EquationCurve3D:
		return v.Sample(segments)
	default:
		return sampleCurve3D(e, segments)
	}
}

// curve3At is the kernel-curve evaluation shape shared by circle/arc/helix/conics.
type curve3At interface{ PointAt(t float64) math.Point3 }

// sampleCurve3D samples the analytic kernel curve of a circle/arc/helix/conic entity
// uniformly over t ∈ [0,1]; a helix gets segments per turn so a long spring stays
// round. Returns nil when the entity has no curve (unknown kind or degenerate).
func sampleCurve3D(e Entity, segments int) []math.Point3 {
	cu, n := analyticCurve3D(e, segments)
	if cu == nil {
		return nil
	}
	pts := make([]math.Point3, n+1)
	for i := 0; i <= n; i++ {
		pts[i] = cu.PointAt(float64(i) / float64(n))
	}
	return pts
}

// analyticCurve3D resolves an entity's kernel curve and its sample count.
func analyticCurve3D(e Entity, segments int) (curve3At, int) {
	switch v := e.(type) {
	case *Circle3D:
		if cu, err := v.Curve(); err == nil {
			return cu, segments
		}
	case *Arc3D:
		if cu, err := v.Curve(); err == nil {
			return cu, segments
		}
	case *HelicalCurve3D:
		if cu, err := v.Curve(); err == nil {
			return cu, helixSampleCount(v.Turns, segments)
		}
	case *Ellipse3D:
		if cu, err := v.Curve(); err == nil {
			return cu, segments
		}
	case *EllipticalArc3D:
		if cu, err := v.Curve(); err == nil {
			return cu, segments
		}
	}
	return nil, 0
}

// helixSampleCount scales the sample budget with the turn count (min one segment
// budget per turn, capped at 16 turns so a pathological helix stays bounded).
func helixSampleCount(turns float64, segments int) int {
	n := min(max(int(turns), 1), 16)
	return n * segments
}
