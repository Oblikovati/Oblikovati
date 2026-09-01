// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func TestNopMotorFaceplateCSG(t *testing.T) {
	t.Parallel()
	body := prismBody(roundedRectPoints(2.8, 2.8, 0.5, 8), -0.25, 0, "motor-faceplate")
	for _, x := range []float64{-0.95, 0.95} {
		for _, y := range []float64{-0.95, 0.95} {
			body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.16, 24, 0), -0.3, 0.05, "faceplate-screw"), "faceplate screw")
		}
	}
	body = joinOrFatal(t, body, annularPrism(t, 0.55, 0.25, 0.45, "faceplate-boss"), "faceplate boss")

	requireValidNopSolid(t, "motor_faceplate", body)
	if got := vol(body); got <= 0 || got >= 2.8*2.8*0.25+stdmath.Pi*0.55*0.55*0.45 {
		t.Errorf("motor_faceplate volume = %.6f, outside expected cut plate range", got)
	}
}
