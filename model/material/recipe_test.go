// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/yamlcodec"
)

// TestRecipeRoundTripEmbedsAssetsAndAssignments: a document-embedded custom appearance
// and its part/body/face assignments (plus a body material assignment) must all survive
// a marshal/apply round trip.
func TestRecipeRoundTripEmbedsAssetsAndAssignments(t *testing.T) {
	set := NewAssetSet()
	assign := NewAssignmentStore()
	spec := AppearanceSpec{DisplayName: "Custom Copper"}
	spec.Base.Color = types.NewColor3(0.72, 0.45, 0.2)
	spec.Base.Weight = 1
	set.PutAppearance(NewAppearance("custom-copper", SourceDocument, spec))
	assign.SetPartAppearance("custom-copper")
	assign.SetBodyAppearance("bodykey", "custom-copper")
	assign.SetFaceAppearance("facekey", "custom-copper")
	assign.SetBodyMaterial("bodykey", "steel")

	data := MarshalRecipe(set, assign)

	// Restore into a fresh document.
	set2, assign2 := NewAssetSet(), NewAssignmentStore()
	if err := ApplyRecipe(data, set2, assign2); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	a, ok := set2.Appearance("custom-copper")
	if !ok {
		t.Fatal("embedded appearance not restored")
	}
	if a.Source() != SourceDocument {
		t.Errorf("restored appearance source = %q, want document", a.Source())
	}
	if a.Base().Color != spec.Base.Color || a.Base().Weight != 1 {
		t.Errorf("restored appearance lost base fields: %+v", a.Base())
	}
	if assign2.PartAppearance() != "custom-copper" ||
		assign2.BodyAppearances()["bodykey"] != "custom-copper" ||
		assign2.FaceAppearances()["facekey"] != "custom-copper" ||
		assign2.BodyMaterials()["bodykey"] != "steel" {
		t.Errorf("assignments not restored: part=%q body=%v face=%v bodyMaterial=%v",
			assign2.PartAppearance(), assign2.BodyAppearances(), assign2.FaceAppearances(), assign2.BodyMaterials())
	}
}

// An un-styled document produces no materials section (empty asset set + no assignments).
func TestMarshalRecipeEmptyWhenUnstyled(t *testing.T) {
	data := MarshalRecipe(NewAssetSet(), NewAssignmentStore())
	if len(data.Appearances) != 0 || len(data.Materials) != 0 || data.Assignments != nil {
		t.Errorf("unstyled document produced a non-empty materials recipe: %+v", data)
	}
}

// TestApplyRecipeMigratesLegacyShapedAppearance is M46-F04's regression: a pre-
// consolidation document embedding an appearance in the OLD 5-scalar shape (a
// top-level "albedo" key) must load correctly, migrated through legacyAppearanceToSpec
// — not silently produce an all-zero-value appearance (the unrecognized albedo/
// metallic/... keys would otherwise just be ignored by a plain typed unmarshal).
func TestApplyRecipeMigratesLegacyShapedAppearance(t *testing.T) {
	raw := []byte(`
appearances:
  - {id: my-custom-red, name: My Custom Red, albedo: "#c02020ff", metallic: 0.0, roughness: 0.4, opacity: 1.0}
assignments:
  partAppearance: my-custom-red
`)
	var data RecipeData
	if err := yamlcodec.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	set, assign := NewAssetSet(), NewAssignmentStore()
	if err := ApplyRecipe(data, set, assign); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	a, ok := set.Appearance("my-custom-red")
	if !ok {
		t.Fatal("legacy-shaped appearance was not migrated into the asset set")
	}
	if a.Base().Metalness != 0 || a.Base().Color == (Color3{}) {
		t.Errorf("migrated appearance base = %+v, want a non-zero color and metalness 0", a.Base())
	}
	if a.Specular().Roughness != 0.4 {
		t.Errorf("migrated appearance specular roughness = %v, want 0.4", a.Specular().Roughness)
	}
	if got := assign.PartAppearance(); got != "my-custom-red" {
		t.Errorf("PartAppearance() = %q, want %q (assignment ids don't change shape)", got, "my-custom-red")
	}
}
