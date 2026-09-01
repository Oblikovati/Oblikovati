// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// newSessionWithPart returns a session whose active document is an empty part — enough to
// drive the Parameters dialog headlessly.
func newSessionWithPart(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	return s
}

// partParams returns the active part's parameter collection.
func partParams(t *testing.T, s *Session) *param.Parameters {
	t.Helper()
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("activePart: %v", err)
	}
	return part.Parameters()
}

// rowByName finds a parameter row by name across the model and user tables.
func rowByName(rows []ParameterRow, name string) (ParameterRow, bool) {
	for _, r := range rows {
		if r.Name == name {
			return r, true
		}
	}
	return ParameterRow{}, false
}

// TestImportParametersRollsBackOnError: a rejected parameter import restores the pre-import
// snapshot and adds nothing — the snapshot/rollback path (now on the fast snapshot codec).
func TestImportParametersRollsBackOnError(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	ps := partParams(t, s)
	if _, err := ps.AddUserParameter("keep", "3 mm"); err != nil {
		t.Fatalf("seed param: %v", err)
	}
	before := ps.Count()

	// A structurally invalid import (numeric parameter with no expression) must be rejected.
	if _, _, err := s.ImportParameters(`<parameters><parameter name="x"/></parameters>`); err == nil {
		t.Fatal("ImportParameters accepted invalid XML")
	}
	if got := ps.Count(); got != before {
		t.Errorf("after rejected import: %d parameters, want %d (rollback failed)", got, before)
	}
	if _, ok := ps.ByName("keep"); !ok {
		t.Error("the pre-import parameter was lost on rollback")
	}
}

func TestParametersCommandOpensDialog(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if s.ParametersOpen() {
		t.Fatal("dialog should start closed")
	}
	if err := s.Execute("Manage.Parameters"); err != nil {
		t.Fatalf("Execute Manage.Parameters: %v", err)
	}
	if !s.ParametersOpen() {
		t.Error("Manage.Parameters should open the dialog")
	}
	if _, ok := BuildRibbon(s).Tab("Manage"); !ok {
		t.Error("Manage tab should exist on the ribbon")
	}
	s.CloseParameters()
	if s.ParametersOpen() {
		t.Error("CloseParameters should close the dialog")
	}
}

// TestParameterRowsSplitAndFilter checks Model vs User splitting and the search filter.
func TestParameterRowsSplitAndFilter(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	ps := partParams(t, s)
	if _, err := ps.AddModelParameter("d0", "10 mm"); err != nil {
		t.Fatalf("AddModelParameter: %v", err)
	}
	_ = s.AddNumericUserParameter("width", "20 mm")
	_ = s.AddTextUserParameter("finish", "anodized")

	model, user := s.ParameterRows("")
	if _, ok := rowByName(model, "d0"); !ok {
		t.Error("d0 should be in the model table")
	}
	if len(user) != 2 {
		t.Errorf("user rows = %d, want 2", len(user))
	}
	if w, ok := rowByName(user, "width"); !ok || w.Value != "20 mm" {
		t.Errorf("width row = %+v ok=%v, want value 20 mm", w, ok)
	}

	// Search matches across name/equation/comment; "anod" only hits the text parameter.
	_, filtered := s.ParameterRows("anod")
	if len(filtered) != 1 || filtered[0].Name != "finish" {
		t.Errorf("filtered user rows = %+v, want only finish", filtered)
	}
}

// TestParameterEditFlow drives the dialog's edit verbs end to end.
func TestParameterEditFlow(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	if err := s.AddNumericUserParameter("len", "10 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	ps := partParams(t, s)
	id := mustParamID(t, ps, "len")

	if err := s.SetParameterEquation(id, "25 mm"); err != nil {
		t.Fatalf("SetParameterEquation: %v", err)
	}
	if err := s.SetParameterComment(id, "the length"); err != nil {
		t.Fatalf("SetParameterComment: %v", err)
	}
	_ = s.SetParameterKey(id, true)
	_ = s.SetParameterExport(id, true)

	_, user := s.ParameterRows("")
	row, _ := rowByName(user, "len")
	if row.Value != "25 mm" || row.Comment != "the length" || !row.IsKey || !row.Export {
		t.Errorf("edited row = %+v, want value 25 mm, comment set, key+export", row)
	}

	// Multi-value: make a list, select a value, then reject an off-list value.
	if err := s.SetParameterValueList(id, []string{"25 mm", "30 mm"}, false); err != nil {
		t.Fatalf("SetParameterValueList: %v", err)
	}
	if err := s.SetParameterEquation(id, "30 mm"); err != nil {
		t.Fatalf("select list value: %v", err)
	}
	if err := s.SetParameterEquation(id, "99 mm"); err == nil {
		t.Error("selecting an off-list value should fail when custom is not allowed")
	}

	// Groups: add, then the row reports membership.
	if err := s.AddParameterToGroup(id, "Frame"); err != nil {
		t.Fatalf("AddParameterToGroup: %v", err)
	}
	_, user = s.ParameterRows("")
	if row, _ := rowByName(user, "len"); row.Group != "Frame" {
		t.Errorf("group = %q, want Frame", row.Group)
	}

	// Copy to user, then delete the original.
	if err := s.CopyParameterToUser(id); err != nil {
		t.Fatalf("CopyParameterToUser: %v", err)
	}
	if _, ok := ps.ByName("len_copy"); !ok {
		t.Error("copy len_copy should exist")
	}
	if err := s.DeleteParameter(id); err != nil {
		t.Fatalf("DeleteParameter: %v", err)
	}
	if _, ok := ps.ByName("len"); ok {
		t.Error("len should be deleted")
	}
}

// TestAddBooleanAndTextParameters covers the non-numeric add verbs and their rows.
func TestAddBooleanAndTextParameters(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	if err := s.AddBooleanUserParameter("vented", true); err != nil {
		t.Fatalf("AddBooleanUserParameter: %v", err)
	}
	_, user := s.ParameterRows("")
	row, ok := rowByName(user, "vented")
	if !ok || row.ValueType != "boolean" || row.Equation != "true" {
		t.Errorf("boolean row = %+v ok=%v, want boolean/true", row, ok)
	}
	id := mustParamID(t, partParams(t, s), "vented")
	if err := s.SetParameterBool(id, false); err != nil {
		t.Fatalf("SetParameterBool: %v", err)
	}
	_, user = s.ParameterRows("")
	if row, _ := rowByName(user, "vented"); row.Equation != "false" {
		t.Errorf("after SetParameterBool(false) equation = %q, want false", row.Equation)
	}
}

func mustParamID(t *testing.T, ps *param.Parameters, name string) param.ID {
	t.Helper()
	p, ok := ps.ByName(name)
	if !ok {
		t.Fatalf("no parameter named %q", name)
	}
	return p.ID()
}
