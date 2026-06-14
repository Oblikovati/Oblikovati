// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// TestAssemblyFeatureProgramRoundTrips is the #785 save/load acceptance: an assembly's machining
// program survives a save and reopen — the features are restored in order with their inputs, not
// dropped (the gap this closes). Uses a parametric hole and a fillet (no geometry needed to record
// them) plus a real placed component, so the reopen runs the full ResolveReferences path.
func TestAssemblyFeatureProgramRoundTrips(t *testing.T) {
	store, ws, asm, widget, asmDef := placedAssembly(t)
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())

	down, err := math.NewUnitVector3(0, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	hole, err := feature.NewAssemblyHoleFeature(math.P3(1, 1, 1), down, 0.5, 1)
	if err != nil {
		t.Fatalf("hole: %v", err)
	}
	asmDef.AddFeature(hole)
	asmDef.AddFeature(feature.NewAssemblyFilletFeature([][]byte{[]byte("e0")}, func() float64 { return 0.3 }))

	def := reopenAssembly(t, store, ws, asm)
	if def.Features().Count() != 2 {
		t.Fatalf("reopened feature count = %d, want 2 (the program was dropped)", def.Features().Count())
	}
	if def.Features().Item(0).Kind() != "assemblyHole" || def.Features().Item(1).Kind() != "assemblyFillet" {
		t.Errorf("reopened kinds = [%s %s], want [assemblyHole assemblyFillet]",
			def.Features().Item(0).Kind(), def.Features().Item(1).Kind())
	}

	// Value fidelity: the reopened program re-marshals with the hole's diameter and the fillet's radius.
	data, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	recipe := string(data)
	for _, want := range []string{"assemblyHole", "diameter: 0.5", "assemblyFillet", "radius: 0.3"} {
		if !strings.Contains(recipe, want) {
			t.Errorf("reopened recipe is missing %q:\n%s", want, recipe)
		}
	}
}
