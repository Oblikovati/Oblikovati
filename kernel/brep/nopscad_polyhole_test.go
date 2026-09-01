// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestNopPolyholeCSG pins NopSCADlib's polyhole helper as an eight-sided through
// cylinder. It is the low-level faceted drill shape used by printable holes.
func TestNopPolyholeCSG(t *testing.T) {
	t.Parallel()
	const radius, height = 0.25, 1.2
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), radius, 8, stdmath.Pi/8), 0, height, "polyhole")
	requireValidNopSolid(t, "polyhole", body)
	want := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), radius, 8, stdmath.Pi/8)) * height
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("polyhole volume = %.6f, want %.6f", got, want)
	}
}
