// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// seededAssemblySession builds a session whose active document is an ASSEMBLY, so the
// parameter wire surface can be exercised against a non-part holder (M39-F03, #1559).
func seededAssemblySession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if _, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true); err != nil {
		t.Fatalf("add assembly: %v", err)
	}
	if _, ok := s.ActiveDocument().Content().(*compdef.AssemblyComponentDefinition); !ok {
		t.Fatalf("active document is not an assembly: %T", s.ActiveDocument().Content())
	}
	return New(opregistry.Default()), s
}

// TestParameterWireSurfaceOnAssembly drives parameters.add/list/get/set against an active
// assembly. Before M39-F03 these handlers resolved the active document as a part only, so an
// assembly returned "not a part"; now they resolve any parameter holder.
func TestParameterWireSurfaceOnAssembly(t *testing.T) {
	r, s := seededAssemblySession(t)

	var added wire.ParameterInfo
	call(t, r, s, "parameters.add", `{"name":"plateWidth","expression":"4 cm"}`, &added)
	if added.Name != "plateWidth" || added.Kind != "user" {
		t.Fatalf("added on assembly = %+v, want plateWidth/user", added)
	}
	if added.Value != "40 mm" {
		t.Errorf("added value = %q, want \"40 mm\" (4 cm in the assembly's mm display units)", added.Value)
	}

	var list wire.ListParametersResult
	call(t, r, s, "parameters.list", `{}`, &list)
	if len(list.Parameters) != 1 || list.Parameters[0].Name != "plateWidth" {
		t.Fatalf("assembly parameters.list = %+v, want [plateWidth]", list.Parameters)
	}

	var got wire.ParameterInfo
	call(t, r, s, "parameters.get", `{"name":"plateWidth"}`, &got)
	if got.Expression != "4 cm" {
		t.Errorf("assembly parameters.get expression = %q, want \"4 cm\"", got.Expression)
	}

	var set wire.ParameterInfo
	call(t, r, s, "parameters.set", `{"name":"plateWidth","expression":"6 cm"}`, &set)
	if set.Expression != "6 cm" {
		t.Errorf("assembly parameters.set expression = %q, want \"6 cm\"", set.Expression)
	}
}

// TestDerivedTableWireListOnAssembly proves the derived-parameter-table list handler resolves
// an active assembly (empty list, but no "not a part" error) — the F02/F03 seam over the wire.
func TestDerivedTableWireListOnAssembly(t *testing.T) {
	r, s := seededAssemblySession(t)
	var out wire.ListDerivedParameterTablesResult
	call(t, r, s, "parameters.derivedTables.list", `{}`, &out)
	if len(out.Tables) != 0 {
		t.Errorf("fresh assembly derived tables = %+v, want none", out.Tables)
	}
}
