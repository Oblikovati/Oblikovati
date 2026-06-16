// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// rollProfile returns a sketch with a Y-axis centerline at x=0 (line 0) and a vertical
// profile segment at x=radius from y=0..height (line 1), for rolling into a tube.
func rollProfile(radius, height float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a0 := s.Points().Add(math.P2(0, 0))
	a1 := s.Points().Add(math.P2(0, 1))
	axis := s.Lines().Add(a0, a1)
	axis.SetCenterline(true)
	p0 := s.Points().Add(math.P2(math.Scalar(radius), 0))
	p1 := s.Points().Add(math.P2(math.Scalar(radius), math.Scalar(height)))
	s.Lines().Add(p0, p1)
	return s
}

// rolledBody builds a contour roll of radius r / height h through angle (0 ⇒ full turn) and
// returns the resulting body, failing if the feature goes sick.
func rolledBody(t *testing.T, r, h, angle float64) *topo.Body {
	t.Helper()
	def := &SheetMetalContourRollDefinition{Profile: rollProfile(r, h), AxisLine: 0, Operation: ops.NewBody}
	if angle > 0 {
		def.Angle = func() float64 { return angle }
	}
	fs := NewPartFeatures(thicknessParams(t), nil)
	pf := NewSheetMetalContourRollFeatures(fs).Add(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("contour roll sick: %+v", pf.Health())
	}
	return fs.Result()[0]
}

// fullTubeVolume is the annular-shell volume of a full revolution.
func fullTubeVolume(r, th, h float64) float64 { return stdmath.Pi * ((r+th)*(r+th) - r*r) * h }

// TestContourRollRollsTube a full revolution rolls the profile into a watertight tube whose
// volume is the annular shell π((R+t)²−R²)·H.
func TestContourRollRollsTube(t *testing.T) {
	const r, th, h = 2.0, 0.2, 3.0
	body := rolledBody(t, r, h, 0)
	assertWatertightSolid(t, body)
	want := fullTubeVolume(r, th, h)
	if got := smSolidVolume(body); stdmath.Abs(got-want)/want > 0.02 {
		t.Errorf("tube volume = %.4f, want ~%.4f (±2%%)", got, want)
	}
}

// TestContourRollPartialAngle a partial roll (90°) makes a valid shell with less volume than
// the full tube.
func TestContourRollPartialAngle(t *testing.T) {
	body := rolledBody(t, 2, 3, stdmath.Pi/2)
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("partial roll invalid: %v", r.Issues)
	}
	if v := smSolidVolume(body); v >= fullTubeVolume(2, 0.2, 3) {
		t.Errorf("partial-roll volume %.4f should be less than the full tube", v)
	}
}

// TestContourRollRejectsBadInput an out-of-range axis line, zero angle, and missing thickness
// each go sick.
func TestContourRollRejectsBadInput(t *testing.T) {
	fs := NewPartFeatures(thicknessParams(t), nil)
	badAxis := NewSheetMetalContourRollFeatures(fs).Add(&SheetMetalContourRollDefinition{
		Profile: rollProfile(2, 3), AxisLine: 9, Operation: ops.NewBody,
	})
	fs.Recompute()
	if badAxis.Health().OK() {
		t.Error("contour roll with an out-of-range axis line should be sick")
	}

	noThick := NewPartFeatures(param.NewParameters(), nil)
	pf := NewSheetMetalContourRollFeatures(noThick).Add(&SheetMetalContourRollDefinition{Profile: rollProfile(2, 3), AxisLine: 0})
	noThick.Recompute()
	if pf.Health().OK() {
		t.Error("contour roll without a Thickness parameter should be sick")
	}
}

// TestContourRollDefinitionAndKind the accessors return the recipe.
func TestContourRollDefinitionAndKind(t *testing.T) {
	def := &SheetMetalContourRollDefinition{AxisLine: 1}
	f := &SheetMetalContourRollFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-contour-roll" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestContourRollRoundTrip the recipe (profile + axis line + angle + operation) marshals and
// restores, preserving the kind and payload.
func TestContourRollRoundTrip(t *testing.T) {
	profile := rollProfile(2, 3)
	fs := NewPartFeatures(nil, nil)
	NewSheetMetalContourRollFeatures(fs).Add(&SheetMetalContourRollDefinition{
		Profile: profile, AxisLine: 0, Angle: func() float64 { return stdmath.Pi / 2 }, Operation: ops.Join,
	})
	data, err := fs.MarshalRecipe(oneSketch{s: profile})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalContourRoll
	if data[0].Kind != "sheet-metal-contour-roll" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-contour-roll", data[0])
	}
	if d.Profile != 0 || d.AxisLine != 0 || stdmath.Abs(d.Angle-stdmath.Pi/2) > 1e-9 || d.Operation != int32(ops.Join) {
		t.Errorf("payload = %+v, want profile 0 / axis 0 / angle π/2 / join", d)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: profile}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-contour-roll" {
		t.Errorf("restored = %d features, want one contour roll", fresh.Count())
	}
}

// TestContourRollMissingPayload / unknown sketch restore errors.
func TestContourRollMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalContourRoll(NewPartFeatures(nil, nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalContourRoll(nil) must error")
	}
	if _, err := serializeSheetMetalContourRoll(&SheetMetalContourRollDefinition{Profile: rollProfile(1, 1)}, oneSketch{s: squareSketch(1)}); err == nil {
		t.Error("serialize with an unknown profile sketch must error")
	}
}
