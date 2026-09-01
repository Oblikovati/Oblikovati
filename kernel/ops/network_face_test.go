// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// flatGrid returns 3 U-polylines and 3 V-polylines forming a flat 2×2 grid in z=0.
func flatGrid() (u, v [][]math.Point3) {
	pt := func(x, y float64) math.Point3 { return math.P3(math.Scalar(x), math.Scalar(y), 0) }
	u = [][]math.Point3{
		{pt(0, 0), pt(1, 0), pt(2, 0)},
		{pt(0, 1), pt(1, 1), pt(2, 1)},
		{pt(0, 2), pt(1, 2), pt(2, 2)},
	}
	v = [][]math.Point3{
		{pt(0, 0), pt(0, 1), pt(0, 2)},
		{pt(1, 0), pt(1, 1), pt(1, 2)},
		{pt(2, 0), pt(2, 1), pt(2, 2)},
	}
	return u, v
}

func TestNetworkSurfaceBodyBuildsOneFace(t *testing.T) {
	t.Parallel()
	u, v := flatGrid()
	body, err := NetworkSurfaceBody(u, v)
	if err != nil {
		t.Fatalf("NetworkSurfaceBody: %v", err)
	}
	if got := len(body.Faces()); got != 1 {
		t.Fatalf("network body has %d faces, want 1", got)
	}
}

func TestNetworkSurfaceBodyRejectsShortPolyline(t *testing.T) {
	t.Parallel()
	u, v := flatGrid()
	u[0] = u[0][:1] // a single point cannot be fitted
	if _, err := NetworkSurfaceBody(u, v); err == nil {
		t.Error("a degenerate U-polyline should error")
	}
}

func TestNetworkSurfaceBodyRejectsNonIntersecting(t *testing.T) {
	t.Parallel()
	u, _ := flatGrid()
	lifted := func(x, y float64) math.Point3 { return math.P3(math.Scalar(x), math.Scalar(y), 5) }
	v := [][]math.Point3{ // every V-curve lifted +5 in z: never meets the U-plane
		{lifted(0, 0), lifted(0, 1), lifted(0, 2)},
		{lifted(2, 0), lifted(2, 1), lifted(2, 2)},
	}
	if _, err := NetworkSurfaceBody(u, v); err == nil {
		t.Error("non-intersecting curves should error")
	}
}
