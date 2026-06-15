// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "testing"

// TestContactSolverMembership checks contact-set creation, idempotent membership, the
// shared-set contact query, removal, and delete (M12-F05).
func TestContactSolverMembership(t *testing.T) {
	s := NewContactSolver()
	if s.Enabled() {
		t.Error("a new contact solver should be disabled")
	}
	s.SetEnabled(true)
	if !s.Enabled() {
		t.Error("SetEnabled(true) did not enable the solver")
	}

	cs := s.Create("group")
	if err := s.AddMember(cs.ID(), 1); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	_ = s.AddMember(cs.ID(), 2)
	_ = s.AddMember(cs.ID(), 2) // idempotent
	if cs.MemberCount() != 2 {
		t.Errorf("member count = %d, want 2", cs.MemberCount())
	}
	if !s.Contacts(1, 2) {
		t.Error("occurrences 1 and 2 share a set — they should contact")
	}
	if s.Contacts(1, 3) {
		t.Error("occurrence 3 is in no set — it should not contact")
	}
	if p := s.PartnersOf(1); len(p) != 1 || p[0] != 2 {
		t.Errorf("PartnersOf(1) = %v, want [2]", p)
	}

	_ = s.RemoveMember(cs.ID(), 2)
	if s.Contacts(1, 2) {
		t.Error("after removing 2, the pair should no longer contact")
	}
	if !s.Delete(cs.ID()) || s.Count() != 0 {
		t.Errorf("delete failed: count = %d", s.Count())
	}
	if err := s.AddMember(404, 1); err == nil {
		t.Error("AddMember to an unknown set should error")
	}
}
