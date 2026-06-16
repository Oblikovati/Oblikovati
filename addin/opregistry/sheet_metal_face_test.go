// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// sheetMetalProfiledPart builds a sheet-metal part with one closed rectangle profile — the
// in-package fixture for exercising applySheetMetalFace's success path (a cross-package
// router test would not count toward this package's coverage).
func sheetMetalProfiledPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "panel.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	if _, err := def.EnableSheetMetal(); err != nil {
		t.Fatalf("enable sheet metal: %v", err)
	}
	addRect(def.Sketches().Add(sketch.XYPlane()), 4, 3)
	def.Recompute()
	return s
}

// TestSheetMetalFaceApply the operation thickens the profile into one wall on a sheet-metal
// part, and reports a clear error on a part without the sheet-metal environment.
func TestSheetMetalFaceApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	out, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0,"operation":"new","direction":"positive"}`)
	if err != nil {
		t.Fatalf("sheetMetalFace apply: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if res.Bodies != 1 || !res.Healthy {
		t.Errorf("base Face: bodies=%d healthy=%v, want 1 healthy", res.Bodies, res.Healthy)
	}

	// On an ordinary part the operation rejects cleanly.
	if _, err := apply(t, profiledPart(t), "sheetMetalFace", `{"sketchIndex":0}`); err == nil {
		t.Error("sheetMetalFace on a non-sheet-metal part must error")
	}
}

// TestSheetMetalFaceApplyBadArgs each malformed-args path on a sheet-metal part returns a
// clean error (no panic): an out-of-range sketch, an unknown operation, and bad JSON.
func TestSheetMetalFaceApplyBadArgs(t *testing.T) {
	for _, bad := range []string{
		`{"sketchIndex":99}`,                   // no such sketch
		`{"sketchIndex":0,"operation":"weld"}`, // unknown operation
		`{"sketchIndex":"x"}`,                  // malformed JSON type
	} {
		if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalFace", bad); err == nil {
			t.Errorf("sheetMetalFace(%s) should error", bad)
		}
	}
}
