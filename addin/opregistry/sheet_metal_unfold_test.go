// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import "testing"

// TestSheetMetalUnfoldRefoldApply seeds a flanged sheet, flattens it (unfold), and refolds it
// — each yields one merged solid — then checks the error paths.
func TestSheetMetalUnfoldRefoldApply(t *testing.T) {
	t.Parallel()
	s, edge := seedSheetMetalSheet(t)
	if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "10 mm", "radius": "2 mm"}); err != nil {
		t.Fatalf("flange apply: %v", err)
	}

	out, err := apply(t, s, "sheetMetalUnfold", `{}`)
	if err != nil {
		t.Fatalf("unfold apply: %v", err)
	}
	expectMergedSolid(t, out, "unfold")

	out, err = apply(t, s, "sheetMetalRefold", `{}`)
	if err != nil {
		t.Fatalf("refold apply: %v", err)
	}
	expectMergedSolid(t, out, "refold")

	// Error paths: a non-sheet-metal part, and a sheet-metal part with no bends.
	if _, err := apply(t, profiledPart(t), "sheetMetalUnfold", `{}`); err == nil {
		t.Error("unfold on a non-sheet-metal part must error")
	}
	if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalUnfold", `{}`); err == nil {
		t.Error("unfold with no bends must error")
	}
}
