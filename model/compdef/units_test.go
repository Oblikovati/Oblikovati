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
