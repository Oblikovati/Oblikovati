// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalRipApply seeds a sheet-metal wall, adds a rip line across the middle, and slits
// it — the result is one healthy solid with material removed; then checks the error paths.
func TestSheetMetalRipApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	rip := def.Sketches().Add(sketch.XYPlane())
	rip.Lines().AddByTwoPoints(math.P2(1, 1.5), math.P2(3, 1.5)) // partial line across the 4×3 sheet

	out, err := apply(t, s, "sheetMetalRip", `{"sketchIndex":1,"lineIndex":0,"gap":"0.5 mm"}`)
	if err != nil {
		t.Fatalf("rip apply: %v", err)
	}
	expectMergedSolid(t, out, "rip")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalRip", `{"sketchIndex":0}`); err == nil {
		t.Error("rip on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalRip", `{"sketchIndex":99}`); err == nil {
		t.Error("rip with an out-of-range sketch must error")
	}
	if _, err := apply(t, s, "sheetMetalRip", `{"sketchIndex":1,"gap":"bad"}`); err == nil {
		t.Error("rip with a bad gap must error")
	}
	if _, err := apply(t, s, "sheetMetalRip", `{"sketchIndex":1,"type":"zigzag"}`); err == nil {
		t.Error("rip with an unknown type must error")
	}
	if _, err := apply(t, s, "sheetMetalRip", `{"sketchIndex":1,"gapSide":"sideways"}`); err == nil {
		t.Error("rip with an unknown gapSide must error")
	}
	if _, err := apply(t, s, "sheetMetalRip", `{"type":"faceExtents"}`); err == nil {
		t.Error("a face-extents rip with no faceKey must error")
	}
}
