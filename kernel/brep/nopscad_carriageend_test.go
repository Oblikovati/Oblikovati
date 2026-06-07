// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopCarriageEndCSG(t *testing.T) {
	body := carriageEndBody(t, -0.9, "carriage-left")
	body = joinOrFatal(t, body, carriageEndBody(t, 0.9, "carriage-right"), "carriage second end")

	requireValidNopSolid(t, "carriage_end", body)
	if got := vol(body); got <= 0 || got >= 2*0.7*0.5*0.8 {
		t.Errorf("carriage_end volume = %.6f, want cut end blocks", got)
	}
}
