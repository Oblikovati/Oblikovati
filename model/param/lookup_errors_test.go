// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"strings"
	"testing"
)

const missingID = ID(999999)

// readOnlyParam adds a reference parameter (a read-only kind) for the read-only error paths.
func readOnlyParam(t *testing.T, ps *Parameters) *Parameter {
	t.Helper()
	p, err := ps.AddReferenceParameter("ref", Quantity{Value: 1, Unit: Length})
	if err != nil {
		t.Fatalf("AddReferenceParameter: %v", err)
	}
	return p
}

// TestParametersLookupMissErrors covers the "no parameter/group/derived-table with id" and
// read-only branches that the named-constant extraction touched — pure error paths reached
// by calling each mutator with an unknown id or on a read-only parameter.
func TestParametersLookupMissErrors(t *testing.T) {
	ps := NewParameters()
	ro := readOnlyParam(t, ps)

	wantErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}

	wantErr("SetExpression/missing", ps.SetExpression(missingID, "1"))
	wantErr("SetExpression/readonly", ps.SetExpression(ro.ID(), "1"))
	wantErr("SetValue/missing", ps.SetValue(missingID, Quantity{Value: 2, Unit: Length}))
	wantErr("SetValue/readonly", ps.SetValue(ro.ID(), Quantity{Value: 2, Unit: Length}))
	wantErr("DeleteGroup/missing", ps.DeleteGroup("nope", false))
	wantErr("AddToGroup/missing", ps.AddToGroup(missingID, "nope"))
	wantErr("RemoveFromGroup/missingParam", ps.RemoveFromGroup(missingID, "nope"))
	wantErr("RemoveFromAllGroups/missing", ps.RemoveFromAllGroups(missingID))
	wantErr("Rename/missing", ps.Rename(missingID, "x"))
	wantErr("Delete/missing", ps.Delete(missingID))
	wantErr("SetDerivedTableLinked/missing", ps.SetDerivedTableLinked(404, nil, nil))
	wantErr("SyncDerivedTable/missing", ps.SyncDerivedTable(404, nil, true))
	wantErr("DeleteDerivedTable/missing", ps.DeleteDerivedTable(404))

	if _, err := ps.CopyToUser(missingID); err == nil {
		t.Error("CopyToUser/missing: expected an error, got nil")
	}
}

// TestParameterReadOnlyEditErrors covers Parameter-level edit guards on a read-only parameter.
func TestParameterReadOnlyEditErrors(t *testing.T) {
	ps := NewParameters()
	ro := readOnlyParam(t, ps)
	for name, err := range map[string]error{
		"SetExpression":     ro.SetExpression("1"),
		"SetValue":          ro.SetValue(Quantity{Value: 1, Unit: Length}),
		"SetExpressionList": ro.SetExpressionList([]string{"a", "b"}, false),
		"SetText":           ro.SetText("x"),
		"SetBool":           ro.SetBool(true),
	} {
		if err == nil {
			t.Errorf("%s on a read-only parameter: expected an error", name)
		} else if !strings.Contains(err.Error(), "read-only") {
			t.Errorf("%s error = %q, want it to mention read-only", name, err)
		}
	}
}

// TestRemoveFromGroupMissingGroup covers the "no group" branch with a valid parameter but an
// unknown group key (distinct from the missing-parameter branch).
func TestRemoveFromGroupMissingGroup(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddUserParameter("w", "10 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	if err := ps.RemoveFromGroup(p.ID(), "no-such-group"); err == nil {
		t.Error("RemoveFromGroup with an unknown group key should error")
	}
}
