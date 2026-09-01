// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopTearslotCSG(t *testing.T) {
	t.Parallel()
	body := tearSlotBody(t, 0.35, 0.8, 0.5, false, "tearslot")
	requireValidNopSolid(t, "tearslot", body)
	if got := vol(body); got <= 0 || got >= (0.8+0.7)*1.0*0.5 {
		t.Errorf("tearslot volume = %.6f, outside expected hulled teardrop range", got)
	}
}
