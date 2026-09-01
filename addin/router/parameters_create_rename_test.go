// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/app"
)

// wantErr asserts that a wire method call fails (the router surfaces model rejections as errors).
func wantErr(t *testing.T, r *Router, s *app.Session, method, args string) {
	t.Helper()
	if _, err := r.Handle(s, method, []byte(args)); err == nil {
		t.Errorf("%s(%s): want an error, got success", method, args)
	}
}

// Non-numeric and model-kind parameter creation (#1845) and parameter rename (#1847) over the
// wire — the previously-unreachable model capability (AddTextUserParameter /
// AddBooleanUserParameter / AddModelParameter / Parameters.Rename).

// TestAddTextParameter creates a text user parameter: the literal string is the value, no units.
func TestAddTextParameter(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"finish","valueType":"text","expression":"anodized"}`, nil)
	d := getDetail(t, r, s, "finish")
	if d.Kind == "" || d.Value != "anodized" {
		t.Fatalf("text param = kind=%q value=%q, want a text value \"anodized\"", d.Kind, d.Value)
	}
	if d.Tolerance != nil {
		t.Errorf("text parameter should carry no tolerance, got %+v", d.Tolerance)
	}
}

// TestAddBooleanParameter creates a true/false user parameter from "true"/"false".
func TestAddBooleanParameter(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"vented","valueType":"boolean","expression":"true"}`, nil)
	if d := getDetail(t, r, s, "vented"); d.Value != "true" && d.Value != "True" {
		t.Fatalf("boolean param value = %q, want true", d.Value)
	}
}

// TestAddBooleanParameterRejectsNonBool: a non-boolean expression is a clean error.
func TestAddBooleanParameterRejectsNonBool(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	wantErr(t, r, s, "parameters.add", `{"name":"bad","valueType":"boolean","expression":"3 cm"}`)
}

// TestAddModelParameter creates a numeric parameter in the MODEL table (Kind=model).
func TestAddModelParameter(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"depth","kind":"model","expression":"5 mm"}`, nil)
	if d := getDetail(t, r, s, "depth"); d.Kind == "" {
		t.Fatalf("model param not created: %+v", d.ParameterInfo)
	}
}

// TestAddParameterUnknownValueType: an unknown valueType is a clean error.
func TestAddParameterUnknownValueType(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	wantErr(t, r, s, "parameters.add", `{"name":"x","valueType":"vector","expression":"1"}`)
}

// TestRenameParameterRewritesDependents renames a driving parameter and confirms its identity is
// kept and a dependent expression is rewritten to the new name (#1847).
func TestRenameParameterRewritesDependents(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"half","expression":"width / 2"}`, nil)
	call(t, r, s, "parameters.rename", `{"name":"width","newName":"span"}`, nil)

	if d := getDetail(t, r, s, "span"); d.Name != "span" || d.Expression != "4 cm" {
		t.Fatalf("renamed param = %+v, want span / 4 cm", d.ParameterInfo)
	}
	if d := getDetail(t, r, s, "half"); d.Expression != "span / 2" {
		t.Errorf("dependent expression = %q, want it rewritten to \"span / 2\"", d.Expression)
	}
}

// TestRenameParameterClashRejected: renaming onto an existing name is refused.
func TestRenameParameterClashRejected(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"height","expression":"2 cm"}`, nil)
	wantErr(t, r, s, "parameters.rename", `{"name":"height","newName":"width"}`)
}
