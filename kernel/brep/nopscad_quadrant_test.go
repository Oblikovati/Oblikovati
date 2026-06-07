// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopQuadrantCSG(t *testing.T) {
	body := prismBody(roundedCornerRectPoints(2.0, 1.4, 0.5, 20), 0, 0.2, "quadrant")
	requireValidNopSolid(t, "quadrant", body)
	if got := vol(body); got <= 0 || got >= 2.0*1.4*0.2 {
		t.Errorf("quadrant volume = %.6f, want below square blank", got)
	}
}
