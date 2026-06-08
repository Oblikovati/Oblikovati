// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

// cProfileSurface extrudes a C-shaped quadratic profile (in y,z) along x: both arm ends sit at
// y=0 (z=0 and z=2) with the bulge at y=2, so a point left of the opening is near BOTH arms —
// projecting it lands on whichever arm the seed is near. The branch-selection ParamNear gives.
func cProfileSurface(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 2, 1), math.P3(0, 0, 2)},
		{math.P3(2, 0, 0), math.P3(2, 2, 1), math.P3(2, 0, 2)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}}
	s, err := NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func TestParamNearRoundTrip(t *testing.T) {
	s := cProfileSurface(t)
	// A point exactly on the surface, seeded from a nearby (u,v): ParamNear must converge back so
	// PointAt round-trips to it.
	for _, uv := range [][2]float64{{0.5, 0.3}, {0.25, 0.7}, {0.8, 0.15}} {
		want := s.PointAt(uv[0], uv[1])
		u, v := s.ParamNear(want, uv[0]+0.05, uv[1]+0.05)
		if got := s.PointAt(u, v); float64(got.DistanceTo(want)) > 1e-4 {
			t.Errorf("ParamNear round-trip at %v: PointAt=%v want %v (off %.5f)", uv, got, want, got.DistanceTo(want))
		}
	}
}

func TestParamNearSelectsBranchBySeed(t *testing.T) {
	s := cProfileSurface(t)
	// A point left of the C's opening, near both arm ends (v≈0 at z=0, v≈1 at z=2). The seed picks
	// the branch — independent grid-seeded ParamAt cannot.
	q := math.P3(1, -0.5, 1)
	_, vLo := s.ParamNear(q, 0.5, 0.05) // seeded near the z=0 arm
	_, vHi := s.ParamNear(q, 0.5, 0.95) // seeded near the z=2 arm
	if vLo > 0.25 {
		t.Errorf("seed near v=0 should stay on the low arm, got v=%.3f", vLo)
	}
	if vHi < 0.75 {
		t.Errorf("seed near v=1 should stay on the high arm, got v=%.3f", vHi)
	}
	if stdmath.Abs(vHi-vLo) < 0.5 {
		t.Errorf("seed should select distinct branches: vLo=%.3f vHi=%.3f", vLo, vHi)
	}
}
