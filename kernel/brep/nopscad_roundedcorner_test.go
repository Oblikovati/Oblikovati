// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopRoundedCornerCSG(t *testing.T) {
	body := prismBody(roundedCornerRectPoints(1.6, 2.4, 0.35, 20), 0, 0.2, "rounded-corner")
	requireValidNopSolid(t, "rounded_corner", body)
	if got := vol(body); got <= 0 || got >= 1.6*2.4*0.2 {
		t.Errorf("rounded_corner volume = %.6f, want below square blank", got)
	}
}
