// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/math"
)

func TestNopExtrusionCenterSectionCSG(t *testing.T) {
	t.Parallel()
	body := box(-0.1, -1.0, 0, 0.2, 2.0, 0.12)
	body = joinOrFatal(t, body, box(-1.0, -0.1, 0, 2.0, 0.2, 0.12), "extrusion center cross spar")
	for _, side := range []float64{-1, 1} {
		body = joinOrFatal(t, body, box(side*0.72-0.09, -0.55, 0, 0.18, 1.1, 0.12), "extrusion side tab")
		body = joinOrFatal(t, body, box(-0.55, side*0.72-0.09, 0, 1.1, 0.18, 0.12), "extrusion end tab")
	}
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.22, 32, 0), -0.05, 0.2, "extrusion center hole"), "extrusion center hole")

	requireValidNopSolid(t, "extrusion_center_section", body)
	if got := vol(body); got <= 0 {
		t.Errorf("extrusion_center_section volume = %.6f, want positive spar section", got)
	}
}
