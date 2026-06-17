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
