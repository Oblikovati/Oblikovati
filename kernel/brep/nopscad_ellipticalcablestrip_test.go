// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

// TestNopEllipticalCableStripCSG pins the elliptical cable strip as a semi-elliptic
// frame extruded to the ribbon width.
func TestNopEllipticalCableStripCSG(t *testing.T) {
	t.Parallel()
	outer := semiEllipseFramePoints(1.5, 2.4, 0.08, 32)
	body := prismBody(outer, 0, 1.0, "elliptical-cable-strip")
	requireValidNopSolid(t, "elliptical_cable_strip", body)
	want := nopPolygonArea(outer)
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("elliptical_cable_strip volume = %.6f, want %.6f", got, want)
	}
}
