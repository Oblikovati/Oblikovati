// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// capPoints samples an n×n spherical-cap height field for the fit body tests.
func capPoints(n int, r, half float64) []math.Point3 {
	pts := make([]math.Point3, 0, n*n)
	for i := range n {
		for j := range n {
			x := -half + 2*half*float64(i)/float64(n-1)
			y := -half + 2*half*float64(j)/float64(n-1)
			pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(stdmath.Sqrt(r*r-x*x-y*y))))
		}
	}
	return pts
}

func TestFitSurfaceToPointsBuildsOneFace(t *testing.T) {
	t.Parallel()
	body, err := FitSurfaceToPoints(capPoints(12, 10, 3), 3, 5, 5)
	if err != nil {
		t.Fatalf("FitSurfaceToPoints: %v", err)
	}
	if got := len(body.Faces()); got != 1 {
		t.Fatalf("fit body has %d faces, want 1", got)
	}
}

func TestFitSurfaceToPointsRejectsSparseRegion(t *testing.T) {
	t.Parallel()
	if _, err := FitSurfaceToPoints(capPoints(4, 10, 3), 3, 5, 5); err == nil {
		t.Error("a region with fewer points than control points should error")
	}
}
