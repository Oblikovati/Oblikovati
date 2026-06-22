// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
)

// TestAssemblyMixingConvertsAtPlacement is the ADR-0042 Phase 2 mixed-unit boundary (#1247): a
// component whose working unit differs from the assembly's is scaled into the assembly's unit at
// the placement transform, so its geometry lands at the right size. A same-unit placement is
// unchanged.
func TestAssemblyMixingConvertsAtPlacement(t *testing.T) {
	// Same block geometry, but the part's working unit is the millimetre (0.1 cm), while the
	// assembly works in centimetres. Placing it must shrink the stored coordinates by 0.1.
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	mm, err := part.Units().CenteredOnLength("mm")
	if err != nil {
		t.Fatal(err)
	}
	part.SetUnits(mm) // part working scale = 0.1 cm/unit

	asm := NewAssemblyComponentDefinition() // assembly working scale = 1 (cm)
	asm.Place("p", part, math.Identity4())

	placed := asm.PlacedBodies()
	if len(placed) != 1 {
		t.Fatalf("PlacedBodies = %d, want 1", len(placed))
	}
	// The placement transform carries the unit conversion 0.1, so the placed bounding box is the
	// part's box scaled by 0.1.
	box := placed[0].Body.RangeBox().Transform(placed[0].Transform)
	if d := float64(box.Diagonal().Length()); d < 0.9*0.1*twoCubeDiag || d > 1.1*0.1*twoCubeDiag {
		t.Errorf("placed diagonal = %v, want ~%v (0.1 × part)", d, 0.1*twoCubeDiag)
	}

	// Control: a same-unit (cm) part is placed unscaled.
	cmPart := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	asm2 := NewAssemblyComponentDefinition()
	asm2.Place("p", cmPart, math.Identity4())
	cb := asm2.PlacedBodies()[0]
	box2 := cb.Body.RangeBox().Transform(cb.Transform)
	if d := float64(box2.Diagonal().Length()); d < 0.9*twoCubeDiag || d > 1.1*twoCubeDiag {
		t.Errorf("same-unit placed diagonal = %v, want ~%v (unscaled)", d, twoCubeDiag)
	}
}

// twoCubeDiag is the body diagonal of a 2×2×2 block (√12).
var twoCubeDiag = float64(math.V3(2, 2, 2).Length())
