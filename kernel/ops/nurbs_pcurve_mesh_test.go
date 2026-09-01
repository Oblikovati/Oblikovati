// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestMetricScaleReflectsAnisotropy: a patch 10 units long in u and 1 in v must report su≈10·sv, so
// the CDT scales u/v into a ≈isometric space (the fix that stopped the (u,v) Delaunay from twisting
// in 3D and folding — M25). su = mean |∂P/∂u|, sv = mean |∂P/∂v|.
func TestMetricScaleReflectsAnisotropy(t *testing.T) {
	t.Parallel()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(10, 0, 0), math.P3(10, 1, 0)},
	}
	w := [][]float64{{1, 1}, {1, 1}}
	s, err := geom.NewBSplineSurface(1, 1, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	su, sv := metricScale(s)
	if r := su / sv; r < 8 || r > 12 {
		t.Errorf("metricScale su/sv = %.2f, want ~10 (u spans 10×, v spans 1×)", r)
	}
}

func TestMetricScaleNeverZero(t *testing.T) {
	t.Parallel()
	su, sv := metricScale(cProfileSurfaceOps(t))
	if su <= 0 || sv <= 0 {
		t.Errorf("metricScale returned non-positive (%g, %g)", su, sv)
	}
}

func cProfileSurfaceOps(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 2, 1), math.P3(0, 0, 2)},
		{math.P3(2, 0, 0), math.P3(2, 2, 1), math.P3(2, 0, 2)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}
