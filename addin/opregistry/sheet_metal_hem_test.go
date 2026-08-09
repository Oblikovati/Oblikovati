// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

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
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{"edge": edge, "length": "6 mm", "type": "curled"}); err == nil {
		t.Error("hem with an unknown type must error")
	}
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{"edge": edge, "length": "bad"}); err == nil {
		t.Error("hem with a bad length must error")
	}
}

// The four hem shapes over the wire (#1956). What matters here is that each spelling reaches the
// definition with the dimensions ITS type is driven by — a rolled hem that arrived carrying only a
// length would silently fall back to a fold.

// lastHemDef returns the definition of the part's most recently added hem.
func lastHemDef(t *testing.T, s *app.Session) *feature.SheetMetalHemDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if h, ok := fs.Item(i).Definition().(*feature.SheetMetalHemFeature); ok {
			return h.Definition()
		}
	}
	t.Fatal("no hem feature on the part")
	return nil
}

// TestHemCurlDimensionsReachTheDefinition: radius is a length and angle is an ANGLE, so they go
// through different parsers. With units on both the two agree, so the test also gives a BARE
// sweep — the case where the parser's unit kind is the only thing that says what the number
// means, and reading it as a length yields nothing usable.
func TestHemCurlDimensionsReachTheDefinition(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{
		"edge": edge, "type": "rolled", "radius": "3 mm", "angle": "270 deg",
	}); err != nil {
		t.Fatalf("rolled hem: %v", err)
	}
	def := lastHemDef(t, s)
	if def.Type != feature.RolledHem {
		t.Errorf("type reached the definition as %d, want the rolled hem", def.Type)
	}
	if r := def.Radius(); r < 0.2999 || r > 0.3001 {
		t.Errorf("radius resolved to %g cm, want 0.3 (3 mm)", r)
	}
	if a := def.Angle(); a < 4.712 || a > 4.713 { // 270° = 4.7124 rad
		t.Errorf("angle resolved to %g rad, want ≈4.712 (270 deg)", a)
	}
	bare, edge2 := seedSheetMetalSheet(t)
	if _, err := applyMap(t, bare, "sheetMetalHem", map[string]any{
		"edge": edge2, "type": "rolled", "radius": "3 mm", "angle": "270",
	}); err != nil {
		t.Fatalf("rolled hem with a bare sweep: %v", err)
	}
	// A bare number in an angle argument is DEGREES, the way the dialogs read one.
	if a := lastHemDef(t, bare).Angle(); a < 4.712 || a > 4.713 {
		t.Errorf("bare sweep resolved to %g rad, want ≈4.712 — 270 was not read as degrees", a)
	}
}

// TestHemTypeSpellingsReachTheDefinition: every Inventor spelling, plus the two this feature
// shipped with, which both mean a single hem.
func TestHemTypeSpellingsReachTheDefinition(t *testing.T) {
	for spelling, want := range map[string]feature.HemType{
		"single": feature.SingleHem, "double": feature.DoubleHem,
		"closed": feature.SingleHem, "open": feature.SingleHem,
	} {
		t.Run(spelling, func(t *testing.T) {
			s, edge := seedSheetMetalSheet(t)
			if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{
				"edge": edge, "type": spelling, "length": "6 mm",
			}); err != nil {
				t.Fatalf("hem %q: %v", spelling, err)
			}
			if got := lastHemDef(t, s).Type; got != want {
				t.Errorf("type %q reached the definition as %d, want %d", spelling, got, want)
			}
		})
	}
}

// TestHemLeavesUnusedDimensionsUnset: a dimension the caller did not give must stay nil so the
// type's own default applies — a zero gap is not the same as no gap, and a zero radius is not a
// hem at all.
func TestHemLeavesUnusedDimensionsUnset(t *testing.T) {
	s, edge := seedSheetMetalSheet(t)
	if _, err := applyMap(t, s, "sheetMetalHem", map[string]any{
		"edge": edge, "type": "single", "length": "6 mm",
	}); err != nil {
		t.Fatalf("single hem: %v", err)
	}
	def := lastHemDef(t, s)
	if def.Gap != nil || def.Radius != nil || def.Angle != nil {
		t.Error("a single hem with only a length carries gap/radius/angle it was never given")
	}
}
