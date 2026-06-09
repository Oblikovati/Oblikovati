// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/math"
)

func TestNopDimensionCSG(t *testing.T) {
	body := prismBody([]math.Point3{
		math.P3(-0.95, 0, 0),
		math.P3(-0.7, -0.08, 0),
		math.P3(-0.7, -0.015, 0),
		math.P3(0.7, -0.015, 0),
		math.P3(0.7, -0.08, 0),
		math.P3(0.95, 0, 0),
		math.P3(0.7, 0.08, 0),
		math.P3(0.7, 0.015, 0),
		math.P3(-0.7, 0.015, 0),
		math.P3(-0.7, 0.08, 0),
	}, -0.018, 0.018, "dimension")

	requireValidNopSolid(t, "dimension", body)
	if got := vol(body); got <= 0 {
		t.Errorf("dimension volume = %.6f, want positive line plus arrowheads", got)
	}
}
