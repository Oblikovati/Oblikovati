// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopFlatFlexCSG(t *testing.T) {
	t.Parallel()
	body := box(-0.85, -0.27, 0, 1.7, 0.14, 0.12)
	body = cutOrFatal(t, body, box(-0.59, -0.32, -0.02, 1.18, 0.1, 0.16), "flat-flex slot")
	body = joinOrFatal(t, body, box(-0.8, -0.27, -0.25, 1.6, 0.4, 0.25), "flat-flex back")
	body = joinOrFatal(t, body, box(-0.6, 0.13, -0.25, 1.2, 0.16, 0.12), "flat-flex middle")

	requireValidNopSolid(t, "flat_flex", body)
	if got := vol(body); got <= 0.1 {
		t.Errorf("flat_flex volume = %.6f, want assembled connector volume", got)
	}
}
