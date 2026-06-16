// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import "testing"

// TestSheetMetalFlangeApply seeds a sheet-metal wall, flanges a top edge, and confirms one
// merged solid results; then checks the error paths (non-sheet-metal part, missing edge).
func TestSheetMetalFlangeApply(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	out, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "10 mm", "angle": "90 deg", "radius": "2 mm"})
	if err != nil {
		t.Fatalf("flange apply: %v", err)
	}
	expectMergedSolid(t, out, "flange")

	// Error paths.
	if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalFlange", `{"height":"5 mm"}`); err == nil {
		t.Error("flange without an edge must error")
	}
	if _, err := apply(t, profiledPart(t), "sheetMetalFlange", `{"edge":"x","height":"5 mm"}`); err == nil {
		t.Error("flange on a non-sheet-metal part must error")
	}
	if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "bad"}); err == nil {
		t.Error("flange with a bad height must error")
	}
}
