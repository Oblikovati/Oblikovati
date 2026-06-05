// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

func floatLabels(p ToolParams) []string {
	out := make([]string, len(p.Floats))
	for i, f := range p.Floats {
		out[i] = f.Label
	}
	return out
}

func intLabels(p ToolParams) []string {
	out := make([]string, len(p.Ints))
	for i, v := range p.Ints {
		out[i] = v.Label
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Every parameterized sketch tool exposes the expected labelled fields for the generic
// property dialog.
func TestToolParamsLabels(t *testing.T) {
	cases := []struct {
		tool   ParameterizedTool
		floats []string
		ints   []string
		texts  int
	}{
		{NewSketchMoveTool(), []string{"Δ X", "Δ Y"}, nil, 0},
		{NewSketchCopyTool(), []string{"Δ X", "Δ Y"}, nil, 0},
		{NewSketchStretchTool(), []string{"Δ X", "Δ Y"}, nil, 0},
		{NewSketchRotateTool(), []string{"Center X", "Center Y", "Angle (deg)"}, nil, 0},
		{NewSketchScaleTool(), []string{"Center X", "Center Y", "Factor"}, nil, 0},
		{NewSketchRectPatternTool(), []string{"Step 1 X", "Step 1 Y", "Step 2 X", "Step 2 Y"}, []string{"Count 1", "Count 2"}, 0},
		{NewSketchCircPatternTool(), []string{"Center X", "Center Y", "Angle (deg)"}, []string{"Count"}, 0},
		{NewSketchSlotTool(1), []string{"Width"}, nil, 0},
		{NewSketchChamferTool(0.5), []string{"Distance"}, nil, 0},
		{NewSketchTextTool(), []string{"Height"}, nil, 1},
		{NewSketchFilletTool(0.5), []string{"Radius"}, nil, 0},
		{NewSketchOffsetTool(0.5), []string{"Distance"}, nil, 0},
	}
	for _, c := range cases {
		p := c.tool.Params()
		if !sameStrings(floatLabels(p), c.floats) {
			t.Errorf("%s floats = %v, want %v", c.tool.Name(), floatLabels(p), c.floats)
		}
		if !sameStrings(intLabels(p), c.ints) {
			t.Errorf("%s ints = %v, want %v", c.tool.Name(), intLabels(p), c.ints)
		}
		if len(p.Texts) != c.texts {
			t.Errorf("%s texts = %d, want %d", c.tool.Name(), len(p.Texts), c.texts)
		}
	}
}

// Setting a float param (what the dialog does) drives the committed geometry — proving the
// closures are bound to the tool's real fields.
func TestToolParamFloatWiresThrough(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	tool := NewSketchMoveTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{})
	tool.Params().Floats[0].Set(5) // Δ X = 5 via the dialog binding
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	near2(t, l.A.Position(), 5, 0)
}

// An angle param is surfaced in degrees and converted to radians on the tool.
func TestToolParamAngleInDegrees(t *testing.T) {
	tool := NewSketchRotateTool()
	for _, f := range tool.Params().Floats {
		if f.Label == "Angle (deg)" {
			f.Set(90)
		}
	}
	// 90° == π/2 rad; read it back through the same binding.
	var got float64
	for _, f := range tool.Params().Floats {
		if f.Label == "Angle (deg)" {
			got = f.Get()
		}
	}
	if got != 90 {
		t.Errorf("angle round-trip = %v deg, want 90", got)
	}
}

// A text param sets the tool's string.
func TestToolParamTextWiresThrough(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewSketchTextTool()
	s.StartTool(tool)
	s.Click(100, 100)
	tool.Params().Texts[0].Set("HI")
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.TextBoxes().Count() != 1 {
		t.Fatalf("text boxes = %d, want 1", sk.TextBoxes().Count())
	}
}

func boolLabels(p ToolParams) []string {
	out := make([]string, len(p.Bools))
	for i, b := range p.Bools {
		out[i] = b.Label
	}
	return out
}

// The parameterized 3D-sketch tools expose their scalar/bool inputs to the generic dialog.
func TestSketch3DToolParams(t *testing.T) {
	if got := floatLabels(NewCircle3DTool().Params()); !sameStrings(got, []string{"Radius"}) {
		t.Errorf("Circle3D floats = %v, want [Radius]", got)
	}
	hp := NewHelix3DTool().Params()
	if !sameStrings(floatLabels(hp), []string{"Radius", "Pitch", "Turns"}) || !sameStrings(boolLabels(hp), []string{"Clockwise"}) {
		t.Errorf("Helix3D params = %v / %v, want [Radius Pitch Turns] / [Clockwise]", floatLabels(hp), boolLabels(hp))
	}
	if got := boolLabels(NewArc3DTool().Params()); !sameStrings(got, []string{"Counter-clockwise"}) {
		t.Errorf("Arc3D bools = %v, want [Counter-clockwise]", got)
	}
}

// Setting a 3D tool's float and bool params via the dialog bindings mutates the tool.
func TestSketch3DToolParamWiring(t *testing.T) {
	h := NewHelix3DTool()
	for _, f := range h.Params().Floats {
		if f.Label == "Pitch" {
			f.Set(2.5)
		}
	}
	for _, b := range h.Params().Bools {
		if b.Label == "Clockwise" {
			b.Set(true)
		}
	}
	var pitch float64
	var cw bool
	for _, f := range h.Params().Floats {
		if f.Label == "Pitch" {
			pitch = f.Get()
		}
	}
	for _, b := range h.Params().Bools {
		if b.Label == "Clockwise" {
			cw = b.Get()
		}
	}
	if pitch != 2.5 || !cw {
		t.Errorf("after Set: pitch=%v clockwise=%v, want 2.5/true", pitch, cw)
	}
}

// A tool with no parameters reports Empty so the head shows no dialog.
func TestToolParamsEmpty(t *testing.T) {
	if !(ToolParams{}).Empty() {
		t.Error("zero ToolParams should be Empty")
	}
	if NewSketchMoveTool().Params().Empty() {
		t.Error("Move tool should expose parameters")
	}
}
