// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
)

// TestUnionPointContactSplitsPinch pins the pinched-vertex resolution (Oblikovati#1693): two
// boxes sharing exactly ONE corner point union into a body whose contact vertex must be CUT
// APART into coincident duplicates (two point-touching shells, χ = 4) — not welded into a single
// vertex with two face fans, which every per-edge check passes while χ = 3 marks the solid
// topologically impossible. This is the minimized fan-blade-tip-on-rim failure.
func TestUnionPointContactSplitsPinch(t *testing.T) {
	t.Parallel()
	a := box(0, 0, 0, 2, 2, 2)
	b := box(2, 2, 2, 2, 2, 2) // shares only the corner (2,2,2) with a
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	r := ops.Validate(res)
	if !r.Valid {
		t.Errorf("point-contact union invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
	if r.EulerCharacteristic != 4 {
		t.Errorf("χ = %d, want 4 (two sphere-topology shells after the pinch split)", r.EulerCharacteristic)
	}
}

// TestUnionEdgeContactStaysManifold guards the neighbouring behavior the pinch split must not
// disturb: two boxes sharing a full EDGE resolve through the tangent-edge machinery
// (resolveEdgeUses) into a valid solid — and with the endpoints of the resolved coincident edges
// now split per fan, the result must still validate.
func TestUnionEdgeContactStaysManifold(t *testing.T) {
	t.Parallel()
	a := box(0, 0, 0, 2, 2, 2)
	b := box(2, 2, 0, 2, 2, 2) // shares the vertical edge x=2,y=2, z∈[0,2]
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	if r := ops.Validate(res); !r.Valid {
		t.Errorf("edge-contact union invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
}
