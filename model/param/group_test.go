// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"slices"
	"testing"
)

func TestGroupRecordLifecycle(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 mm")
	b, _ := ps.AddUserParameter("b", "2 mm")

	g, err := ps.AddGroup("com.example:bracket", "Bracket", "com.example")
	if err != nil {
		t.Fatalf("AddGroup: %v", err)
	}
	if g.InternalName() != "com.example:bracket" || g.DisplayName != "Bracket" || g.ClientID != "com.example" {
		t.Fatalf("group record = %+v, want key/display/client kept", g)
	}
	if _, err := ps.AddGroup("com.example:bracket", "", ""); err == nil {
		t.Error("duplicate internal name must be rejected")
	}
	if _, err := ps.AddGroup("", "x", ""); err == nil {
		t.Error("empty internal name must be rejected")
	}
	// An empty display name defaults to the internal name.
	plain, _ := ps.AddGroup("Frame", "", "")
	if plain.DisplayName != "Frame" {
		t.Errorf("display name = %q, want the internal-name default", plain.DisplayName)
	}

	if err := ps.AddToGroup(a.ID(), "com.example:bracket"); err != nil {
		t.Fatalf("AddToGroup: %v", err)
	}
	if err := ps.AddToGroup(a.ID(), "Frame"); err != nil {
		t.Fatalf("AddToGroup(second group): %v", err)
	}
	_ = ps.AddToGroup(b.ID(), "Frame")
	if err := ps.AddToGroup(a.ID(), "nope"); err == nil {
		t.Error("membership in an unknown group must be rejected")
	}

	// Multi-membership: a sits in both groups, in creation order.
	if got := ps.GroupsOf(a.ID()); !slices.Equal(got, []string{"com.example:bracket", "Frame"}) {
		t.Errorf("GroupsOf(a) = %v, want both groups", got)
	}
	if m := ps.GroupMembers("Frame"); len(m) != 2 {
		t.Errorf("Frame members = %v, want a and b", m)
	}

	// Leaving one group touches neither the parameter nor its other memberships.
	if err := ps.RemoveFromGroup(a.ID(), "Frame"); err != nil {
		t.Fatalf("RemoveFromGroup: %v", err)
	}
	if got := ps.GroupsOf(a.ID()); !slices.Equal(got, []string{"com.example:bracket"}) {
		t.Errorf("GroupsOf(a) after leave = %v, want the bracket group only", got)
	}
	if _, ok := ps.ByID(a.ID()); !ok {
		t.Error("leaving a group must not delete the parameter")
	}
}

func TestGroupDisplayNameEditsKeepKey(t *testing.T) {
	ps := NewParameters()
	g, _ := ps.AddGroup("ratios", "Ratios", "")
	g.DisplayName = "Gear Ratios"
	got, ok := ps.GroupByKey("ratios")
	if !ok || got.DisplayName != "Gear Ratios" {
		t.Errorf("group after display rename = %+v, want addressable by its key with the new display", got)
	}
}

// TestDeleteGroupCascadeIsOptIn checks the two delete flavors: plain delete
// keeps the members, deleteParameters cascades onto them.
func TestDeleteGroupCascadeIsOptIn(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 mm")
	keep, _ := ps.AddUserParameter("keep", "2 mm")
	_, _ = ps.AddGroup("G", "", "")
	_ = ps.AddToGroup(a.ID(), "G")

	if err := ps.DeleteGroup("G", false); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, ok := ps.ByID(a.ID()); !ok {
		t.Error("plain group delete must keep the member parameters")
	}
	if len(ps.GroupsOf(a.ID())) != 0 {
		t.Error("membership must not survive its group")
	}

	_, _ = ps.AddGroup("G2", "", "")
	_ = ps.AddToGroup(a.ID(), "G2")
	if err := ps.DeleteGroup("G2", true); err != nil {
		t.Fatalf("DeleteGroup(cascade): %v", err)
	}
	if _, ok := ps.ByID(a.ID()); ok {
		t.Error("cascade delete must delete the member parameters")
	}
	if _, ok := ps.ByID(keep.ID()); !ok {
		t.Error("a parameter outside the group must survive the cascade")
	}
	if len(ps.Groups()) != 0 {
		t.Errorf("groups = %v, want none", ps.Groups())
	}
}

// TestDeleteParameterDetachesFromAllGroups checks the M02-F05 invariant from
// the other side: the parameter goes, every membership goes with it.
func TestDeleteParameterDetachesFromAllGroups(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "1 mm")
	_, _ = ps.AddGroup("G1", "", "")
	_, _ = ps.AddGroup("G2", "", "")
	_ = ps.AddToGroup(a.ID(), "G1")
	_ = ps.AddToGroup(a.ID(), "G2")

	if err := ps.Delete(a.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(ps.GroupMembers("G1"))+len(ps.GroupMembers("G2")) != 0 {
		t.Error("a deleted parameter must leave every group")
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
