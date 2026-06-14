// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/attr"
)

// TestPartPropertiesRoundTrip: a part's authored iProperties survive a recipe round-trip (save →
// reopen), so a part number set on a document is there when it is read back (#156).
func TestPartPropertiesRoundTrip(t *testing.T) {
	d := NewPartComponentDefinition()
	d.Properties().Set(attr.DesignTracking).Put("Part Number", attr.StringValue("BRK-001"))
	d.Properties().Set(attr.UserDefined).Put("Vendor", attr.StringValue("Acme"))

	data, err := d.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	got := NewPartComponentDefinition()
	if err := got.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if v := propertyString(t, got, attr.DesignTracking, "Part Number"); v != "BRK-001" {
		t.Errorf("restored Part Number = %q, want BRK-001", v)
	}
	if v := propertyString(t, got, attr.UserDefined, "Vendor"); v != "Acme" {
		t.Errorf("restored custom Vendor = %q, want Acme", v)
	}
}

// TestAssemblyPropertiesRoundTrip: an assembly's iProperties round-trip through its recipe too.
func TestAssemblyPropertiesRoundTrip(t *testing.T) {
	a := NewAssemblyComponentDefinition()
	a.Properties().Set(attr.SummaryInformation).Put("Title", attr.StringValue("Gearbox"))

	data, err := a.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	got := NewAssemblyComponentDefinition()
	if err := got.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	set, ok := got.Properties().Lookup(attr.SummaryInformation)
	if !ok {
		t.Fatal("restored assembly has no Summary Information set")
	}
	p, ok := set.Property("Title")
	if !ok {
		t.Fatal("restored assembly has no Title property")
	}
	if v, _ := p.Value().Str(); v != "Gearbox" {
		t.Errorf("restored Title = %q, want Gearbox", v)
	}
}

// TestPropertyValueTypesRoundTrip: every value type survives the (type, string) recipe encoding.
func TestPropertyValueTypesRoundTrip(t *testing.T) {
	cases := []attr.Value{
		attr.StringValue("hello"),
		attr.IntValue(-42),
		attr.FloatValue(3.5),
		attr.BoolValue(true),
		attr.BytesValue([]byte{1, 2, 3}),
	}
	for _, want := range cases {
		typ, val := valueToRecipe(want)
		got, err := valueFromRecipe(typ, val)
		if err != nil {
			t.Fatalf("valueFromRecipe(%q,%q): %v", typ, val, err)
		}
		if !got.Equal(want) {
			t.Errorf("round-trip of %v via (%q,%q) = %v", want, typ, val, got)
		}
	}
}

// propertyString reads a string property from a part's set, failing if absent or not a string.
func propertyString(t *testing.T, d *PartComponentDefinition, set, name string) string {
	t.Helper()
	s, ok := d.Properties().Lookup(set)
	if !ok {
		t.Fatalf("no property set %q", set)
	}
	p, ok := s.Property(name)
	if !ok {
		t.Fatalf("no property %q in set %q", name, set)
	}
	v, _ := p.Value().Str()
	return v
}
