// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopChequerboardCSG(t *testing.T) {
	t.Parallel()
	body := chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 0)
	requireValidNopSolid(t, "chequerboard", body)
	if got, want := vol(body), 32*0.3*0.2*0.08; stdmath.Abs(got-want)/want > 1e-9 {
		t.Errorf("chequerboard volume = %.6f, want %.6f", got, want)
	}
}
