// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// TestSheetMetalCornerSeamApply seeds a sheet-metal wall and cuts a gap seam at a corner edge,
// confirming one healthy solid; then checks the error paths.
func TestSheetMetalCornerSeamApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	edge := verticalCornerEdgeKey(t, def)

	out, err := applyMap(t, s, "sheetMetalCornerSeam", map[string]any{"edges": []string{edge}, "gap": "3 mm"})
	if err != nil {
		t.Fatalf("corner seam apply: %v", err)
	}
	expectMergedSolid(t, out, "cornerSeam")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalCornerSeam", `{"edges":["x"],"gap":"1 mm"}`); err == nil {
		t.Error("corner seam on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalCornerSeam", `{"edges":[],"gap":"1 mm"}`); err == nil {
		t.Error("corner seam with no edges must error")
	}
	if _, err := applyMap(t, s, "sheetMetalCornerSeam", map[string]any{"edges": []string{edge}, "gap": "1 mm", "type": "overlap"}); err == nil {
		t.Error("corner seam with an unsupported type must error")
	}
	if _, err := applyMap(t, s, "sheetMetalCornerSeam", map[string]any{"edges": []string{edge}, "gap": "bad"}); err == nil {
		t.Error("corner seam with a bad gap must error")
	}
}
