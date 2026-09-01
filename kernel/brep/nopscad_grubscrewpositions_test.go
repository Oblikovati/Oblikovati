// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopGrubScrewPositionsCSG(t *testing.T) {
	t.Parallel()
	body := annularPrism(t, 0.6, 0.25, 2.0, "grub-coupling")
	for _, z := range []float64{0.5, 1.5} {
		body = cutOrFatal(t, body, cylinderAlongY(0.08, -0.8, 0.8, z, "grub-screw-y"), "grub screw y")
		body = cutOrFatal(t, body, cylinderAlongX(0.08, -0.8, 0.8, z, "grub-screw-x"), "grub screw x")
	}

	requireValidNopSolid(t, "grub_screw_positions", body)
	if got := vol(body); got >= stdmath.Pi*(0.6*0.6-0.25*0.25)*2.0 {
		t.Errorf("grub_screw_positions volume = %.6f, want below uncut coupling", got)
	}
}
