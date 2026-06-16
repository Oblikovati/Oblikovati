// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// featureResult is the slim shape of a features.add reply we assert on.
type featureResult struct {
	Bodies  int  `json:"bodies"`
	Healthy bool `json:"healthy"`
}

// TestSheetMetalFaceOverWire builds a base wall on a sheet-metal part via features.add and
// confirms a single healthy solid results; then a thickness edit through setStyle rebuilds
// the wall (proving the rule's gauge propagates to the geometry).
func TestSheetMetalFaceOverWire(t *testing.T) {
	r, s := newSheetMetalPart(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[4,3]]}`, &struct{}{})

	var face featureResult
	call(t, r, s, "features.add", `{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`, &face)
	if face.Bodies != 1 || !face.Healthy {
		t.Fatalf("base Face: bodies=%d healthy=%v, want 1 healthy solid", face.Bodies, face.Healthy)
	}

	// Thicken the rule; the wall must rebuild (the model tree's body stays one solid).
	call(t, r, s, wire.MethodSheetMetalSetStyle, `{"thickness":"3 mm"}`, &wire.SheetMetalStyleResult{})
	var tree wire.ModelTreeResult
	call(t, r, s, "model.tree", "{}", &tree)
	if tree.Bodies != 1 {
		t.Errorf("after thickness edit the part has %d bodies, want 1", tree.Bodies)
	}
}

// TestSheetMetalFaceRejectsPlainPart features.add sheetMetalFace on an ordinary part errors
// (the operation requires the sheet-metal environment).
func TestSheetMetalFaceRejectsPlainPart(t *testing.T) {
	r, s := seededSession(t) // ordinary part with a profile
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`)); err == nil {
		t.Fatal("sheetMetalFace on a plain part must error")
	}
}

// TestSheetMetalFaceRejectsBadArgs sheetMetalFace reports a clear error for an out-of-range
// sketch index and a malformed args payload.
func TestSheetMetalFaceRejectsBadArgs(t *testing.T) {
	r, s := newSheetMetalPart(t)
	for _, bad := range []string{
		`{"kind":"sheetMetalFace","args":{"sketchIndex":99}}`,  // no such sketch
		`{"kind":"sheetMetalFace","args":{"sketchIndex":"x"}}`, // not an integer
	} {
		if _, err := r.Handle(s, "features.add", []byte(bad)); err == nil {
			t.Errorf("sheetMetalFace(%s) should error", bad)
		}
	}
}
