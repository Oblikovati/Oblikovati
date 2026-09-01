// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/query"
)

// TestSculptDirectedFromBoxFaces closes the six outward-facing box planes (keep each −normal side,
// the inside) into the solid box they bound — volume 32 for a 4×4×2 box (#1881).
func TestSculptDirectedFromBoxFaces(t *testing.T) {
	t.Parallel()
	faces := boxFaces(4, 4, 2)
	keep := make([]bool, len(faces)) // false = keep the −normal (inside) side of each outward face
	solid, err := SculptDirected(faces, keep, "sc")
	if err != nil {
		t.Fatalf("SculptDirected: %v", err)
	}
	if !solid.IsSolid() {
		t.Fatal("directed sculpt should yield a solid")
	}
	if v := query.BodyGeometryProperties(solid, DefaultQuality()).Volume; stdmath.Abs(v-32) > 0.1 {
		t.Errorf("volume = %g, want 32 (the 4×4×2 box the planes bound)", v)
	}
}

// TestSculptDirectedErrors: fewer than two surfaces, or a mismatched direction count, errors.
func TestSculptDirectedErrors(t *testing.T) {
	t.Parallel()
	faces := boxFaces(2, 2, 2)
	if _, err := SculptDirected(faces[:1], []bool{false}, "sc"); err == nil {
		t.Error("a single bounding surface should error")
	}
	if _, err := SculptDirected(faces, []bool{false}, "sc"); err == nil {
		t.Error("a direction count that does not match the surfaces should error")
	}
}
