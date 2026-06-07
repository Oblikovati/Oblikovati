// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopVerticalTearslot2DCSG(t *testing.T) {
	body := tearSlotBody(t, 0.35, 0.8, 0.08, true, "vertical-tearslot-2d")
	requireValidNopSolid(t, "vertical_tearslot_2d", body)
	if got := vol(body); got <= 0 {
		t.Errorf("vertical_tearslot_2d volume = %.6f, want positive thin solid", got)
	}
}
