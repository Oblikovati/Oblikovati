// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopRibbonGrommetHoleCSG(t *testing.T) {
	t.Parallel()
	body := prismBody(ribbonGrommetProfile(2.72, 0.405, 0.15, 16), 0, 5.0, "ribbon-grommet-hole")
	requireValidNopSolid(t, "ribbon_grommet_hole", body)
	if got, want := vol(body), nopPolygonArea(ribbonGrommetProfile(2.72, 0.405, 0.15, 16))*5.0; stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("ribbon_grommet_hole volume = %.6f, want %.6f", got, want)
	}
}
