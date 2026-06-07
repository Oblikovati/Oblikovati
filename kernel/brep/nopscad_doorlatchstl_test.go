// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

func TestNopDoorLatchStlCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(3.5, 1.2, 0.3, 8), 0, 0.5, "door-latch-rounded-base")
	body = joinOrFatal(t, body, box(-1.75, -0.2, 0.25, 3.5, 0.4, 0.35), "door-latch-ridge")
	body = joinOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.6, 48, 0), 0, 1.425, "door-latch-boss"), "door-latch-boss")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.22, 32, 0), -0.05, 1.5, "door-latch-screw"), "door latch screw clearance")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.42, 6, stdmath.Pi/6), 0.7, 1.5, "door-latch-nut-trap"), "door latch nut trap")

	requireValidNopSolid(t, "door_latch_stl", body)
	if got := vol(body); got <= 3.5*1.2*0.5 {
		t.Errorf("door_latch_stl volume = %.6f, want larger than latch plate", got)
	}
}
