// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"oblikovati/math"
	"testing"
)

func TestNopRibbonGrommetCSG(t *testing.T) {
	outer := ribbonGrommetProfile(2.75, 0.42, 0.16, 16)
	inner := []math.Point3{math.P3(-0.95, 0.08, 0), math.P3(0.95, 0.08, 0), math.P3(0.95, 0.24, 0), math.P3(-0.95, 0.24, 0)}
	body := prismBody(outer, -0.15, 0.15, "ribbon-grommet-side")
	body = cutOrFatal(t, body, prismBody(inner, -0.2, 0.2, "ribbon-grommet-slot"), "ribbon slot")

	requireValidNopSolid(t, "ribbon_grommet", body)
	want := (nopPolygonArea(outer) - nopPolygonArea(inner)) * 0.3
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("ribbon_grommet volume = %.6f, want %.6f", got, want)
	}
}
