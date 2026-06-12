// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"strconv"
	"strings"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
)

// TestBlockLifecycleOverWire: create-from-selection moves the seeded
// rectangle's lines into a definition, instances place/enumerate with their
// placement decomposed, and in-use deletion is refused (M06-F07, #622).
func TestBlockLifecycleOverWire(t *testing.T) {
	r, s := seededSession(t)
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	refs := make([]string, 0, 4)
	for _, e := range ents.Entities {
		if e.Kind == "line" {
			refs = append(refs, intToString(e.ID))
		}
	}
	if len(refs) != 4 {
		t.Fatalf("seeded lines = %d, want the 4 rectangle lines", len(refs))
	}

	var def wire.SketchBlockDefinitionInfo
	call(t, r, s, "sketch.blockDefinitions.create",
		`{"name":"plate","sourceSketchIndex":0,"entityRefs":[`+strings.Join(refs, ",")+`]}`, &def)
	if def.Name != "plate" || def.EntityCount != 4 || def.InstanceCount != 1 {
		t.Fatalf("definition = %+v, want plate with 4 entities and the replacing instance", def)
	}

	var placed wire.AddSketchBlockResult
	call(t, r, s, "sketch.addBlockInstance",
		`{"sketchIndex":0,"definition":"plate","position":[10,5],"rotationAngle":"90 deg","scale":2}`, &placed)
	if placed.EntityID == 0 {
		t.Fatal("addBlockInstance returned no id")
	}

	var list wire.ListBlockInstancesResult
	call(t, r, s, "sketch.blockInstances", `{"sketchIndex":0}`, &list)
	if len(list.Instances) != 2 {
		t.Fatalf("instances = %d, want the replacing + the placed one", len(list.Instances))
	}
	got := list.Instances[1]
	if got.Definition != "plate" || got.Position[0] != 10 || got.Position[1] != 5 {
		t.Errorf("placed instance = %+v, want plate at (10, 5)", got)
	}
	if stdmath.Abs(got.Rotation-stdmath.Pi/2) > 1e-9 || stdmath.Abs(got.Scale-2) > 1e-9 {
		t.Errorf("placement = rot %v scale %v, want π/2 / 2", got.Rotation, got.Scale)
	}

	if _, err := r.Handle(s, "sketch.blockDefinitions.delete", []byte(`{"name":"plate"}`)); err == nil {
		t.Error("deleting an in-use definition must be refused")
	}

	var defs wire.ListBlockDefinitionsResult
	call(t, r, s, "sketch.blockDefinitions.list", ``, &defs)
	if len(defs.Definitions) != 1 || defs.Definitions[0].InstanceCount != 2 {
		t.Errorf("definitions = %+v, want plate with 2 instances", defs.Definitions)
	}
}

// TestBlockRecipeRoundTripOverDocument: a part with a block survives the
// document save→load path (the M21 DoD round-trip requirement).
func TestBlockRoundTripThroughRecipe(t *testing.T) {
	r, s := seededSession(t)
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	call(t, r, s, "sketch.blockDefinitions.create",
		`{"name":"plate","sourceSketchIndex":0,"entityRefs":[`+intToString(ents.Entities[0].ID)+`]}`,
		&wire.SketchBlockDefinitionInfo{})

	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	recipe, err := part.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if err := part.RestoreRecipe(recipe); err != nil {
		t.Fatalf("RestoreRecipe: %v", err)
	}

	var defs wire.ListBlockDefinitionsResult
	call(t, r, s, "sketch.blockDefinitions.list", ``, &defs)
	if len(defs.Definitions) != 1 || defs.Definitions[0].Name != "plate" || defs.Definitions[0].InstanceCount != 1 {
		t.Errorf("definitions after round-trip = %+v, want plate with its instance", defs.Definitions)
	}
}

// intToString renders an entity id for inline JSON.
func intToString(v uint64) string { return strconv.FormatUint(v, 10) }
