// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// rectPunchSketch builds a sketch with one w×h rectangle centred at (cx,cy) — an asymmetric punch
// die whose rotation is observable.
func rectPunchSketch(cx, cy, w, h float64) *sketch.Sketch {
	return rectSketchOn(sketch.XYPlane(), cx-w/2, cy-h/2, cx+w/2, cy+h/2)
}

// p3Near reports whether two model points are within tol on every axis.
func p3Near(a, b math.Point3, tol float64) bool {
	return stdmath.Abs(float64(a.X-b.X)) < tol && stdmath.Abs(float64(a.Y-b.Y)) < tol &&
		stdmath.Abs(float64(a.Z-b.Z)) < tol
}

// TestRotatedPlaneAboutCentroid the rotated frame turns a point about the centroid: the centroid is
// a fixed point, and a point one unit +X of it lands one unit +Y after a 90° turn.
func TestRotatedPlaneAboutCentroid(t *testing.T) {
	t.Parallel()
	rp := rotatedPlaneAboutCentroid(sketch.XYPlane(), math.P2(2, 0), stdmath.Pi/2)
	if got := rp.ToModel(math.P2(2, 0)); !p3Near(got, math.P3(2, 0, 0), 1e-9) {
		t.Errorf("centroid moved to %v, want (2,0,0)", got)
	}
	if got := rp.ToModel(math.P2(3, 0)); !p3Near(got, math.P3(2, 1, 0), 1e-9) {
		t.Errorf("(3,0) turned to %v, want (2,1,0) about the centroid", got)
	}
}

// TestPunchRotationTurnsButPreservesArea a rotated punch is a valid watertight solid that removes
// the SAME volume as the un-rotated punch (a rotation preserves the die's area) — proof the turn
// did not break or resize the cut.
func TestPunchRotationTurnsButPreservesArea(t *testing.T) {
	t.Parallel()
	removed := func(angle float64) float64 {
		ps := param.NewParameters()
		mustParam(t, ps, "Thickness", "2 mm")
		fs := NewPartFeatures(ps)
		NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
		fs.Recompute()
		full := sheetVolume(fs.Result()[0])
		pf := NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{
			Sketch: rectPunchSketch(2, 2, 2, 1), Angle: func() float64 { return angle },
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("punch (angle %g) sick: %+v", angle, pf.Health())
		}
		assertWatertightSolid(t, fs.Result()[0])
		return full - sheetVolume(fs.Result()[0])
	}
	flat, turned := removed(0), removed(stdmath.Pi/2)
	if stdmath.Abs(flat-turned) > 1e-4 {
		t.Errorf("rotated punch removed %.5f, unrotated %.5f; a rotation must preserve the area", turned, flat)
	}
	if flat <= 0 {
		t.Fatalf("punch removed nothing (%.5f)", flat)
	}
}

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
	t.Parallel()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps)
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
	t.Parallel()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	empty := sketch.NewSketches().Add(sketch.XYPlane()) // no profiles
	pf := NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{Sketch: empty})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("punch with no closed profile should be sick")
	}
}

// TestPunchRoundTrip a punch persists its sketch + depth and the die metadata (#1968) and restores.
func TestPunchRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := twoHolePunchSketch()
	NewSheetMetalPunchFeatures(fs).Add(&SheetMetalPunchDefinition{
		Sketch: sk, Depth: constFloat(0.1), Angle: constFloat(0.5), AcrossBends: true, UnfoldInFlat: true,
		ToolID: "D12", Representation: types.CentermarkPunchRepresentation,
	})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalPunch
	if data[0].Kind != "sheet-metal-punch" || d == nil || d.Depth != 0.1 {
		t.Fatalf("marshaled = %+v", data[0])
	}
	if d.Angle != 0.5 || !d.AcrossBends || !d.UnfoldInFlat || d.ToolID != "D12" || d.Representation != int32(types.CentermarkPunchRepresentation) {
		t.Errorf("payload = %+v, want angle 0.5 / acrossBends / unfoldInFlat / D12 / centermark", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*SheetMetalPunchFeature).Definition()
	if evalFloat(got.Angle) != 0.5 || !got.AcrossBends || !got.UnfoldInFlat || got.ToolID != "D12" ||
		got.Representation != types.CentermarkPunchRepresentation {
		t.Errorf("restored = %+v, want angle 0.5 / acrossBends / unfoldInFlat / D12 / centermark", got)
	}
}
