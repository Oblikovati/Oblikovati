// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopVerticalTearslotCSG(t *testing.T) {
	t.Parallel()
	body := tearSlotBody(t, 0.35, 0.8, 0.5, true, "vertical-tearslot")
	requireValidNopSolid(t, "vertical_tearslot", body)
	if got := vol(body); got <= 0 || got >= 1.0*(0.8+0.7)*0.5 {
		t.Errorf("vertical_tearslot volume = %.6f, outside expected hulled teardrop range", got)
	}
}
