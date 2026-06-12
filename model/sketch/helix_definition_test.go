// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	gmath "oblikovati.org/math"
)

// testHelix returns a 3-turn, pitch-1, radius-2 helix about +Z.
func testHelix(t *testing.T) (*Sketch3D, *HelicalCurve3D) {
	t.Helper()
	s := NewSketches3D().Add()
	z, err := gmath.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("z axis: %v", err)
	}
	return s, s.AddHelix3D(gmath.P3(0, 0, 0), z, 2, 1, 0, 3, false)
}

// TestVariableShapeRegeneratesCurve: a row table turns the analytic helix
// into a variable kernel curve whose end point honors the table's heights —
// the standard variable-pitch spring (M06-F09, #624).
func TestVariableShapeRegeneratesCurve(t *testing.T) {
	_, h := testHelix(t)
	err := h.SetVariableShape(types.HelixShapePitchRevolution, []HelixRow{
		{Diameter: 4, Pitch: 0.2},
		{Diameter: 4, Pitch: 1, Revolution: 2},
		{Diameter: 4, Pitch: 0.2, Revolution: 4},
	})
	if err != nil {
		t.Fatalf("SetVariableShape: %v", err)
	}
	cu, err := h.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	if _, ok := cu.(geom.VariableHelix3d); !ok {
		t.Fatalf("curve = %T, want a variable helix", cu)
	}
	// Heights: (0.2+1)/2·2 + (1+0.2)/2·2 = 2.4.
	end := cu.PointAt(1)
	if math.Abs(float64(end.Z)-2.4) > 1e-9 {
		t.Errorf("end height = %v, want 2.4 (trapezoid pitch integral)", end.Z)
	}
	if h.Turns != 4 {
		t.Errorf("entity turns = %v, want the table's 4", h.Turns)
	}
}

// TestVariableShapeInterpolatesAndHeightRows: zero diameters interpolate
// between specified rows, and height-positioned rows convert via the
// closed-form trapezoid inversion.
func TestVariableShapeInterpolatesAndHeightRows(t *testing.T) {
	_, h := testHelix(t)
	err := h.SetVariableShape(types.HelixShapePitchHeight, []HelixRow{
		{Diameter: 2, Pitch: 1},
		{Pitch: 1, Height: 1}, // diameter interpolates to 3
		{Diameter: 4, Pitch: 1, Height: 2},
	})
	if err != nil {
		t.Fatalf("SetVariableShape: %v", err)
	}
	rows := h.Definition().Rows
	if math.Abs(rows[1].Diameter-3) > 1e-12 {
		t.Errorf("interpolated diameter = %v, want 3", rows[1].Diameter)
	}
	// Constant pitch 1: 1 cm of height = exactly 1 turn per row.
	stations, err := h.definitionStations()
	if err != nil {
		t.Fatalf("definitionStations: %v", err)
	}
	if math.Abs(stations[1].Turn-1) > 1e-12 || math.Abs(stations[2].Turn-2) > 1e-12 {
		t.Errorf("station turns = %v/%v, want 1/2", stations[1].Turn, stations[2].Turn)
	}
}

// TestFlatEndsAppendStations: a flat end adds the transition + flat sweeps —
// the curve gains the extra quarter turns and its tail advances no further
// axially.
func TestFlatEndsAppendStations(t *testing.T) {
	_, h := testHelix(t)
	h.SetEndConditions(nil, &HelixEndCondition{
		Kind: types.HelixEndFlat, TransitionAngle: math.Pi / 2, FlatAngle: math.Pi / 2,
	})
	cu, err := h.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	v, ok := cu.(geom.VariableHelix3d)
	if !ok {
		t.Fatalf("curve = %T, want the station-compiled variable helix", cu)
	}
	if got := v.TotalTurns(); math.Abs(got-3.5) > 1e-12 {
		t.Errorf("total turns = %v, want 3.5 (3 + two quarter sweeps)", got)
	}
	// The flat sweep's end keeps the transition-end height: zero pitch.
	flatStart := v.PointAt((3.25) / 3.5)
	end := v.PointAt(1)
	if math.Abs(float64(end.Z-flatStart.Z)) > 1e-9 {
		t.Errorf("flat sweep advanced axially (%v → %v), want constant height", flatStart.Z, end.Z)
	}
	if view := (HelixDefinitionView{H: h}); view.EndEnd() != types.HelixEndFlat || view.StartEnd() != types.HelixEndNatural {
		t.Errorf("definition view ends = %v/%v, want natural/flat", view.StartEnd(), view.EndEnd())
	}
}

// TestHelixDefinitionRoundTrip: shape kind, rows and end conditions survive
// the 3D recipe round-trip and regenerate the same curve.
func TestHelixDefinitionRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	h := s.AddHelix3D(gmath.P3(0, 0, 0), z, 2, 1, 0, 3, false)
	if err := h.SetVariableShape(types.HelixShapePitchRevolution, []HelixRow{
		{Diameter: 4, Pitch: 0.2},
		{Diameter: 6, Pitch: 1, Revolution: 3},
	}); err != nil {
		t.Fatalf("SetVariableShape: %v", err)
	}
	h.SetEndConditions(&HelixEndCondition{Kind: types.HelixEndFlat, TransitionAngle: math.Pi}, nil)

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("MarshalRecipe3D: %v", err)
	}
	fresh := NewSketches3D()
	if err := fresh.ApplyRecipe3D(data); err != nil {
		t.Fatalf("ApplyRecipe3D: %v", err)
	}
	var restored *HelicalCurve3D
	for _, e := range fresh.Item(0).Entities() {
		if hc, ok := e.(*HelicalCurve3D); ok {
			restored = hc
		}
	}
	if restored == nil {
		t.Fatal("helix missing after round-trip")
	}
	def := restored.Definition()
	if !def.Variable() || len(def.Rows) != 2 || def.Rows[1].Diameter != 6 {
		t.Fatalf("restored definition = %+v, want the 2-row variable shape", def)
	}
	if def.Start.Kind != types.HelixEndFlat || def.Start.TransitionAngle != math.Pi {
		t.Errorf("restored start end = %+v, want the flat π transition", def.Start)
	}
	want, err := h.Curve()
	if err != nil {
		t.Fatalf("source curve: %v", err)
	}
	got, err := restored.Curve()
	if err != nil {
		t.Fatalf("restored curve: %v", err)
	}
	for i := 0; i <= 16; i++ {
		u := float64(i) / 16
		if d := float64(want.PointAt(u).DistanceTo(got.PointAt(u))); d > 1e-9 {
			t.Fatalf("curves diverge at t=%v by %g", u, d)
		}
	}
}
