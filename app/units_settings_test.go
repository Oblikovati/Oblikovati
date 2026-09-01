// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// TestSetDocumentUnitsAppliesToActivePart drives the write side of the units
// dialog: an edited units object lands on the active part (units, precision and
// format), and there is no active-part no-op crash.
func TestSetDocumentUnitsAppliesToActivePart(t *testing.T) {
	t.Parallel()
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

// TestSessionUnitPrecisionFollowsDocument checks the accessors the head's ParameterInput formats
// with (#1519): LengthPrecision/AnglePrecision report the active part's display decimals, falling
// back to the metric defaults when there is no part.
func TestSessionUnitPrecisionFollowsDocument(t *testing.T) {
	t.Parallel()
	s := NewSession()
	def := param.DefaultUnitsOfMeasure()
	if got := s.LengthPrecision(); got != def.LengthPrecision() {
		t.Errorf("default length precision = %d, want %d", got, def.LengthPrecision())
	}
	if got := s.AnglePrecision(); got != def.AnglePrecision() {
		t.Errorf("default angle precision = %d, want %d", got, def.AnglePrecision())
	}
	if _, err := compdef.AddPart(s.Workspace(), "prec.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	u := s.DocumentUnits().Clone()
	if err := u.SetLengthPrecision(4); err != nil {
		t.Fatal(err)
	}
	if err := u.SetAnglePrecision(1); err != nil {
		t.Fatal(err)
	}
	s.SetDocumentUnits(u)
	if got := s.LengthPrecision(); got != 4 {
		t.Errorf("length precision = %d, want 4", got)
	}
	if got := s.AnglePrecision(); got != 1 {
		t.Errorf("angle precision = %d, want 1", got)
	}
}

// TestUnitsSettingsOpenClose drives the dialog's open/close flag.
func TestUnitsSettingsOpenClose(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if !slices.Contains(LengthUnitOptions(), "mm") || !slices.Contains(LengthUnitOptions(), "in") {
		t.Error("length options missing mm/in")
	}
	// Micrometres are now the smallest selectable length unit.
	if LengthUnitOptions()[0] != "µm" {
		t.Errorf("smallest length option = %q, want \"µm\"", LengthUnitOptions()[0])
	}
	if !slices.Contains(AngleUnitOptions(), "deg") || !slices.Contains(AngleUnitOptions(), "rad") {
		t.Error("angle options missing deg/rad")
	}
	if !slices.Contains(MassUnitOptions(), "kg") || !slices.Contains(TimeUnitOptions(), "s") {
		t.Error("mass/time options missing kg/s")
	}
}

// TestApplyUnitsHelpers covers the pure apply helpers behind the dialog controls.
func TestApplyUnitsHelpers(t *testing.T) {
	t.Parallel()
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
