// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/app"
)

// TestDefaultDescriptors checks the default registry exposes the additive extrude plus the
// subtractive/dress-up family, and that each descriptor is complete with valid-JSON schema —
// so add_feature (and the MCP bridge) can drive every one of them.
func TestDefaultDescriptors(t *testing.T) {
	r := Default()
	all := []string{
		"extrude", "revolve", "rib", "emboss", "coil", "loft",
		"fillet", "ruleFillet", "snapFit", "chamfer", "shell", "draft", "lip", "hole", "boss", "grill", "thread",
		"combine", "thicken", "trim", "directEdit", "moveFace", "faceOffset", "deleteFace", "split",
		"replaceFace", "simplify", "unwrap", "modelTolerance", "moveBody", "bendPart", "sheetMetalFace", "sheetMetalFlange", "sheetMetalHem", "sheetMetalBend", "sheetMetalFold", "sheetMetalCorner", "sheetMetalContourFlange", "sheetMetalLoftedFlange", "sheetMetalContourRoll", "sheetMetalCornerSeam", "sheetMetalCut", "sheetMetalRip", "sheetMetalPunch", "sheetMetalLip", "sheetMetalCosmeticBend", "sheetMetalUnfold", "sheetMetalRefold", "splitSolid", "coreCavity", "hull",
		"sweep", "patternRectangular", "patternCircular", "mirror", "patternSketchDriven",
		"boundaryPatch", "ruledSurface", "surfaceOffset", "extend", "midSurface", "stitch", "sculpt",
		"freeformBox", "freeformPlane", "freeformQuadBall", "mesh",
	}
	if got := len(r.All()); got != len(all) {
		t.Errorf("default registry has %d operations, want %d", got, len(all))
	}
	for _, name := range all {
		d, ok := r.ByName(name)
		if !ok {
			t.Errorf("default registry missing %q", name)
			continue
		}
		if d.Summary == "" || len(d.Schema) == 0 || d.Apply == nil {
			t.Errorf("descriptor %q incomplete: %+v", name, d)
		}
		var schema map[string]any
		if err := json.Unmarshal(d.Schema, &schema); err != nil {
			t.Errorf("%q schema is not valid JSON: %v", name, err)
		}
	}
}

func TestRegisterValidates(t *testing.T) {
	r := New()
	if err := r.Register(&OperationDescriptor{Name: "", Apply: dummyApply}); err == nil {
		t.Error("expected error for empty name")
	}
	if err := r.Register(&OperationDescriptor{Name: "x"}); err == nil {
		t.Error("expected error for missing Apply")
	}
	if err := r.Register(&OperationDescriptor{Name: "x", Apply: dummyApply}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(&OperationDescriptor{Name: "x", Apply: dummyApply}); err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestAllInRegistrationOrder(t *testing.T) {
	r := New()
	_ = r.Register(&OperationDescriptor{Name: "a", Apply: dummyApply})
	_ = r.Register(&OperationDescriptor{Name: "b", Apply: dummyApply})
	all := r.All()
	if len(all) != 2 || all[0].Name != "a" || all[1].Name != "b" {
		t.Fatalf("All order = %v, want [a b]", names(all))
	}
}

func names(ds []*OperationDescriptor) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Name
	}
	return out
}

func dummyApply(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }
