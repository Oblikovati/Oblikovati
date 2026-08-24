// SPDX-License-Identifier: GPL-2.0-only

package material

import "testing"

func TestEffectiveAppearancePrecedence(t *testing.T) {
	lib := NewLibrary()
	st := NewAssignmentStore()
	const body, face = "bodykey", "facekey"

	// Nothing assigned → neutral default.
	if got := st.EffectiveAppearance(lib, body, ""); got.ID() != DefaultAppearanceID {
		t.Errorf("unassigned body appearance = %q, want %q", got.ID(), DefaultAppearanceID)
	}

	// Part-default material → that material's appearance.
	st.SetPartMaterial("steel") // steel material → "steel" appearance
	if got := st.EffectiveAppearance(lib, body, ""); got.ID() != "steel" {
		t.Errorf("with part material, appearance = %q, want \"steel\"", got.ID())
	}

	// Body material override beats the part default.
	st.SetBodyMaterial(body, "aluminum-6061") // → "aluminum" appearance
	if got := st.EffectiveAppearance(lib, body, ""); got.ID() != "aluminum" {
		t.Errorf("with body material, appearance = %q, want \"aluminum\"", got.ID())
	}

	// Body appearance override beats the material's appearance.
	st.SetBodyAppearance(body, "oak")
	if got := st.EffectiveAppearance(lib, body, ""); got.ID() != "oak" {
		t.Errorf("with body appearance override, = %q, want \"oak\"", got.ID())
	}

	// Face override beats everything.
	st.SetFaceAppearance(face, "abs-black")
	if got := st.EffectiveAppearance(lib, body, face); got.ID() != "abs-black" {
		t.Errorf("with face override, = %q, want \"abs-black\"", got.ID())
	}
	// A different face (no override) still resolves to the body override.
	if got := st.EffectiveAppearance(lib, body, "otherface"); got.ID() != "oak" {
		t.Errorf("other face = %q, want \"oak\" (body override)", got.ID())
	}
}

// TestEffectiveOpenPBRAppearancePrecedence mirrors TestEffectiveAppearancePrecedence for
// the separate OpenPBRAppearance chain (M45-F05 PBI-350/#2150): face → body → part, and
// ok=false (not a default fallback) when nothing in this chain is assigned, so a caller
// can fall back to the legacy Appearance chain instead.
func TestEffectiveOpenPBRAppearancePrecedence(t *testing.T) {
	lib := NewLibrary()
	partAppr, err := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "part-red", SourceProject)
	if err != nil {
		t.Fatalf("duplicate part appearance: %v", err)
	}
	bodyAppr, err := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "body-green", SourceProject)
	if err != nil {
		t.Fatalf("duplicate body appearance: %v", err)
	}
	faceAppr, err := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "face-blue", SourceProject)
	if err != nil {
		t.Fatalf("duplicate face appearance: %v", err)
	}

	st := NewAssignmentStore()
	const body, face = "bodykey", "facekey"

	if _, ok := st.EffectiveOpenPBRAppearance(lib, body, ""); ok {
		t.Error("unassigned body resolved an OpenPBR appearance, want ok=false")
	}

	st.SetPartOpenPBRAppearance(partAppr.ID())
	if got, ok := st.EffectiveOpenPBRAppearance(lib, body, ""); !ok || got.ID() != partAppr.ID() {
		t.Errorf("with part assignment, = %v/%v, want %q/true", got, ok, partAppr.ID())
	}

	st.SetBodyOpenPBRAppearance(body, bodyAppr.ID())
	if got, ok := st.EffectiveOpenPBRAppearance(lib, body, ""); !ok || got.ID() != bodyAppr.ID() {
		t.Errorf("body override = %v/%v, want %q/true", got, ok, bodyAppr.ID())
	}

	st.SetFaceOpenPBRAppearance(face, faceAppr.ID())
	if got, ok := st.EffectiveOpenPBRAppearance(lib, body, face); !ok || got.ID() != faceAppr.ID() {
		t.Errorf("face override = %v/%v, want %q/true", got, ok, faceAppr.ID())
	}
	if got, ok := st.EffectiveOpenPBRAppearance(lib, body, "otherface"); !ok || got.ID() != bodyAppr.ID() {
		t.Errorf("other face = %v/%v, want %q/true (body override)", got, ok, bodyAppr.ID())
	}
}

func TestEffectiveMaterialOverride(t *testing.T) {
	lib := NewLibrary()
	st := NewAssignmentStore()
	st.SetPartMaterial("steel")
	st.SetBodyMaterial("b1", "aluminum-6061")

	if m, ok := st.EffectiveMaterial(lib, "b1"); !ok || m.ID() != "aluminum-6061" {
		t.Errorf("body b1 material = %v, want aluminum-6061", m)
	}
	if m, ok := st.EffectiveMaterial(lib, "b2"); !ok || m.ID() != "steel" {
		t.Errorf("body b2 material = %v, want steel (part default)", m)
	}
}

func TestEffectiveMaterialID(t *testing.T) {
	st := NewAssignmentStore()
	if got := st.EffectiveMaterialID("b1"); got != "" {
		t.Errorf("unassigned body material id = %q, want empty", got)
	}
	st.SetPartMaterial("steel")
	st.SetBodyMaterial("b1", "aluminum-6061")
	if got := st.EffectiveMaterialID("b1"); got != "aluminum-6061" {
		t.Errorf("b1 material id = %q, want aluminum-6061 (override)", got)
	}
	if got := st.EffectiveMaterialID("b2"); got != "steel" {
		t.Errorf("b2 material id = %q, want steel (part default)", got)
	}
}

func TestSetEmptyClearsAssignment(t *testing.T) {
	st := NewAssignmentStore()
	st.SetBodyMaterial("b1", "steel")
	st.SetBodyMaterial("b1", "") // clear
	if len(st.BodyMaterials()) != 0 {
		t.Errorf("empty id did not clear the assignment: %v", st.BodyMaterials())
	}
}

// TestPartAppearanceOverrideBeatsMaterialAppearance locks the #1103 precedence fix: an explicit
// PART-level appearance override wins over the assigned material's own appearance. Before the fix
// the material appearance (step 3) was checked before the part override (step 4), so assigning a
// material (e.g. from an add-in) made a later "set appearance" a silent no-op (rendered grey).
func TestPartAppearanceOverrideBeatsMaterialAppearance(t *testing.T) {
	lib := NewLibrary()
	st := NewAssignmentStore()
	st.SetPartMaterial("steel") // material default → "steel" appearance
	st.SetPartAppearance("oak") // explicit part-level override
	if got := st.EffectiveAppearance(lib, "body", ""); got.ID() != "oak" {
		t.Errorf("part appearance override = %q, want \"oak\" (override must beat the material appearance)", got.ID())
	}
	// Clearing the override falls back to the material's appearance.
	st.SetPartAppearance("")
	if got := st.EffectiveAppearance(lib, "body", ""); got.ID() != "steel" {
		t.Errorf("after clearing the override, appearance = %q, want the material's \"steel\"", got.ID())
	}
}
