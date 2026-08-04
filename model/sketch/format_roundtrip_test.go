// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Format overrides and centre-point markers must survive a save/load, or the Format panel's work
// is lost the moment the document is closed.
func TestFormatAndCenterPointRoundTrip(t *testing.T) {
	sketches := NewSketches()
	src := sketches.Add(XYPlane())
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	src.SetEntityFormat(l.EntityID(), EntityFormat{
		LineType: "dashed", Color: types.NewColor(255, 0, 0), LineWeight: 0.5,
	})
	cp := src.Points().Add(math.P2(3, 4))
	cp.SetCenterPoint(true)
	plain := src.Points().Add(math.P2(6, 7))
	_ = plain

	data, err := sketches.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	loaded := NewSketches()
	if err := loaded.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	out := loaded.Item(0)

	if n := out.EntityFormatCount(); n != 1 {
		t.Fatalf("restored format entries = %d, want 1", n)
	}
	var restored EntityFormat
	for i := 0; i < out.Lines().Count(); i++ {
		if f, ok := out.EntityFormat(out.Lines().Item(i).EntityID()); ok {
			restored = f
		}
	}
	if restored.LineType != "dashed" || restored.LineWeight != 0.5 {
		t.Errorf("restored format = %+v, want the dashed 0.5 override", restored)
	}
	if !restored.Color.IsOverride() || restored.Color.R != 255 || restored.Color.G != 0 || restored.Color.B != 0 {
		t.Errorf("restored colour = %+v, want opaque red as an override", restored.Color)
	}

	centres := 0
	for i := 0; i < out.Points().Count(); i++ {
		if out.Points().Item(i).IsCenterPoint() {
			centres++
		}
	}
	if centres != 1 {
		t.Errorf("restored centre points = %d, want exactly the one that was marked", centres)
	}
}

// An unstyled sketch must round-trip with no format entries — absence stays absence.
func TestUnstyledSketchRoundTripsWithNoFormat(t *testing.T) {
	sketches := NewSketches()
	src := sketches.Add(XYPlane())
	src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	data, err := sketches.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	loaded := NewSketches()
	if err := loaded.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	out := loaded.Item(0)
	if n := out.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}
