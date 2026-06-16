// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// twoHolePunchSketch builds a sketch with two small square profiles (the punch die pattern),
// placed inside a 4×4 sheet.
func twoHolePunchSketch() *sketch.Sketch {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	for _, c := range []math.Point2{math.P2(1, 1), math.P2(3, 3)} {
		a := sk.Points().Add(math.P2(c.X-0.3, c.Y-0.3))
		b := sk.Points().Add(math.P2(c.X+0.3, c.Y-0.3))
		d := sk.Points().Add(math.P2(c.X+0.3, c.Y+0.3))
		e := sk.Points().Add(math.P2(c.X-0.3, c.Y+0.3))
		sk.Lines().Add(a, b)
		sk.Lines().Add(b, d)
		sk.Lines().Add(d, e)
		sk.Lines().Add(e, a)
	}
	return sk
}

// TestPunchStampsEveryProfile a punch removes all of the sketch's closed profiles in one shot,
// leaving one watertight solid with material removed.
func TestPunchStampsEveryProfile(t *testing.T) {
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()
	full := sheetVolume(fs.Result()[0])

	pf := NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{Sketch: twoHolePunchSketch()})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("punch sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	// Two 0.6×0.6 holes through a 2 mm sheet remove 2·0.36·0.2 = 0.144 cm³.
	if v := sheetVolume(body); !(v < full-0.1) {
		t.Errorf("punch removed too little: %g vs %g", v, full)
	}
}

// TestPunchRejectsEmptySketch a punch with no closed profile is sick.
func TestPunchRejectsEmptySketch(t *testing.T) {
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	empty := sketch.NewSketches().Add(sketch.XYPlane()) // no profiles
	pf := NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{Sketch: empty})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("punch with no closed profile should be sick")
	}
}

// TestPunchRoundTrip a punch persists its sketch + depth and restores.
func TestPunchRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := twoHolePunchSketch()
	NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{Sketch: sk, Depth: constFloat(0.1)})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Kind != "sheet-metal-punch" || data[0].SheetMetalPunch == nil || data[0].SheetMetalPunch.Depth != 0.1 {
		t.Fatalf("marshaled = %+v", data[0])
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-punch" {
		t.Errorf("restored %d features, want one punch", fresh.Count())
	}
}
