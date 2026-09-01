// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchTextHandlers drives sketch.addText -> getText -> editText, exercising the
// justification/height metric helpers.
func TestSketchTextHandlers(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var added wire.AddEntityIDResult
	call(t, r, s, "sketch.addText", mustJSON(t, wire.AddTextArgs{
		SketchIndex: 0, Anchor: []float64{1, 1}, Text: "Hi", Height: "5 mm",
		Rotation: "0 deg", Justify: "center", VJustify: "middle",
	}), &added)
	if added.EntityID == 0 {
		t.Fatal("addText returned no entity id")
	}
	call(t, r, s, "sketch.getText", mustJSON(t, wire.GetTextArgs{SketchIndex: 0, EntityID: added.EntityID}), nil)
	edited := "Edited"
	call(t, r, s, "sketch.editText", mustJSON(t, wire.EditTextArgs{
		SketchIndex: 0, EntityID: added.EntityID, Text: &edited, Height: "6 mm", Justify: "left", VJustify: "top",
	}), nil)
}

// TestSketchOffsetFillAutoDim drives sketch.offset, addFillRegion, and autoDimension.
func TestSketchOffsetFillAutoDim(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var circ wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"3 cm"}`, &circ)
	_ = tryCall(t, r, s, "sketch.offset", mustJSON(t, wire.OffsetSketchArgs{
		SketchIndex: 0, Entity: circ.EntityID, Distance: "5 mm", ArcSegments: 8,
	}))
	// Seed inside the seeded 4×3 rectangle.
	_ = tryCall(t, r, s, "sketch.addFillRegion", mustJSON(t, wire.AddFillRegionArgs{
		SketchIndex: 0, Seed: []float64{2, 1.5}, Style: "solid",
	}))
	_ = tryCall(t, r, s, "sketch.autoDimension", `{"sketchIndex":0}`)
}

// TestSketchTextErrors covers the bad-sketch / bad-id error branches.
func TestSketchTextErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	if err := tryCall(t, r, s, "sketch.getText", `{"sketchIndex":0,"entityId":99999}`); err == nil {
		t.Error("getText on an unknown entity should error")
	}
	if err := tryCall(t, r, s, "sketch.addText", `{"sketchIndex":9,"text":"x","height":"5 mm"}`); err == nil {
		t.Error("addText on an out-of-range sketch should error")
	}
}
