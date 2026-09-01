// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopE3dFanDuctCSG(t *testing.T) {
	t.Parallel()
	body := e3dFanDuctBody(t)
	requireValidNopSolid(t, "e3d_fan_duct", body)
	if got := vol(body); got <= 0 {
		t.Errorf("e3d_fan_duct volume = %.6f, want positive duct", got)
	}
}
