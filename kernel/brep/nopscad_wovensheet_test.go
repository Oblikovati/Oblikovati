// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopWovenSheetCSG(t *testing.T) {
	body := chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 0)
	body = joinOrFatal(t, body, chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 1), "woven inverse sheet")
	requireValidNopSolid(t, "woven_sheet", body)
	if got, want := vol(body), 64*0.3*0.2*0.08; stdmath.Abs(got-want)/want > 1e-9 {
		t.Errorf("woven_sheet volume = %.6f, want %.6f", got, want)
	}
}
