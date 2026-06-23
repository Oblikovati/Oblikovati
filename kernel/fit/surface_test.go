// SPDX-License-Identifier: GPL-2.0-only

package fit

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// sphericalCap samples an n×n grid of points on a sphere of radius r as a height field
// z = sqrt(r²−x²−y²) over x,y ∈ [−half, half] — a scanned spherical-cap region (#1291 acceptance).
func sphericalCap(n int, r, half float64) []math.Point3 {
	pts := make([]math.Point3, 0, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := -half + 2*half*float64(i)/float64(n-1)
			y := -half + 2*half*float64(j)/float64(n-1)
			z := stdmath.Sqrt(r*r - x*x - y*y)
			pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
		}
	}
	return pts
}

// maxDeviation is the largest distance from any point to the fitted surface (via on-surface inverse).
func maxDeviation(s interface {
	ParamAt(math.Point3) (float64, float64)
	PointAt(u, v float64) math.Point3
}, pts []math.Point3) float64 {
	worst := 0.0
	for _, p := range pts {
		u, v := s.ParamAt(p)
		if d := float64(p.DistanceTo(s.PointAt(u, v))); d > worst {
			worst = d
		}
	}
	return worst
}

// TestSurfaceToPointsFitsSphericalCapWithinTolerance is the F15 acceptance: a degree-3 5×5 patch
// fitted to a scanned spherical cap stays within a tight deviation, and the result is a clean
// bicubic NURBS (degree 3 each way, an even 5×5 control net).
func TestSurfaceToPointsFitsSphericalCapWithinTolerance(t *testing.T) {
	const r, half = 10.0, 3.0
	pts := sphericalCap(12, r, half)
	surf, err := SurfaceToPoints(pts, 3, 5, 5)
	if err != nil {
		t.Fatalf("SurfaceToPoints: %v", err)
	}
	if surf.UDegree != 3 || surf.VDegree != 3 {
		t.Errorf("fitted degree = (%d,%d), want (3,3)", surf.UDegree, surf.VDegree)
	}
	if len(surf.Ctrl) != 5 || len(surf.Ctrl[0]) != 5 {
		t.Errorf("control net = %dx%d, want 5x5", len(surf.Ctrl), len(surf.Ctrl[0]))
	}
	if dev := maxDeviation(surf, pts); dev > 0.02*r { // within 2% of radius across the whole region
		t.Errorf("max deviation %.4g exceeds tolerance %.4g", dev, 0.02*r)
	}
}

func TestSurfaceToPointsRejectsCollinear(t *testing.T) {
	pts := make([]math.Point3, 30)
	for i := range pts { // all on the x-axis: no base plane
		pts[i] = math.P3(math.Scalar(i), 0, 0)
	}
	if _, err := SurfaceToPoints(pts, 3, 5, 5); err == nil {
		t.Error("collinear points should error (no base plane)")
	}
}

func TestSurfaceToPointsRejectsTooFewPoints(t *testing.T) {
	pts := sphericalCap(4, 10, 3) // 16 points, fewer than a 5x5=25 control net
	if _, err := SurfaceToPoints(pts, 3, 5, 5); err == nil {
		t.Error("fitting 25 control points to 16 points should error")
	}
}
