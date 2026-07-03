// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// TestAssemblyExtrudeEditableDistance checks the extrude exposes its depth as a single
// editable length scalar whose Set reflows the closure the next recompute reads (#725).
func TestAssemblyExtrudeEditableDistance(t *testing.T) {
	f := NewAssemblyExtrudeFeature(nil, 0, ops.Cut, func() float64 { return 3 })
	ps := f.EditableParams()
	if len(ps) != 1 || ps[0].Label != "Distance" || ps[0].Unit != param.Length {
		t.Fatalf("EditableParams = %+v, want one Distance length scalar", ps)
	}
	if got := ps[0].Get(); got != 3 {
		t.Errorf("Get = %g, want 3", got)
	}
	ps[0].Set(8)
	if got := f.EditableParams()[0].Get(); got != 8 {
		t.Errorf("after Set(8) the distance closure should reflow to 8, got %g", got)
	}
}

// TestAssemblyRevolveEditableAngle checks the revolve exposes its sweep as a single
// editable angle scalar whose Set reflows the closure the next recompute reads (#725).
func TestAssemblyRevolveEditableAngle(t *testing.T) {
	f := NewAssemblyRevolveFeature(nil, 0, nil, ops.Cut, func() float64 { return stdmath.Pi })
	ps := f.EditableParams()
	if len(ps) != 1 || ps[0].Label != "Angle" || ps[0].Unit != param.Angle {
		t.Fatalf("EditableParams = %+v, want one Angle scalar", ps)
	}
	if got := ps[0].Get(); got != stdmath.Pi {
		t.Errorf("Get = %g, want π", got)
	}
	ps[0].Set(stdmath.Pi / 2)
	if got := f.EditableParams()[0].Get(); got != stdmath.Pi/2 {
		t.Errorf("after Set(π/2) the angle closure should reflow, got %g", got)
	}
}

// TestAssemblySweepEditableTwist checks the sweep exposes its total twist as a single editable
// angle scalar whose Set reflows the closure the next recompute reads — the #1648 parity with the
// extrude/revolve siblings, so a placed assembly sweep edits in place.
func TestAssemblySweepEditableTwist(t *testing.T) {
	f := NewAssemblySweepFeature(nil, 0, ops.Cut, nil, func() float64 { return 0.4 })
	ps := f.EditableParams()
	if len(ps) != 1 || ps[0].Label != "Twist" || ps[0].Unit != param.Angle {
		t.Fatalf("EditableParams = %+v, want one Twist angle scalar", ps)
	}
	if got := ps[0].Get(); got != 0.4 {
		t.Errorf("Get = %g, want 0.4", got)
	}
	ps[0].Set(stdmath.Pi / 3)
	if got := f.EditableParams()[0].Get(); got != stdmath.Pi/3 {
		t.Errorf("after Set(π/3) the twist closure should reflow, got %g", got)
	}
}

// TestAssemblySweepTwistChangesGeometry proves the editable twist is not cosmetic: two sweeps that
// differ only in twist build different tools, so editing the twist actually reshapes the swept
// geometry (#1648 guarding test — an edit that recompiles to the same body would be a silent no-op).
func TestAssemblySweepTwistChangesGeometry(t *testing.T) {
	path := []math.Point3{math.P3(0, 0, 0), math.P3(0, 0, 5)}
	straight, err := NewAssemblySweepFeature(squareSketch(2), 0, ops.Join, path, nil).buildTool()
	if err != nil {
		t.Fatalf("straight sweep buildTool: %v", err)
	}
	twisted, err := NewAssemblySweepFeature(squareSketch(2), 0, ops.Join, path, func() float64 { return stdmath.Pi / 4 }).buildTool()
	if err != nil {
		t.Fatalf("twisted sweep buildTool: %v", err)
	}
	if straight.RangeBox() == twisted.RangeBox() {
		t.Errorf("a π/4 twist should reshape the swept tool, but its range box is unchanged: %+v", straight.RangeBox())
	}
}
