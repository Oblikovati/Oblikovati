// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestCosmeticBendLeavesGeometryUnchanged a cosmetic bend records the fold but does not deform
// the sheet — the body is the flat base, unchanged, and the feature is healthy.
func TestCosmeticBendLeavesGeometryUnchanged(t *testing.T) {
	fs, line := sheetForBend(t, 4, 2)
	fs.Recompute()
	flat := fs.Result()[0]
	flatBox := flat.RangeBox()

	pf := NewSheetMetalCosmeticBendFeatures(fs).Add(&SheetMetalCosmeticBendDefinition{Sketch: line, LineIndex: 0})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("cosmetic bend sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if box := body.RangeBox(); box.Max.Z != flatBox.Max.Z {
		t.Errorf("cosmetic bend changed the geometry: maxZ %g → %g (it must not fold)", flatBox.Max.Z, box.Max.Z)
	}
}

// TestCosmeticBendReportsBendSpec the cosmetic bend contributes its angle/radius to the bend
// table (BendLineage), defaulting to a 90° fold at the rule radius.
func TestCosmeticBendReportsBendSpec(t *testing.T) {
	f := &SheetMetalCosmeticBendFeature{def: &SheetMetalCosmeticBendDefinition{}}
	specs := f.BendSpecs(0.2)
	if len(specs) != 1 {
		t.Fatalf("BendSpecs = %d, want 1", len(specs))
	}
	if stdmath.Abs(specs[0].Angle-stdmath.Pi/2) > 1e-12 {
		t.Errorf("default cosmetic bend angle = %g, want π/2", specs[0].Angle)
	}
	if specs[0].Radius != 0 {
		t.Errorf("a nil radius override should report 0 (rule default), got %g", specs[0].Radius)
	}
}

// TestCosmeticBendRejectsBadLine a line index outside the sketch makes the feature sick (the
// dangling bend line is reported, not silently ignored).
func TestCosmeticBendRejectsBadLine(t *testing.T) {
	fs, line := sheetForBend(t, 4, 2)
	pf := NewSheetMetalCosmeticBendFeatures(fs).Add(&SheetMetalCosmeticBendDefinition{Sketch: line, LineIndex: 7})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("cosmetic bend with an out-of-range line index should be sick")
	}
}

// TestCosmeticBendRoundTrip a cosmetic bend persists its bend line + angle/radius and restores.
func TestCosmeticBendRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 4))
	NewSheetMetalCosmeticBendFeatures(fs).Add(&SheetMetalCosmeticBendDefinition{
		Sketch: sk, LineIndex: 0, Angle: constFloat(stdmath.Pi / 3), Radius: constFloat(0.3),
	})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Kind != "sheet-metal-cosmetic-bend" || data[0].SheetMetalCosmeticBend == nil {
		t.Fatalf("marshaled = %+v", data[0])
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-cosmetic-bend" {
		t.Errorf("restored %d features, want one cosmetic bend", fresh.Count())
	}
}
