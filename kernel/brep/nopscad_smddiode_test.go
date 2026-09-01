// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopSmdDiodeCSG(t *testing.T) {
	t.Parallel()
	body := taperedBoxBody(0.46, 0.28, 0.42, 0.24, 0.02, 0.18, "smd-diode-body")
	for _, x := range []float64{-0.26, 0.26} {
		body = joinOrFatal(t, body, box(x-0.09, -0.12, 0.02, 0.18, 0.24, 0.04), "smd diode lead")
	}
	body = cutOrFatal(t, body, box(-0.11, -0.14, -0.01, 0.22, 0.28, 0.08), "smd diode lead gap")

	requireValidNopSolid(t, "smd_diode", body)
	if got := vol(body); got <= 0 {
		t.Errorf("smd_diode volume = %.6f, want positive package", got)
	}
}
