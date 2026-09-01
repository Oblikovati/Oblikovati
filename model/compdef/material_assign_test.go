// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/material"
)

// Assignments are keyed by persistent reference keys, so Recompute (which regenerates the
// bodies) must leave the assignment store intact — otherwise a material would vanish every
// time the model recomputes.
func TestAssignmentsSurviveRecompute(t *testing.T) {
	t.Parallel()
	def := NewPartComponentDefinition()
	def.Assignments().SetPartMaterial("steel")
	def.Assignments().SetBodyMaterial("abcd", "aluminum-6061")

	def.Recompute()

	if got := def.Assignments().PartMaterial(); got != "steel" {
		t.Errorf("part material after Recompute = %q, want \"steel\"", got)
	}
	if got := def.Assignments().BodyMaterials()["abcd"]; got != "aluminum-6061" {
		t.Errorf("body material after Recompute = %q, want \"aluminum-6061\"", got)
	}
}

// The materials section must survive a full recipe round-trip (Marshal → Apply through
// YAML), restoring the document's embedded appearance and its assignment.
func TestMaterialsRecipeRoundTrip(t *testing.T) {
	t.Parallel()
	def := NewPartComponentDefinition()
	wantColor := material.Color3{R: 0.1, G: 0.25, B: 1}
	def.Assets().PutAppearance(material.NewAppearance("doc-blue", material.SourceDocument,
		material.AppearanceSpec{
			DisplayName: "Doc Blue",
			Base:        material.OpenPBRBase{Color: wantColor, Weight: 1},
			Geometry:    material.OpenPBRGeometry{Opacity: 1},
		}))
	def.Assignments().SetPartAppearance("doc-blue")

	yaml, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewPartComponentDefinition()
	if err := restored.ApplyRecipe(yaml); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	a, ok := restored.Assets().Appearance("doc-blue")
	if !ok || a.Base().Color != wantColor {
		t.Errorf("embedded appearance not restored from recipe: %v ok=%v", a, ok)
	}
	if restored.Assignments().PartAppearance() != "doc-blue" {
		t.Errorf("assignment not restored: %q", restored.Assignments().PartAppearance())
	}
}
