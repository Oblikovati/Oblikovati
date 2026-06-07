// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopWireLinkCSG(t *testing.T) {
	body := cylinderZAt(-0.6, 0, -0.3, 0.9, 0.06, "wire-left-leg")
	body = joinOrFatal(t, body, cylinderZAt(0.6, 0, -0.3, 0.9, 0.06, "wire-right-leg"), "wire right leg")
	body = joinOrFatal(t, body, cylinderAlongX(0.06, -0.6, 0.6, 0.9, "wire-top-link"), "wire top link")

	requireValidNopSolid(t, "wire_link", body)
	if got := vol(body); got <= 2*stdmath.Pi*0.06*0.06*1.2 {
		t.Errorf("wire_link volume = %.6f, want legs plus top link", got)
	}
}
