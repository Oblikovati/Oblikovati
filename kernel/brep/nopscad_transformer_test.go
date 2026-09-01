// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/math"
)

func TestNopTransformerCSG(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~5s): `make test-corpus`")
	}
	t.Parallel()
	body := prismBody(roundedRectPoints(4.0, 3.0, 0.2, 8), 0, 0.2, "transformer-foot")
	for _, x := range []float64{-1.5, 1.5} {
		for _, y := range []float64{-1.0, 1.0} {
			body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.16, 24, 0), -0.05, 0.25, "transformer-hole"), "transformer hole")
		}
	}
	body = joinOrFatal(t, body, box(-1.6, -0.6, 0.2, 3.2, 1.2, 1.8), "transformer-laminations")
	body = joinOrFatal(t, body, box(-1.0, -1.1, 0.45, 2.0, 2.2, 1.1), "transformer-bobbin")

	requireValidNopSolid(t, "transformer", body)
	if got := vol(body); got <= 4.0*3.0*0.2 {
		t.Errorf("transformer volume = %.6f, want larger than mounting foot", got)
	}
}
