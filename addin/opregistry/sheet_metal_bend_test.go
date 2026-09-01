// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalBendApply seeds a sheet-metal wall, adds a bend line crossing it, folds it,
// and confirms one valid solid; then checks the error paths.
func TestSheetMetalBendApply(t *testing.T) {
	t.Parallel()
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	line := def.Sketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 3)) // across the 4×3 sheet at x=2

	out, err := apply(t, s, "sheetMetalBend", `{"sketchIndex":1,"lineIndex":0,"angle":"90 deg","radius":"2 mm"}`)
	if err != nil {
		t.Fatalf("bend apply: %v", err)
	}
	expectMergedSolid(t, out, "bend")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalBend", `{"sketchIndex":0}`); err == nil {
		t.Error("bend on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalBend", `{"sketchIndex":99}`); err == nil {
		t.Error("bend with an out-of-range sketch must error")
	}
	if _, err := apply(t, s, "sheetMetalBend", `{"sketchIndex":1,"angle":"bad"}`); err == nil {
		t.Error("bend with a bad angle must error")
	}
}
