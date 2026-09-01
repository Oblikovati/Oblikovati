// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestNopVariacDialCSG pins variac_dial as a round dial with one shaft hole and
// three screw holes through the plate.
func TestNopVariacDialCSG(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 2.5, 64, 0), 0, 0.3, "variac-dial")
	for _, hole := range append([]math.Point3{math.P3(0, 0, 0)}, regularPolygonPoints(math.P3(0, 0, 0), 1.6, 3, -stdmath.Pi/2)...) {
		radius := 0.25
		if hole.X == 0 && hole.Y == 0 {
			radius = 0.55
		}
		tool := prismBody(regularPolygonPoints(hole, radius, 32, 0), -0.05, 0.35, "variac-hole")
		var err error
		body, err = ops.Boolean(ops.Cut, body, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut dial hole): %v", err)
		}
	}

	requireValidNopSolid(t, "variac_dial", body)
	dialArea := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 2.5, 64, 0))
	centerHole := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.55, 32, 0))
	screwHole := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 32, 0))
	want := (dialArea - centerHole - 3*screwHole) * 0.3
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("variac_dial volume = %.6f, want %.6f", got, want)
	}
}
