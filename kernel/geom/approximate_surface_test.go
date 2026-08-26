// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// paramGrid lays out an n×n parameter grid over [0,1]² and the matching points from height(u,v).
func paramGrid(n int, height func(u, v float64) float64) (pts []math.Point3, us, vs []float64) {
	for i := range n {
		for j := range n {
			u := float64(i) / float64(n-1)
			v := float64(j) / float64(n-1)
			us = append(us, u)
			vs = append(vs, v)
			pts = append(pts, math.P3(math.Scalar(u), math.Scalar(v), math.Scalar(height(u, v))))
		}
	}
	return pts, us, vs
}

// TestApproximateSurfaceLSFitsPlane: a planar point grid is reproduced (within round-off) by the
// least-squares fit — the simplest exactness check.
func TestApproximateSurfaceLSFitsPlane(t *testing.T) {
	pts, us, vs := paramGrid(8, func(u, v float64) float64 { return 2*u - 3*v + 1 })
	s, err := ApproximateSurfaceLS(pts, us, vs, 3, 3, 5, 5)
	if err != nil {
		t.Fatalf("ApproximateSurfaceLS: %v", err)
	}
	for k := range pts {
		got := s.PointAt(us[k], vs[k])
		if d := float64(got.DistanceTo(pts[k])); d > 1e-6 {
			t.Fatalf("planar fit off by %.3g at (%g,%g)", d, us[k], vs[k])
		}
	}
}

// TestApproximateSurfaceLSFitsQuadratic: a smooth bump is approximated within a tight tolerance by a
// bicubic fit.
func TestApproximateSurfaceLSFitsQuadratic(t *testing.T) {
	pts, us, vs := paramGrid(10, func(u, v float64) float64 { return (u-0.5)*(u-0.5) + (v-0.5)*(v-0.5) })
	s, err := ApproximateSurfaceLS(pts, us, vs, 3, 3, 6, 6)
	if err != nil {
		t.Fatalf("ApproximateSurfaceLS: %v", err)
	}
	worst := 0.0
	for k := range pts {
		if d := float64(pts[k].DistanceTo(s.PointAt(us[k], vs[k]))); d > worst {
			worst = d
		}
	}
	if worst > 1e-3 {
		t.Errorf("quadratic fit max deviation %.4g exceeds 1e-3", worst)
	}
}

func TestApproximateSurfaceLSValidates(t *testing.T) {
	pts, us, vs := paramGrid(6, func(u, v float64) float64 { return 0 })
	cases := []struct {
		name                   string
		du, dv, nu, nv, ptsLen int
	}{
		{"degree<1", 0, 3, 5, 5, len(pts)},
		{"too few ctrl u", 3, 3, 3, 5, len(pts)},
		{"more ctrl than pts", 3, 3, 6, 6, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ApproximateSurfaceLS(pts[:c.ptsLen], us[:c.ptsLen], vs[:c.ptsLen], c.du, c.dv, c.nu, c.nv); err == nil {
				t.Errorf("%s should error", c.name)
			}
		})
	}
}

func TestApproximateSurfaceLSRejectsMismatchedParams(t *testing.T) {
	pts, us, vs := paramGrid(6, func(u, v float64) float64 { return 0 })
	if _, err := ApproximateSurfaceLS(pts, us[:5], vs, 3, 3, 5, 5); err == nil {
		t.Error("mismatched u-param length should error")
	}
}

func TestUniformClampedKnotsClampedAndEven(t *testing.T) {
	k := uniformClampedKnots(6, 3) // 6 ctrl, degree 3 → length 10, 2 interior at 1/3,2/3
	if len(k) != 10 {
		t.Fatalf("knot length = %d, want 10", len(k))
	}
	for i := range 4 {
		if k[i] != 0 || k[6+i] != 1 {
			t.Fatalf("clamp broken: %v", k)
		}
	}
	if d := k[4] - 1.0/3.0; d > 1e-12 || d < -1e-12 {
		t.Errorf("interior knot 0 = %g, want 1/3", k[4])
	}
}
