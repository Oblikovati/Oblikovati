// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

func TestNopStarWasherCSG(t *testing.T) {
	t.Parallel()
	outerR, innerR, thickness := 0.9, 0.31, 0.12
	body := annularPrism(t, outerR, innerR, thickness, "star-washer")
	inner := (innerR + outerR) / 2
	spoke := outerR - innerR
	for a := 0.0; a < 360; a += 30 {
		tool := rotatedBox(spoke, 2*stdmath.Pi*inner/36, thickness+0.1, inner+spoke/2, a*stdmath.Pi/180, "star-slot")
		body = cutOrFatal(t, body, tool, "star slot")
	}

	requireValidNopSolid(t, "star_washer", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*(outerR*outerR-innerR*innerR)*thickness {
		t.Errorf("star_washer volume = %.6f, want below uncut annulus", got)
	}
}
