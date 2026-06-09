// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestNopOpengrabTargetCSG pins the target plate as a square slab with six circular
// through-cuts, matching opengrab_target before MCP sketch/profile integration.
func TestNopOpengrabTargetCSG(t *testing.T) {
	body := box(-2, -2, 0, 4, 4, 0.1)
	removed := 0.0
	for _, hole := range []struct {
		center math.Point3
		radius float64
	}{
		{math.P3(-1.69, -1.69, 0), 0.16}, {math.P3(-1.69, 1.69, 0), 0.16},
		{math.P3(1.69, -1.69, 0), 0.16}, {math.P3(1.69, 1.69, 0), 0.16},
		{math.P3(-1.65, 0, 0), 0.20}, {math.P3(1.65, 0, 0), 0.20},
	} {
		tool := prismBody(regularPolygonPoints(hole.center, hole.radius, 32, 0), -0.05, 0.15, "opengrab-hole")
		var err error
		body, err = ops.Boolean(ops.Cut, body, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut hole r=%g at %+v): %v", hole.radius, hole.center, err)
		}
		removed += nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), hole.radius, 32, 0)) * 0.1
	}

	requireValidNopSolid(t, "opengrab_target", body)
	want := 4.0*4.0*0.1 - removed
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("opengrab_target volume = %.6f, want ~%.6f", got, want)
	}
}
