// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// sheetForRip builds a 4×4 sheet-metal wall (2 mm thick) on XY plus a rip-line sketch, and
// returns the engine and the rip sketch ready to add a rip.
func sheetForRip(t *testing.T, a, b math.Point2) (*PartFeatures, *sketch.Sketch) {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	rip := sketch.NewSketches().Add(sketch.XYPlane())
	rip.Lines().AddByTwoPoints(a, b)
	return fs, rip
}

// TestRipSlitsSheet a rip cuts a slit along the line, removing material and leaving one
// watertight solid (a partial rip line keeps the sheet connected).
func TestRipSlitsSheet(t *testing.T) {
	fs, rip := sheetForRip(t, math.P2(1, 2), math.P2(3, 2)) // a line across the middle, ends inside
	fs.Recompute()
	full := sheetVolume(fs.Result()[0])

	pf := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: rip, LineIndex: 0, Gap: func() float64 { return 0.05 }})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rip sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if v := sheetVolume(body); !(v < full) {
		t.Errorf("rip removed no material: %g vs %g", v, full)
	}
}

// TestRipRejectsBadInput a rip with an out-of-range line index, or a non-positive gap, is sick.
func TestRipRejectsBadInput(t *testing.T) {
	fs, rip := sheetForRip(t, math.P2(1, 2), math.P2(3, 2))
	bad := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: rip, LineIndex: 9, Gap: func() float64 { return 0.05 }})
	fs.Recompute()
	if bad.Health().OK() {
		t.Error("rip with an out-of-range line index should be sick")
	}

	fs2, rip2 := sheetForRip(t, math.P2(1, 2), math.P2(3, 2))
	zero := NewSheetMetalRipFeatures(fs2).Add(&SheetMetalRipDefinition{Sketch: rip2, LineIndex: 0, Gap: func() float64 { return 0 }})
	fs2.Recompute()
	if zero.Health().OK() {
		t.Error("rip with a non-positive gap should be sick")
	}
}

// TestRipRoundTrip a rip persists its line + gap and restores.
func TestRipRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(1, 2), math.P2(3, 2))
	NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: sk, LineIndex: 0, Gap: constFloat(0.05)})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Kind != "sheet-metal-rip" || data[0].SheetMetalRip == nil || data[0].SheetMetalRip.Gap != 0.05 {
		t.Fatalf("marshaled = %+v", data[0])
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-rip" {
		t.Errorf("restored %d features, want one rip", fresh.Count())
	}
}
