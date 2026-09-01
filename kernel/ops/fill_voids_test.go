// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// TestFillInternalVoidsRemovesCavity gates the hole-patch: the 4³−2³=56 cavity body
// becomes the solid 4³=64 block once its void shell is dropped, and stays manifold.
func TestFillInternalVoidsRemovesCavity(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	if v := query.BodyGeometryProperties(body, DefaultQuality()).Volume; stdmath.Abs(v-56) > 0.1 {
		t.Fatalf("fixture volume = %g, want 56 (4³ minus 2³ cavity)", v)
	}
	filled := FillInternalVoids(body, DefaultQuality())
	if got := len(filled.Shells()); got != 1 {
		t.Fatalf("filled body has %d shells, want 1 (void removed)", got)
	}
	if v := query.BodyGeometryProperties(filled, DefaultQuality()).Volume; stdmath.Abs(v-64) > 0.1 {
		t.Errorf("filled volume = %g, want 64 (cavity filled to solid 4³)", v)
	}
	if r := Validate(filled); !r.Valid {
		t.Errorf("filled body invalid: %v", r.Issues)
	}
}

// TestFillInternalVoidsLeavesSolidUnchanged: a void-free body is returned as-is.
func TestFillInternalVoidsLeavesSolidUnchanged(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if got := FillInternalVoids(block, DefaultQuality()); got != block {
		t.Error("a void-free body should be returned unchanged (same pointer)")
	}
}
