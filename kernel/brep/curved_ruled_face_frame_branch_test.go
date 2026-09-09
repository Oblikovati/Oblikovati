// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestParamOnSpanBranchFindsTheWholeTurn: a closed curve's span can end AT the domain's end, where the
// inverted parameter comes back as the domain's START — a whole period away (ADR-0061's whole-turn
// trap). Without the branch the piece of a rim touching the seam matches nothing.
func TestParamOnSpanBranchFindsTheWholeTurn(t *testing.T) {
	t.Parallel()
	circle := geom.Circle{Center: math.P3(0, 0, 0), Normal: math.V3(0, 0, 1).AsUnit(),
		RefDir: math.V3(1, 0, 0).AsUnit(), Radius: 2}
	got, ok := paramOnSpanBranch(circle, 0, 0.845, 1, 1e-9)
	if !ok || got != 1 {
		t.Errorf("paramOnSpanBranch(t=0, span [0.845, 1]) = (%v, %v), want (1, true)", got, ok)
	}
	if _, ok := paramOnSpanBranch(circle, 0.5, 0.845, 1, 1e-9); ok {
		t.Error("a parameter genuinely outside the span was folded into it")
	}
}
