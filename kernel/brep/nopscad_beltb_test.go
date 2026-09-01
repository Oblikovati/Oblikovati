// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopBeltbCSG(t *testing.T) {
	t.Parallel()
	body := annularPrismRange(t, 0.8, 0.68, 0, 0.6, "beltb-pulley-arc")
	body = joinOrFatal(t, body, box(-0.8, -0.06, 0, 1.8, 0.12, 0.6), "beltb straight run")

	requireValidNopSolid(t, "beltb", body)
	if got := vol(body); got <= 0.12*0.6*1.8 {
		t.Errorf("beltb volume = %.6f, want straight belt plus pulley arc", got)
	}
}
