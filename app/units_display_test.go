// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// TestUnitDisplayConversionsDefault covers the metric defaults (mm / deg): the
// converters are identity-ish but exercise the ToPreferred/FromPreferred path.
func TestUnitDisplayConversionsDefault(t *testing.T) {
	t.Parallel()
	s := NewSession() // no document ⇒ default mm/deg units
	if got := s.LengthToDisplay(1); got != 10 {
		t.Errorf("LengthToDisplay(1 cm) = %g, want 10 mm", got)
	}
	if got := s.LengthFromDisplay(10); got != 1 {
		t.Errorf("LengthFromDisplay(10 mm) = %g, want 1 cm", got)
	}
	if got := s.AngleDegToDisplay(90); math.Abs(got-90) > 1e-9 {
		t.Errorf("AngleDegToDisplay(90) = %g, want 90 deg", got)
	}
	if got := s.AngleDisplayToDeg(90); math.Abs(got-90) > 1e-9 {
		t.Errorf("AngleDisplayToDeg(90) = %g, want 90", got)
	}
}

// TestUnitDisplayConversionsInchRadian sets the document to inches and radians
// and checks the converters honor it.
func TestUnitDisplayConversionsInchRadian(t *testing.T) {
	t.Parallel()
	s := NewSession()
	part, err := compdef.AddPart(s.Workspace(), "widget.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	u := def.Units().Clone()
	if err := u.SetPreferred(param.Length, "in"); err != nil {
		t.Fatalf("set length unit: %v", err)
	}
	if err := u.SetPreferred(param.Angle, "rad"); err != nil {
		t.Fatalf("set angle unit: %v", err)
	}
	def.SetUnits(u)

	// 2.54 cm = 1 in.
	if got := s.LengthToDisplay(2.54); math.Abs(got-1) > 1e-9 {
		t.Errorf("LengthToDisplay(2.54 cm) = %g, want 1 in", got)
	}
	if got := s.LengthFromDisplay(1); math.Abs(got-2.54) > 1e-9 {
		t.Errorf("LengthFromDisplay(1 in) = %g, want 2.54 cm", got)
	}
	// 180 deg = π rad.
	if got := s.AngleDegToDisplay(180); math.Abs(got-math.Pi) > 1e-9 {
		t.Errorf("AngleDegToDisplay(180) = %g, want π rad", got)
	}
	if got := s.AngleDisplayToDeg(math.Pi); math.Abs(got-180) > 1e-9 {
		t.Errorf("AngleDisplayToDeg(π) = %g, want 180 deg", got)
	}
}
