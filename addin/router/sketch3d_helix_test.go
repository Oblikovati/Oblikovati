// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestVariableHelixOverWire: a helical entity created with shape rows reports
// a variable definition; editHelix swaps shapes and end conditions in place
// (M06-F09, #624).
func TestVariableHelixOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "sketch3d.create", `{"name":"Coil"}`, &wire.CreateSketch3DResult{})

	var added wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity",
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"20 mm","revolutions":4,
		  "rows":[{"diameter":"40 mm","pitch":"2 mm"},{"diameter":"40 mm","pitch":"8 mm","revolution":4}]}`,
		&added)
	if added.EntityID == 0 {
		t.Fatal("addEntity returned no id")
	}

	var view wire.HelixDefinitionView
	call(t, r, s, "sketch3d.editHelix",
		fmt.Sprintf(`{"sketchIndex":0,"entity":%d,"end":{"kind":"flat","transitionAngle":"90 deg","flatAngle":"180 deg"}}`, added.EntityID),
		&view)
	if !view.Variable || len(view.Rows) != 2 {
		t.Fatalf("definition view = %+v, want the 2-row variable shape", view)
	}
	if view.End == nil || view.End.Kind != "flat" {
		t.Fatalf("end condition = %+v, want flat", view.End)
	}
	if view.Start != nil {
		t.Errorf("start condition = %+v, want natural (absent)", view.Start)
	}
	if stdmath.Abs(view.Revolutions-4) > 1e-9 {
		t.Errorf("revolutions = %v, want the table's 4", view.Revolutions)
	}

	// Swap back to a constant shape: rows drop, the new pitch holds.
	var constant wire.HelixDefinitionView
	call(t, r, s, "sketch3d.editHelix",
		fmt.Sprintf(`{"sketchIndex":0,"entity":%d,"mode":"pitchRevolution","pitch":"5 mm","revolutions":3}`, added.EntityID),
		&constant)
	if constant.Variable || len(constant.Rows) != 0 {
		t.Errorf("after constant edit: %+v, want no rows", constant)
	}
	if stdmath.Abs(constant.Pitch-0.5) > 1e-9 || stdmath.Abs(constant.Revolutions-3) > 1e-9 {
		t.Errorf("constant shape = pitch %v / turns %v, want 0.5 cm / 3", constant.Pitch, constant.Revolutions)
	}
	if constant.End == nil || constant.End.Kind != "flat" {
		t.Errorf("end condition lost by the shape edit: %+v", constant.End)
	}

	if _, err := r.Handle(s, "sketch3d.editHelix",
		[]byte(fmt.Sprintf(`{"sketchIndex":0,"entity":%d,"mode":"wavy"}`, added.EntityID))); err == nil {
		t.Error("an unknown helix mode must be rejected")
	}
}
