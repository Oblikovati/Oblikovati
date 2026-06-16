// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalCutApply seeds a sheet-metal wall, adds a smaller profile, and cuts it through
// all the material — confirming one healthy solid; then checks the error paths.
func TestSheetMetalCutApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	addRect(def.Sketches().Add(sketch.XYPlane()), 2, 2) // a 2×2 cut profile inside the 4×3 sheet

	out, err := apply(t, s, "sheetMetalCut", `{"sketchIndex":1,"profileIndex":0}`)
	if err != nil {
		t.Fatalf("cut apply: %v", err)
	}
	expectMergedSolid(t, out, "cut")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalCut", `{"sketchIndex":0}`); err == nil {
		t.Error("cut on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalCut", `{"sketchIndex":99}`); err == nil {
		t.Error("cut with an out-of-range sketch must error")
	}
	// across-bend is reserved for F04: the feature goes sick (a healthy:false result), not a
	// Go error, since the add succeeds and the recompute reports the unsupported option.
	out, err = apply(t, s, "sheetMetalCut", `{"sketchIndex":1,"acrossBend":true}`)
	if err != nil {
		t.Fatalf("across-bend cut apply: %v", err)
	}
	var res struct {
		Healthy bool `json:"healthy"`
	}
	if e := json.Unmarshal(out, &res); e == nil && res.Healthy {
		t.Error("across-bend cut should be unhealthy (reserved for F04)")
	}
}
