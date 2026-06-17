// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// TestSetDocumentUnitsAppliesToActivePart drives the write side of the units
// dialog: an edited units object lands on the active part (units, precision and
// format), and there is no active-part no-op crash.
func TestSetDocumentUnitsAppliesToActivePart(t *testing.T) {
	s := NewSession()
	// No active part: SetDocumentUnits is a safe no-op.
	s.SetDocumentUnits(param.DefaultUnitsOfMeasure())

	doc, err := compdef.AddPart(s.Workspace(), "p.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := doc.Content().(*compdef.PartComponentDefinition)

	u := s.DocumentUnits().Clone()
	if err := u.SetPreferred(param.Length, "in"); err != nil {
		t.Fatal(err)
	}
	if err := u.SetLengthPrecision(5); err != nil {
		t.Fatal(err)
	}
	u.SetAngleFormat(param.AngleDMS)
	s.SetDocumentUnits(u)

	got := def.Units()
	if got.PreferredName(param.Length) != "in" || got.LengthPrecision() != 5 || got.AngleFormat() != param.AngleDMS {
		t.Errorf("units after SetDocumentUnits = %+v, want in / prec 5 / DMS", got)
	}
}

// TestUnitsSettingsOpenClose drives the dialog's open/close flag.
func TestUnitsSettingsOpenClose(t *testing.T) {
	s := NewSession()
	if s.UnitsSettingsOpen() {
		t.Error("units settings should start closed")
	}
	s.OpenUnitsSettings()
	if !s.UnitsSettingsOpen() {
		t.Error("OpenUnitsSettings did not open the dialog")
	}
	s.CloseUnitsSettings()
	if s.UnitsSettingsOpen() {
		t.Error("CloseUnitsSettings did not close the dialog")
	}
}

// TestUnitOptionLists checks each category offers its expected units.
func TestUnitOptionLists(t *testing.T) {
	has := func(opts []string, want string) bool {
		for _, o := range opts {
			if o == want {
				return true
			}
		}
		return false
	}
	if !has(LengthUnitOptions(), "mm") || !has(LengthUnitOptions(), "in") {
		t.Error("length options missing mm/in")
	}
	if !has(AngleUnitOptions(), "deg") || !has(AngleUnitOptions(), "rad") {
		t.Error("angle options missing deg/rad")
	}
	if !has(MassUnitOptions(), "kg") || !has(TimeUnitOptions(), "s") {
		t.Error("mass/time options missing kg/s")
	}
}

// TestApplyUnitsHelpers covers the pure apply helpers behind the dialog controls.
func TestApplyUnitsHelpers(t *testing.T) {
	u := param.DefaultUnitsOfMeasure().Clone()

	if !ApplyUnit(&u, param.Length, "in") || u.PreferredName(param.Length) != "in" {
		t.Error("ApplyUnit(length, in) should change to in")
	}
	if ApplyUnit(&u, param.Length, "in") {
		t.Error("ApplyUnit to the same unit should be a no-op")
	}
	if ApplyUnit(&u, param.Length, "deg") { // wrong category → rejected
		t.Error("ApplyUnit with a wrong-category unit should be a no-op")
	}

	if !ApplyLengthFormat(&u, "fractional") || u.LengthFormat() != types.DisplayFormatFractional {
		t.Error("ApplyLengthFormat(fractional) should change the format")
	}
	if ApplyLengthFormat(&u, "fractional") || ApplyLengthFormat(&u, "bogus") {
		t.Error("ApplyLengthFormat unchanged/invalid should be a no-op")
	}

	if !ApplyLengthPrecision(&u, 5) || u.LengthPrecision() != 5 {
		t.Error("ApplyLengthPrecision(5) should change precision")
	}
	if ApplyLengthPrecision(&u, 5) || ApplyLengthPrecision(&u, -1) {
		t.Error("ApplyLengthPrecision unchanged/negative should be a no-op")
	}
	if !ApplyAnglePrecision(&u, 4) || ApplyAnglePrecision(&u, 4) {
		t.Error("ApplyAnglePrecision change/no-op wrong")
	}

	if !ApplyAngleDMS(&u, true) || u.AngleFormat() != param.AngleDMS {
		t.Error("ApplyAngleDMS(true) should switch to DMS")
	}
	if ApplyAngleDMS(&u, true) {
		t.Error("ApplyAngleDMS to the same state should be a no-op")
	}
	if !ApplyAngleDMS(&u, false) || u.AngleFormat() != param.AngleDecimal {
		t.Error("ApplyAngleDMS(false) should switch back to decimal")
	}
}
