// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
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
