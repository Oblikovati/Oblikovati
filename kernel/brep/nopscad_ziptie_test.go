// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopZiptieCSG(t *testing.T) {
	t.Parallel()
	outer := stadiumBandPoints(0, 0, 1.0, 0.45, 24)
	inner := stadiumBandPoints(0, 0, 0.82, 0.27, 24)
	body := prismBody(outer, -0.18, 0.18, "ziptie-outer")
	body = cutOrFatal(t, body, prismBody(inner, -0.22, 0.22, "ziptie-inner"), "ziptie inner offset")
	strapVolume := vol(body)
	body = joinOrFatal(t, body, box(0.65, -0.16, -0.3, 0.35, 0.32, 0.6), "ziptie latch")

	requireValidNopSolid(t, "ziptie", body)
	if got := vol(body); got <= strapVolume {
		t.Errorf("ziptie volume = %.6f, want larger than strap band %.6f", got, strapVolume)
	}
}
