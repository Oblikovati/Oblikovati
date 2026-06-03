// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestGroupLifecycle(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 mm")
	b, _ := ps.AddUserParameter("b", "2 mm")

	if err := ps.AddToGroup(a.ID(), "Bracket"); err != nil { // auto-creates the group
		t.Fatalf("AddToGroup: %v", err)
	}
	_ = ps.AddToGroup(b.ID(), "Bracket")
	if g, ok := ps.GroupOf(a.ID()); !ok || g != "Bracket" {
		t.Errorf("GroupOf(a) = %q,%v, want Bracket,true", g, ok)
	}
	if m := ps.GroupMembers("Bracket"); len(m) != 2 {
		t.Errorf("members = %v, want 2", m)
	}

	if err := ps.RenameGroup("Bracket", "Frame"); err != nil {
		t.Fatalf("RenameGroup: %v", err)
	}
	if g, _ := ps.GroupOf(a.ID()); g != "Frame" {
		t.Errorf("after rename GroupOf(a) = %q, want Frame", g)
	}

	if err := ps.RemoveFromGroup(b.ID()); err != nil {
		t.Fatalf("RemoveFromGroup: %v", err)
	}
	if _, ok := ps.GroupOf(b.ID()); ok {
		t.Error("b should be ungrouped after RemoveFromGroup")
	}
}

// TestDeleteGroupRemovesMembers checks deleting a group also deletes its parameters.
func TestDeleteGroupRemovesMembers(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 mm")
	keep, _ := ps.AddUserParameter("keep", "2 mm")
	_ = ps.AddToGroup(a.ID(), "G")

	if err := ps.DeleteGroup("G"); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, ok := ps.ByID(a.ID()); ok {
		t.Error("grouped parameter should be deleted with its group")
	}
	if _, ok := ps.ByID(keep.ID()); !ok {
		t.Error("ungrouped parameter should survive")
	}
	if len(ps.Groups()) != 0 {
		t.Errorf("groups = %v, want none", ps.Groups())
	}
}

func TestCopyToUser(t *testing.T) {
	ps := NewParameters()
	src, _ := ps.AddModelParameter("d0", "5 mm")
	src.Comment = "depth"
	src.IsKey = true

	cp, err := ps.CopyToUser(src.ID())
	if err != nil {
		t.Fatalf("CopyToUser: %v", err)
	}
	if cp.Name() != "d0_copy" {
		t.Errorf("copy name = %q, want d0_copy", cp.Name())
	}
	if cp.Kind() != UserParam {
		t.Errorf("copy kind = %v, want user", cp.Kind())
	}
	if !approxScalar(cp.Value().Value, 0.5) || cp.Comment != "depth" || !cp.IsKey {
		t.Errorf("copy did not carry value/comment/key: %+v", cp)
	}
	// A second copy gets a distinct name and is independent of the source.
	cp2, _ := ps.CopyToUser(src.ID())
	if cp2.Name() != "d0_copy2" {
		t.Errorf("second copy name = %q, want d0_copy2", cp2.Name())
	}
	_ = cp.SetExpression("9 mm")
	if approxScalar(src.Value().Value, 0.9) {
		t.Error("editing the copy must not change the source")
	}
}
