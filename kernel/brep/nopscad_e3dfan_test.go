// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopE3dFanCSG(t *testing.T) {
	body := e3dFanDuctBody(t)
	body = joinOrFatal(t, body, box(1.5, -1.5, 0.2, 1.0, 3.0, 0.3), "e3d fan frame")
	body = cutOrFatal(t, body, cylinderZAt(2.0, 0, 0.15, 0.55, 1.1, "e3d fan aperture"), "e3d fan aperture")

	requireValidNopSolid(t, "e3d_fan", body)
	if got := vol(body); got <= 0 {
		t.Errorf("e3d_fan volume = %.6f, want positive duct plus fan assembly", got)
	}
}
