// SPDX-License-Identifier: GPL-2.0-only

package material

import "testing"

func TestRecipeRoundTripEmbedsAssetsAndAssignments(t *testing.T) {
	set := NewAssetSet()
	assign := NewAssignmentStore()
	set.PutAppearance(NewAppearance("custom-red", SourceDocument, AppearanceSpec{
		DisplayName: "Custom Red", Albedo: mustColor("#ff0000ff"),
		Metallic: 0.2, Roughness: 0.5, Emissive: mustColor("#000000ff"), Opacity: 1,
	}))
	assign.SetPartAppearance("custom-red")
	assign.SetBodyMaterial("bodykey", "steel")

	data := MarshalRecipe(set, assign)

	// Restore into a fresh document.
	set2, assign2 := NewAssetSet(), NewAssignmentStore()
	if err := ApplyRecipe(data, set2, assign2); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	a, ok := set2.Appearance("custom-red")
	if !ok {
		t.Fatal("embedded appearance not restored")
	}
	if a.Source() != SourceDocument {
		t.Errorf("restored appearance source = %q, want document", a.Source())
	}
	if a.Albedo() != mustColor("#ff0000ff") || a.Metallic() != 0.2 {
		t.Errorf("restored appearance lost PBR fields: %+v", a.Spec())
	}
	if assign2.PartAppearance() != "custom-red" || assign2.BodyMaterials()["bodykey"] != "steel" {
		t.Errorf("assignments not restored: part=%q body=%v", assign2.PartAppearance(), assign2.BodyMaterials())
	}
}

// An un-styled document produces no materials section (empty asset set + no assignments).
func TestMarshalRecipeEmptyWhenUnstyled(t *testing.T) {
	data := MarshalRecipe(NewAssetSet(), NewAssignmentStore())
	if len(data.Appearances) != 0 || len(data.Materials) != 0 || data.Assignments != nil {
		t.Errorf("unstyled document produced a non-empty materials recipe: %+v", data)
	}
}

func TestRecipeRejectsBadColor(t *testing.T) {
	data := RecipeData{Appearances: []AppearanceRecipe{{ID: "x", Albedo: "not-a-color"}}}
	if err := ApplyRecipe(data, NewAssetSet(), NewAssignmentStore()); err == nil {
		t.Error("ApplyRecipe accepted a malformed color")
	}
}
