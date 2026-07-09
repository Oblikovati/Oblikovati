// SPDX-License-Identifier: GPL-2.0-only

package router

import "testing"

// Parameter type conversion over the wire (#1850): parameters.convert changes a parameter's kind
// (user/model/reference) in place, preserving identity and dependents; reference is read-only;
// built-in/auto, derived, and unknown targets are refused.

// TestConvertParameterUserModelRoundTrip converts the seeded "width" user→model and back, and
// confirms a dependent expression stays bound across the conversion.
func TestConvertParameterUserModelRoundTrip(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"half","expression":"width / 2"}`, nil)

	call(t, r, s, "parameters.convert", `{"name":"width","targetKind":"model"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Kind != "model" {
		t.Errorf("after convert, width kind = %q, want model", d.Kind)
	}
	if d := getDetail(t, r, s, "half"); d.Expression != "width / 2" {
		t.Errorf("dependent expression = %q, want it still bound to width", d.Expression)
	}

	call(t, r, s, "parameters.convert", `{"name":"width","targetKind":"user"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Kind != "user" {
		t.Errorf("after convert back, width kind = %q, want user", d.Kind)
	}
}

// TestConvertParameterToReferenceIsReadOnly: a reference parameter refuses parameters.set.
func TestConvertParameterToReferenceIsReadOnly(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.convert", `{"name":"width","targetKind":"reference"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Kind != "reference" {
		t.Fatalf("width kind = %q, want reference", d.Kind)
	}
	wantErr(t, r, s, "parameters.set", `{"name":"width","expression":"6 cm"}`)
}

// TestConvertParameterErrors covers the guard rails: an unsupported target, an unknown parameter,
// and an unknown targetKind spelling are all clean errors.
func TestConvertParameterErrors(t *testing.T) {
	r, s := seededSession(t)
	wantErr(t, r, s, "parameters.convert", `{"name":"width","targetKind":"derived"}`)
	wantErr(t, r, s, "parameters.convert", `{"name":"missing","targetKind":"model"}`)
	wantErr(t, r, s, "parameters.convert", `{"name":"width","targetKind":"banana"}`)
}
