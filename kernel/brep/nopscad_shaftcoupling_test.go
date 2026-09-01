// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopShaftCouplingCSG(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~14s): `make test-corpus`")
	}
	t.Parallel()
	body := annularPrismRange(t, 0.6, 0.2, -1.0, 0, "shaft-coupling-small")
	body = joinOrFatal(t, body, annularPrismRange(t, 0.6, 0.3, 0, 1.0, "shaft-coupling-large"), "coupling second bore")
	for _, z := range []float64{-0.5, 0.5} {
		body = cutOrFatal(t, body, cylinderAlongY(0.08, -0.8, 0.8, z, "shaft-grub-y"), "shaft grub y")
		body = cutOrFatal(t, body, cylinderAlongX(0.08, -0.8, 0.8, z, "shaft-grub-x"), "shaft grub x")
	}

	requireValidNopSolid(t, "shaft_coupling", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*0.6*0.6*2.0 {
		t.Errorf("shaft_coupling volume = %.6f, outside expected coupling range", got)
	}
}
