// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestParametersAddLookupAndCount(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 cm")
	b, _ := ps.AddUserParameter("b", "2 cm")
	if ps.Count() != 2 {
		t.Fatalf("Count = %d, want 2", ps.Count())
	}
	if got, ok := ps.ByName("a"); !ok || got.ID() != a.ID() {
		t.Error("ByName(a) did not return the right parameter")
	}
	if got, ok := ps.ByID(b.ID()); !ok || got.Name() != "b" {
		t.Error("ByID did not return the right parameter")
	}
	if _, ok := ps.ByName("missing"); ok {
		t.Error("ByName(missing) should report not found")
	}
}

func TestDuplicateNameRejected(t *testing.T) {
	ps := NewParameters()
	if _, err := ps.AddUserParameter("a", "1 cm"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if _, err := ps.AddUserParameter("a", "2 cm"); err == nil {
		t.Error("duplicate name should be rejected")
	}
	if ps.Count() != 1 {
		t.Errorf("Count = %d, want 1 after rejected duplicate", ps.Count())
	}
}

func TestFailedAddDoesNotRetainParameter(t *testing.T) {
	ps := NewParameters()
	if _, err := ps.AddUserParameter("bad", "1 +"); err == nil {
		t.Fatal("malformed expression should fail the add")
	}
	if ps.Count() != 0 {
		t.Errorf("Count = %d, want 0 (half-constructed parameter removed)", ps.Count())
	}
	if _, ok := ps.ByName("bad"); ok {
		t.Error("failed add left a name registered")
	}
}

func TestRenamePreservesIdentity(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 cm")
	if err := ps.Rename(a.ID(), "alpha"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, ok := ps.ByName("a"); ok {
		t.Error("old name should no longer resolve")
	}
	if got, ok := ps.ByName("alpha"); !ok || got.ID() != a.ID() {
		t.Error("new name should resolve to the same id")
	}
}

func TestRenameClashRejected(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 cm")
	if _, err := ps.AddUserParameter("b", "2 cm"); err != nil {
		t.Fatalf("add b: %v", err)
	}
	if err := ps.Rename(a.ID(), "b"); err == nil {
		t.Error("renaming onto an existing name should error")
	}
}

func TestDelete(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 cm")
	if err := ps.Delete(a.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ps.Count() != 0 {
		t.Errorf("Count = %d, want 0", ps.Count())
	}
	if err := ps.Delete(a.ID()); err == nil {
		t.Error("deleting a missing id should error")
	}
}

func TestParametersImplementsScope(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "5 cm")
	var scope Scope = ps
	if q, ok := scope.ValueOf(a.ID()); !ok || q != (Quantity{5, Length}) {
		t.Errorf("ValueOf = %v, %v; want {5 length}", q, ok)
	}
}
