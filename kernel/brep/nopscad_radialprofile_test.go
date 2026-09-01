// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/math"
)

func TestNopRadialProfileCSG(t *testing.T) {
	t.Parallel()
	points := []math.Point3{math.P3(0.16, 0, 0), math.P3(0.5, 0, 0), math.P3(0.5, 1.0, 0), math.P3(0.42, 1.16, 0), math.P3(0.22, 1.16, 0), math.P3(0.16, 1.0, 0)}
	body := prismBody(points, -0.03, 0.03, "radial-profile")

	requireValidNopSolid(t, "profile", body)
	if got := vol(body); got <= 0 {
		t.Errorf("profile volume = %.6f, want positive radial half-profile", got)
	}
}
