// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/sketch"
)

// variableCoil builds a coil over the offset square with the given rail extras.
func variableCoil(t *testing.T, rows []CoilPitchRow, start, end CoilEndCondition) (*PartFeatures, *PartFeature) {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	pf := NewCoilFeatures(fs).Add(offsetSquareSketch(4, 1), 0, yAxis(),
		func() float64 { return 2 }, func() float64 { return 4 }, 0, ops.NewBody)
	def := pf.Definition().(*CoilFeature).Definition()
	def.PitchRows, def.StartEnd, def.EndEnd = rows, start, end
	fs.Recompute()
	return fs, pf
}

// TestVariablePitchCoilClimbsByPitchIntegral: a variable-pitch spring's axial
// span is the trapezoid pitch integral, not pitch × revolutions (M06-F09, #624).
func TestVariablePitchCoilClimbsByPitchIntegral(t *testing.T) {
	// Pitch 1 → 3 over 4 turns: height = (1+3)/2·4 = 8, plus profile height 1.
	fs, pf := variableCoil(t, []CoilPitchRow{
		{Pitch: 1, Revolution: 0},
		{Pitch: 3, Revolution: 4},
	}, CoilEndCondition{}, CoilEndCondition{})
	if !pf.Health().OK() {
		t.Fatalf("variable coil went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("variable coil is not a valid solid: %+v", r)
	}
	if span := body.RangeBox().Max.Y - body.RangeBox().Min.Y; relErr(span, 8+1) > 0.05 {
		t.Errorf("axial span = %g, want ≈9 (trapezoid pitch integral + profile height)", span)
	}
}

// TestFlatEndCoilAddsSweepWithoutRise: a flat end adds winding but no axial
// advance — the span stays the constant-pitch height while the rail gains the
// extra sweep angle.
func TestFlatEndCoilAddsSweepWithoutRise(t *testing.T) {
	fs, pf := variableCoil(t, nil, CoilEndCondition{},
		CoilEndCondition{Flat: true, TransitionAngle: stdmath.Pi / 2, FlatAngle: stdmath.Pi})
	if !pf.Health().OK() {
		t.Fatalf("flat-end coil went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("flat-end coil is not a valid solid: %+v", r)
	}
	// Constant pitch 2 over 4 turns climbs 8; the transition adds a quarter
	// turn of decaying pitch (2+0)/2·0.25 = 0.25; the flat sweep adds none.
	if span := body.RangeBox().Max.Y - body.RangeBox().Min.Y; relErr(span, 8+0.25+1) > 0.05 {
		t.Errorf("axial span = %g, want ≈9.25 (constant rise + transition quarter-turn)", span)
	}
}

// TestVariableCoilRoundTrip: pitch rows and end conditions survive the
// feature recipe round-trip.
func TestVariableCoilRoundTrip(t *testing.T) {
	_, pf := variableCoil(t, []CoilPitchRow{
		{Pitch: 1, Revolution: 0},
		{Pitch: 3, Revolution: 4},
	}, CoilEndCondition{}, CoilEndCondition{Flat: true, TransitionAngle: stdmath.Pi})

	def := pf.Definition().(*CoilFeature).Definition()
	// Re-anchor on a keyed origin axis so the axis ref round-trips.
	work := NewWorkGeometry()
	axis, ok := work.AxisByRef(OriginYAxis)
	if !ok {
		t.Fatal("origin Y axis missing from fresh work geometry")
	}
	def.Axis = axis
	index := sketchList{sks: []*sketch.Sketch{def.Sketch}}
	data, err := serializeCoil(def, index)
	if err != nil {
		t.Fatalf("serializeCoil: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	restored, err := restoreCoil(fresh, data, index, work)
	if err != nil {
		t.Fatalf("restoreCoil: %v", err)
	}
	rdef := restored.Definition().(*CoilFeature).Definition()
	if len(rdef.PitchRows) != 2 || rdef.PitchRows[1].Pitch != 3 {
		t.Errorf("restored rows = %+v, want the 2-row table", rdef.PitchRows)
	}
	if !rdef.EndEnd.Flat || rdef.EndEnd.TransitionAngle != stdmath.Pi {
		t.Errorf("restored end condition = %+v, want the flat π transition", rdef.EndEnd)
	}
}
