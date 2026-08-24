// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"testing"

	"oblikovati.org/api/types"
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
