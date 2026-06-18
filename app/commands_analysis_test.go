// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// TestPhysicalPropertiesNotice: the Physical Properties command computes the active part's mass
// properties and reports them in the status bar. (The numeric accuracy is covered by the
// model/analysis and router tests; this checks the command wiring.)
func TestPhysicalPropertiesNotice(t *testing.T) {
	s := newPartSession(t)
	if err := physicalProperties(s); err != nil {
		t.Fatalf("physicalProperties: %v", err)
	}
	notice := s.Notice()
	if !strings.Contains(notice, "Physical Properties") || !strings.Contains(notice, "volume") {
		t.Fatalf("notice = %q, want a Physical Properties summary", notice)
	}
}

// TestPhysicalPropertiesWithoutPart: the command errors when no part is active.
func TestPhysicalPropertiesWithoutPart(t *testing.T) {
	s := NewSession()
	if err := physicalProperties(s); err == nil {
		t.Error("physicalProperties with no active part = ok, want error")
	}
}

// TestModelHealthCommand: the Model Health command reports a clean part as all-OK, lists a
// suppressed feature, and errors without an active part.
func TestModelHealthCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	if err := modelHealth(s); err != nil {
		t.Fatalf("modelHealth: %v", err)
	}
	if !strings.Contains(s.Notice(), "all features OK") {
		t.Errorf("clean notice = %q, want all-OK", s.Notice())
	}

	activePartDef(t, s).Features().Item(0).SetSuppressed(true)
	if err := modelHealth(s); err != nil {
		t.Fatalf("modelHealth after suppress: %v", err)
	}
	if !strings.Contains(s.Notice(), "suppressed") {
		t.Errorf("suppressed notice = %q, want a suppressed feature listed", s.Notice())
	}

	if err := modelHealth(NewSession()); err == nil {
		t.Error("modelHealth with no active part = ok, want error")
	}
}
