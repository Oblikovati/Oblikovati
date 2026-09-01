// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopTearslot2DCSG(t *testing.T) {
	t.Parallel()
	body := tearSlotBody(t, 0.35, 0.8, 0.08, false, "tearslot-2d")
	requireValidNopSolid(t, "tearslot_2d", body)
	if got := vol(body); got <= 0 {
		t.Errorf("tearslot_2d volume = %.6f, want positive thin solid", got)
	}
}
