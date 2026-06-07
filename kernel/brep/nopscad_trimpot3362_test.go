// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati/math"
)

func TestNopTrimpot3362CSG(t *testing.T) {
	body := box(-0.3495, -0.33, 0.019, 0.699, 0.66, 0.45)
	for _, p := range []math.Point3{math.P3(-0.26, -0.22, -0.019), math.P3(0.26, -0.22, -0.019), math.P3(0, 0.22, -0.019)} {
		body = joinOrFatal(t, body, box(p.X-0.019, p.Y-0.019, p.Z, 0.038, 0.038, 0.038), "trimpot foot")
	}
	body = cutOrFatal(t, body, cylinderZAt(0, 0, 0.36, 0.52, 0.1385, "trimpot adjust recess"), "trimpot adjust recess")
	body = cutOrFatal(t, body, box(-0.16, -0.03, 0.39, 0.32, 0.06, 0.16), "trimpot screw slot")

	requireValidNopSolid(t, "trimpot3362", body)
	if got := vol(body); got <= 0.699*0.66*0.3 {
		t.Errorf("trimpot3362 volume = %.6f, want body plus feet", got)
	}
}
