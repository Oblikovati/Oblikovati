// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "testing"

// Assignments are keyed by persistent reference keys, so Recompute (which regenerates the
// bodies) must leave the assignment store intact — otherwise a material would vanish every
// time the model recomputes.
func TestAssignmentsSurviveRecompute(t *testing.T) {
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
