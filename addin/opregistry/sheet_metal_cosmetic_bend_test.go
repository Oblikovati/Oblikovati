// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalCosmeticBendApply seeds a sheet-metal wall, adds a bend line crossing it, and
// marks it cosmetic — the result is the unchanged base solid; then checks the error paths.
func TestSheetMetalCosmeticBendApply(t *testing.T) {
	t.Parallel()
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	line := def.Sketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 3)) // across the 4×3 sheet at x=2

	out, err := apply(t, s, "sheetMetalCosmeticBend", `{"sketchIndex":1,"lineIndex":0,"angle":"90 deg","radius":"2 mm"}`)
	if err != nil {
		t.Fatalf("cosmetic bend apply: %v", err)
	}
	expectMergedSolid(t, out, "cosmetic bend") // the body is unchanged but still one healthy solid

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalCosmeticBend", `{"sketchIndex":0}`); err == nil {
		t.Error("cosmetic bend on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalCosmeticBend", `{"sketchIndex":99}`); err == nil {
		t.Error("cosmetic bend with an out-of-range sketch must error")
	}
	if _, err := apply(t, s, "sheetMetalCosmeticBend", `{"sketchIndex":1,"angle":"bad"}`); err == nil {
		t.Error("cosmetic bend with a bad angle must error")
	}
}
