// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopAdjustCSG(t *testing.T) {
	t.Parallel()
	body := cylinderZAt(0, 0, -0.089, 0.089, 0.1385, "adjust-dial")
	body = cutOrFatal(t, body, box(-0.16, -0.032, 0, 0.32, 0.064, 0.11), "adjust slot x")
	body = cutOrFatal(t, body, box(-0.032, -0.16, 0, 0.064, 0.32, 0.11), "adjust slot y")

	requireValidNopSolid(t, "adjust", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*0.1385*0.1385*0.178 {
		t.Errorf("adjust volume = %.6f, want slotted trimpot dial", got)
	}
}
