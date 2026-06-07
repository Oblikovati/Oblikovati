// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopRdElectrolyticCSG(t *testing.T) {
	body := frustumBody(0.48, 0.44, 0, 1.15, 48, "rd-electrolytic-can")
	body = joinOrFatal(t, body, annularPrismRange(t, 0.5, 0.16, 0.02, 1.1, "rd-electrolytic-jacket"), "rd electrolytic jacket")
	for _, x := range []float64{-0.125, 0.125} {
		body = joinOrFatal(t, body, cylinderZAt(x, 0, -0.3, 0.02, 0.025, "rd-electrolytic-lead"), "rd electrolytic lead")
	}
	body = joinOrFatal(t, body, box(0.18, -0.02, 1.08, 0.22, 0.04, 0.04), "rd electrolytic crimp")

	requireValidNopSolid(t, "rd_electrolytic", body)
	if got := vol(body); got <= stdmath.Pi*0.44*0.44*1.0 {
		t.Errorf("rd_electrolytic volume = %.6f, want can plus leads", got)
	}
}
