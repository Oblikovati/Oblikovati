// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalFoldApply seeds a sheet-metal wall, adds a fold line crossing it, folds it at
// the end-of-bend location, and confirms one valid solid; then checks the error paths.
func TestSheetMetalFoldApply(t *testing.T) {
	t.Parallel()
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	line := def.Sketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 3))

	out, err := apply(t, s, "sheetMetalFold", `{"sketchIndex":1,"lineIndex":0,"angle":"90 deg","radius":"2 mm","location":"end"}`)
	if err != nil {
		t.Fatalf("fold apply: %v", err)
	}
	expectMergedSolid(t, out, "fold")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalFold", `{"sketchIndex":0}`); err == nil {
		t.Error("fold on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalFold", `{"sketchIndex":1,"location":"middle"}`); err == nil {
		t.Error("fold with an unknown location must error")
	}
	if _, err := apply(t, s, "sheetMetalFold", `{"sketchIndex":1,"radius":"bad"}`); err == nil {
		t.Error("fold with a bad radius must error")
	}
}
