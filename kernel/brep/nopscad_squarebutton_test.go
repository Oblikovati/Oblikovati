// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati/math"
)

func TestNopSquareButtonCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(1.2, 1.2, 0.12, 8), 0, 0.35, "button-base")
	for _, x := range []float64{-0.4, 0.4} {
		for _, y := range []float64{-0.4, 0.4} {
			body = joinOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.08, 24, 0), 0, 0.4, "button-rivet"), "button rivet")
		}
	}
	body = joinOrFatal(t, body, frustumBody(0.22, 0.18, 0.35, 0.62, 32, "button-stem"), "button stem")
	body = joinOrFatal(t, body, frustumBody(0.3, 0.25, 0.62, 0.9, 32, "button-cap"), "button cap")

	requireValidNopSolid(t, "square_button", body)
	if got := vol(body); got <= 1.2*1.2*0.35 {
		t.Errorf("square_button volume = %.6f, want larger than base", got)
	}
}
