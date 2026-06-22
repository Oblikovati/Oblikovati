// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/param"
)

func TestPartDefaultsToMetricUnits(t *testing.T) {
	d := NewPartComponentDefinition()
	if name := d.Units().PreferredName(param.Length); name != "mm" {
		t.Errorf("default part length unit = %q, want mm", name)
	}
}

func TestPartSetLengthUnit(t *testing.T) {
	d := NewPartComponentDefinition()
	if err := d.SetLengthUnit("in"); err != nil {
		t.Fatalf("SetLengthUnit: %v", err)
	}
	if name := d.Units().PreferredName(param.Length); name != "in" {
		t.Errorf("length unit after set = %q, want in", name)
	}
	if err := d.SetLengthUnit("deg"); err == nil {
		t.Error("setting a non-length unit as the length unit should error")
	}
}

func TestPartSetUnits(t *testing.T) {
	d := NewPartComponentDefinition()
	u := d.Units().Clone()
	if err := u.SetPreferred(param.Length, "in"); err != nil {
		t.Fatalf("set preferred: %v", err)
	}
	if err := u.SetLengthPrecision(5); err != nil {
		t.Fatalf("set precision: %v", err)
	}
	d.SetUnits(u)
	if name := d.Units().PreferredName(param.Length); name != "in" {
		t.Errorf("length unit after SetUnits = %q, want in", name)
	}
	if p := d.Units().LengthPrecision(); p != 5 {
		t.Errorf("length precision after SetUnits = %d, want 5", p)
	}
}

func TestAssemblySetUnits(t *testing.T) {
	a := NewAssemblyComponentDefinition()
	u := a.Units().Clone()
	if err := u.SetPreferred(param.Length, "ft"); err != nil {
		t.Fatalf("set preferred: %v", err)
	}
	a.SetUnits(u)
	if name := a.Units().PreferredName(param.Length); name != "ft" {
		t.Errorf("assembly length unit after SetUnits = %q, want ft", name)
	}
}

func TestApplyUnitsToReservedKeys(t *testing.T) {
	u := param.DefaultUnitsOfMeasure().Clone()
	if err := applyUnitsTo(&u, map[string]string{
		"lengthPrecision": "4", "anglePrecision": "1",
		"lengthFormat": "fractional", "angleFormat": "dms",
	}); err != nil {
		t.Fatalf("apply reserved keys: %v", err)
	}
	if u.LengthPrecision() != 4 || u.AnglePrecision() != 1 || u.AngleFormat() != param.AngleDMS {
		t.Errorf("reserved keys not applied: %+v", u)
	}
	for _, bad := range []map[string]string{
		{"lengthPrecision": "x"}, {"lengthFormat": "bogus"},
		{"angleFormat": "bogus"}, {"nonsense": "mm"},
	} {
		if err := applyUnitsTo(&u, bad); err == nil {
			t.Errorf("applyUnitsTo(%v) should error", bad)
		}
	}
}

// TestWorkingScaleRecipeRoundTrip is the ADR-0042 Phase 2 .obk round-trip (#1246): a
// document centred on a non-cm working unit persists and restores its working scale, so a
// reloaded µm part keeps its O(1) coordinates instead of being misread as cm.
func TestWorkingScaleRecipeRoundTrip(t *testing.T) {
	src, err := param.DefaultUnitsOfMeasure().CenteredOnLength("µm") // working scale 1e-4
	if err != nil {
		t.Fatal(err)
	}
	recipe := unitsRecipeFor(src)
	if recipe[keyWorkingScale] == "" {
		t.Fatalf("recipe omits %s for a non-cm document: %v", keyWorkingScale, recipe)
	}
	restored := param.DefaultUnitsOfMeasure().Clone()
	if err := applyUnitsTo(&restored, recipe); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.WorkingScale() != 1e-4 {
		t.Errorf("restored WorkingScale = %v, want 1e-4", restored.WorkingScale())
	}
}

// TestWorkingScaleDefaultOmittedAndMigrates covers two halves of one guarantee: a cm document
// omits the key (so existing .obk stay byte-identical), and a legacy recipe lacking the key
// restores the cm default (the automatic migration for pre-Phase-2 documents).
func TestWorkingScaleDefaultOmittedAndMigrates(t *testing.T) {
	if _, present := unitsRecipeFor(param.DefaultUnitsOfMeasure())[keyWorkingScale]; present {
		t.Error("cm document should omit the workingScale key")
	}
	legacy := param.DefaultUnitsOfMeasure().Clone()
	if err := applyUnitsTo(&legacy, map[string]string{"length": "mm"}); err != nil {
		t.Fatalf("legacy restore: %v", err)
	}
	if legacy.WorkingScale() != 1 {
		t.Errorf("legacy (no key) WorkingScale = %v, want cm default 1", legacy.WorkingScale())
	}
	// A corrupt value is rejected, not silently dropped — both a non-positive number and a
	// non-numeric string.
	for _, badVal := range []string{"0", "-1", "abc"} {
		bad := param.DefaultUnitsOfMeasure().Clone()
		if err := applyUnitsTo(&bad, map[string]string{keyWorkingScale: badVal}); err == nil {
			t.Errorf("workingScale %q should be rejected", badVal)
		}
	}
}
