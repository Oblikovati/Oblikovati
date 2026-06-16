// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import (
	"math"
	"testing"
)

// TestNewBendDevelopsAllowanceAndDeduction checks NewBend fills the developed values from
// the unfold method, matching the method's own formulas for the same bend.
func TestNewBendDevelopsAllowanceAndDeduction(t *testing.T) {
	m := KFactorMethod(0.5)
	const angle, radius, thickness = math.Pi / 2, 0.2, 0.1

	b := NewBend("Flange1", angle, radius, thickness, m)

	if b.Feature != "Flange1" {
		t.Errorf("Feature = %q, want Flange1", b.Feature)
	}
	if b.Angle != angle || b.Radius != radius || b.Thickness != thickness {
		t.Errorf("geometry = (%g,%g,%g), want (%g,%g,%g)", b.Angle, b.Radius, b.Thickness, angle, radius, thickness)
	}
	if want := m.BendAllowance(angle, radius, thickness); b.Allowance != want {
		t.Errorf("Allowance = %g, want %g", b.Allowance, want)
	}
	if want := m.BendDeduction(angle, radius, thickness); b.Deduction != want {
		t.Errorf("Deduction = %g, want %g", b.Deduction, want)
	}
	// A 90° bend at K=0.5 develops to the neutral-axis arc (r + 0.5t)·angle.
	if got, want := b.Allowance, (radius+0.5*thickness)*angle; math.Abs(got-want) > 1e-12 {
		t.Errorf("Allowance = %g, want %g (neutral-axis arc)", got, want)
	}
}

// TestNewBendUsesActiveUnfoldMethod confirms a different method changes the developed length
// (so the record is fixed to the rule active at projection time, not a constant).
func TestNewBendUsesActiveUnfoldMethod(t *testing.T) {
	const angle, radius, thickness = math.Pi / 2, 0.2, 0.1
	loose := NewBend("B", angle, radius, thickness, KFactorMethod(0.5))
	tight := NewBend("B", angle, radius, thickness, KFactorMethod(0.25))
	if !(tight.Allowance < loose.Allowance) {
		t.Errorf("tighter K-factor allowance %g should be < %g", tight.Allowance, loose.Allowance)
	}
}
