// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

// TestNopSemiTeardropCSG pins the kernel unit shape for NopSCADlib's semi_teardrop:
// a positive-Y semicircular profile extruded to height. The bridge integration test
// later proves the same shape through sketch constraints and parametric recompute.
func TestNopSemiTeardropCSG(t *testing.T) {
	const radius, height = 0.4, 2.0
	points := []math.Point3{math.P3(radius, 0, 0)}
	for i := 1; i < 32; i++ {
		angle := stdmath.Pi * float64(i) / 32
		points = append(points, math.P3(radius*stdmath.Cos(angle), radius*stdmath.Sin(angle), 0))
	}
	points = append(points, math.P3(-radius, 0, 0))

	body := prismBody(points, 0, height, "semi-teardrop")
	requireValidNopSolid(t, "semi_teardrop", body)
	want := stdmath.Pi * radius * radius * height / 2
	if got := vol(body); stdmath.Abs(got-want)/want > 0.02 {
		t.Errorf("semi_teardrop volume = %.6f, want ~%.6f", got, want)
	}
}
