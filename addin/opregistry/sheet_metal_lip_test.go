// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// TestSheetMetalLipApply seeds a sheet-metal wall and folds a stiffening lip onto a top edge,
// confirming one healthy solid; then checks the error paths.
func TestSheetMetalLipApply(t *testing.T) {
	t.Parallel()
	s, _ := seedSheetMetalSheet(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	edge := topEdgeKey(t, def)

	out, err := applyMap(t, s, "sheetMetalLip", map[string]any{"edge": edge, "height": "8 mm", "returnLength": "3 mm"})
	if err != nil {
		t.Fatalf("lip apply: %v", err)
	}
	expectMergedSolid(t, out, "lip")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalLip", `{"edge":"x","height":"8 mm"}`); err == nil {
		t.Error("lip on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalLip", `{"height":"8 mm"}`); err == nil {
		t.Error("lip without an edge must error")
	}
	if _, err := applyMap(t, s, "sheetMetalLip", map[string]any{"edge": edge, "height": "bad"}); err == nil {
		t.Error("lip with a bad height must error")
	}
}
