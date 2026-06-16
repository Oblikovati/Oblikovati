// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import "testing"

// TestSheetMetalHemApply seeds a sheet-metal wall, hems a top edge, and confirms one merged
// solid results; then checks the error paths.
func TestSheetMetalHemApply(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	out, err := applyMap(t, s, "sheetMetalHem", map[string]any{"edge": edge, "length": "6 mm", "type": "open", "gap": "4 mm"})
	if err != nil {
		t.Fatalf("hem apply: %v", err)
	}
	expectMergedSolid(t, out, "hem")

	// Error paths.
	if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalHem", `{"length":"5 mm"}`); err == nil {
		t.Error("hem without an edge must error")
	}
	if _, err := apply(t, profiledPart(t), "sheetMetalHem", `{"edge":"x","length":"5 mm"}`); err == nil {
		t.Error("hem on a non-sheet-metal part must error")
	}
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{"edge": edge, "length": "6 mm", "type": "rolled"}); err == nil {
		t.Error("hem with an unknown type must error")
	}
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{"edge": edge, "length": "bad"}); err == nil {
		t.Error("hem with a bad length must error")
	}
}
