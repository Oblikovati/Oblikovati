// SPDX-License-Identifier: GPL-2.0-only

package nopscad

import (
	"math"
	"testing"
)

// TestWasherGoldenMetrics sanity-checks the STL loader + metric reduction
// against an M3 washer's known dimensions (OD 7, ID 3.1, thickness 0.5): the
// loaded volume must match the analytic annulus volume within tessellation
// error, proving the golden harness before part tests depend on it.
func TestWasherGoldenMetrics(t *testing.T) {
	m, err := Golden("washer")
	if err != nil {
		t.Fatalf("load washer golden: %v", err)
	}
	// NopSCADlib draws the washer as linear_extrude(thickness-0.05) (a
	// z-fighting fudge), so the rendered solid is 0.45 thick, not 0.5.
	const od, id, th = 7.0, 3.1, 0.45
	want := math.Pi / 4 * (od*od - id*id) * th
	if rel := math.Abs(m.Volume-want) / want; rel > 0.02 {
		t.Errorf("washer volume = %.4f mm^3, want ~%.4f (rel err %.3f)", m.Volume, want, rel)
	}
	if sz := m.Size(); math.Abs(float64(sz.X)-od) > 0.1 || math.Abs(float64(sz.Z)-th) > 0.02 {
		t.Errorf("washer bbox size = %v, want ~(%g,%g,%g)", sz, od, od, th)
	}
	if m.TriCount == 0 {
		t.Error("washer golden has no triangles")
	}
}

// TestBallGoldenMetrics checks the sphere golden (d=5) volume against 4/3 pi r^3.
func TestBallGoldenMetrics(t *testing.T) {
	m, err := Golden("bearing_ball")
	if err != nil {
		t.Fatalf("load bearing_ball golden: %v", err)
	}
	const d = 5.0
	want := math.Pi / 6 * d * d * d
	if rel := math.Abs(m.Volume-want) / want; rel > 0.02 {
		t.Errorf("ball volume = %.4f mm^3, want ~%.4f (rel err %.3f)", m.Volume, want, rel)
	}
}
