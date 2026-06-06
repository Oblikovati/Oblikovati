// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"math"
	"testing"
)

// TestModelParameterFollowsDriverEdit is a regression for the parametric-update
// bug: editing a driver parameter through the graph must recompute the dependent
// parameters that reference it. A dimension parameter "od/2" stayed at its initial
// value when od changed because the edit went through Parameter.SetExpression
// (which updates only that parameter) instead of Parameters.SetExpression (which
// rewires edges and recomputes dependents).
func TestModelParameterFollowsDriverEdit(t *testing.T) {
	ps := NewParameters()
	od, err := ps.AddUserParameter("od", "7 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	half, err := ps.AddModelParameter("half", "od / 2")
	if err != nil {
		t.Fatalf("AddModelParameter: %v", err)
	}
	if got := half.ModelValue(); math.Abs(got-0.35) > 1e-9 {
		t.Fatalf("half initial = %v cm, want 0.35 (od/2 at od=7mm)", got)
	}

	if err := ps.SetExpression(od.ID(), "10 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if got := half.ModelValue(); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("half after od=10mm = %v cm, want 0.5 (dependent did not recompute)", got)
	}
}
