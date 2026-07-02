// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// The group Session verbs added for the shared mutation seam (#1612, audit B1):
// each is the one path both the head UI and the wire router mutate through, so
// they are covered here once, on the verb, not per driver.

func TestParameterGroupVerbsRoundTrip(t *testing.T) {
	s := newSessionWithPart(t)
	g, err := s.AddParameterGroup("frame", "Frame", "com.example")
	if err != nil {
		t.Fatalf("AddParameterGroup: %v", err)
	}
	if g.InternalName() != "frame" || g.DisplayName() != "Frame" {
		t.Fatalf("created group = %+v, want frame/Frame", g)
	}
	if err := s.RenameParameterGroup("frame", "Chassis"); err != nil {
		t.Fatalf("RenameParameterGroup: %v", err)
	}
	if g.DisplayName() != "Chassis" {
		t.Errorf("display name = %q, want Chassis", g.DisplayName())
	}
	// The empty rename is refused by the aggregate through the verb.
	if err := s.RenameParameterGroup("frame", ""); err == nil {
		t.Error("RenameParameterGroup(\"\") must be refused")
	}
	if err := s.RenameParameterGroup("nope", "x"); err == nil {
		t.Error("renaming an unknown group must error")
	}
}

func TestDetachParameterFromGroupKeepsOtherMemberships(t *testing.T) {
	s := newSessionWithPart(t)
	ps := partParams(t, s)
	p, err := ps.AddUserParameter("len", "10 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	for _, key := range []string{"frame", "gears"} {
		if _, err := s.AddParameterGroup(key, "", ""); err != nil {
			t.Fatalf("AddParameterGroup(%s): %v", key, err)
		}
		if err := s.AddParameterToGroup(p.ID(), key); err != nil {
			t.Fatalf("AddParameterToGroup(%s): %v", key, err)
		}
	}
	if err := s.DetachParameterFromGroup(p.ID(), "frame"); err != nil {
		t.Fatalf("DetachParameterFromGroup: %v", err)
	}
	if got := ps.GroupsOf(p.ID()); len(got) != 1 || got[0] != "gears" {
		t.Errorf("memberships after detach = %v, want [gears] kept", got)
	}
}

func TestDeleteParameterGroupFlagControlsCascade(t *testing.T) {
	s := newSessionWithPart(t)
	ps := partParams(t, s)
	p, _ := ps.AddUserParameter("len", "10 mm")
	if _, err := s.AddParameterGroup("keepers", "", ""); err != nil {
		t.Fatalf("AddParameterGroup: %v", err)
	}
	if err := s.AddParameterToGroup(p.ID(), "keepers"); err != nil {
		t.Fatalf("AddParameterToGroup: %v", err)
	}
	// Plain delete keeps the member.
	if err := s.DeleteParameterGroup("keepers", false); err != nil {
		t.Fatalf("DeleteParameterGroup(false): %v", err)
	}
	if _, ok := ps.ByID(p.ID()); !ok {
		t.Fatal("plain group delete must keep the members")
	}
	// Cascade deletes it.
	if _, err := s.AddParameterGroup("doomed", "", ""); err != nil {
		t.Fatalf("AddParameterGroup: %v", err)
	}
	if err := s.AddParameterToGroup(p.ID(), "doomed"); err != nil {
		t.Fatalf("AddParameterToGroup: %v", err)
	}
	if err := s.DeleteParameterGroup("doomed", true); err != nil {
		t.Fatalf("DeleteParameterGroup(true): %v", err)
	}
	if _, ok := ps.ByID(p.ID()); ok {
		t.Error("cascade group delete must delete the members")
	}
}

// TestDeleteParameterRefusalSetsNotice: the UI surfaces the aggregate's
// refusal through the session notice (the B1 UI path, #1612).
func TestDeleteParameterRefusalSetsNotice(t *testing.T) {
	s := newSessionWithPart(t)
	ps := partParams(t, s)
	w, _ := ps.AddUserParameter("width", "10 mm")
	if _, err := ps.AddUserParameter("half", "width / 2"); err != nil {
		t.Fatalf("AddUserParameter(half): %v", err)
	}
	err := s.DeleteParameter(w.ID())
	if err == nil || !strings.Contains(err.Error(), "half") {
		t.Fatalf("DeleteParameter(in-use) = %v, want a refusal naming half", err)
	}
	if !strings.Contains(s.Notice(), "half") {
		t.Errorf("notice = %q, want the refusal surfaced to the UI", s.Notice())
	}
	if _, ok := ps.ByID(w.ID()); !ok {
		t.Error("a refused delete must keep the parameter")
	}
}
