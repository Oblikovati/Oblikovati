// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

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

// Where the wall lands, over the wire (#1957). The two controls do not change the height number,
// so the tests read them back off the definition rather than trusting a request that succeeded.

// lastFlangeDef returns the definition of the part's most recently added flange.
func lastFlangeDef(t *testing.T, s *app.Session) *feature.SheetMetalFlangeDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if f, ok := fs.Item(i).Definition().(*feature.SheetMetalFlangeFeature); ok {
			return f.Definition()
		}
	}
	t.Fatal("no flange feature on the part")
	return nil
}

// TestFlangePlacementReachesTheDefinition: the position, its offset and the datum all arrive.
func TestFlangePlacementReachesTheDefinition(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{
		"edge": edge, "height": "10 mm",
		"bendPosition": "outerEdgeOffset", "positionOffset": "2 mm", "heightDatum": "outerOrtho",
	}); err != nil {
		t.Fatalf("flange with a placement: %v", err)
	}
	def := lastFlangeDef(t, s)
	if def.Position != feature.BendOuterEdgeOffset || def.HeightDatum != feature.HeightFromOuterFaceOrtho {
		t.Errorf("placement reached the definition as position %d datum %d, want the outer offset and ortho datum",
			def.Position, def.HeightDatum)
	}
	if got := def.PositionOffset(); got < 0.1999 || got > 0.2001 {
		t.Errorf("positionOffset resolved to %g cm, want 0.2 (2 mm)", got)
	}
}

// TestFlangePlacementDefaults: omitting both keeps what this feature has always built, so an
// existing caller's flange does not move.
func TestFlangePlacementDefaults(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "10 mm"}); err != nil {
		t.Fatalf("plain flange: %v", err)
	}
	def := lastFlangeDef(t, s)
	if def.Position != feature.BendAtAdjacentFace || def.HeightDatum != feature.HeightFromTangent || def.PositionOffset != nil {
		t.Errorf("a plain flange = position %d datum %d offset %v, want the adjacent face, tangent datum and no offset",
			def.Position, def.HeightDatum, def.PositionOffset != nil)
	}
}

// TestUnknownFlangePlacementIsRefused: a misspelled position or datum must not fall back to the
// default and quietly build the part at different dimensions.
func TestUnknownFlangePlacementIsRefused(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	for _, args := range []map[string]any{
		{"edge": edge, "height": "10 mm", "bendPosition": "tangentToSideFace"},
		{"edge": edge, "height": "10 mm", "heightDatum": "outside"},
	} {
		if _, err := applyMap(t, s, "sheetMetalFlange", args); err == nil {
			t.Errorf("flange with %v should be refused", args)
		}
	}
}
