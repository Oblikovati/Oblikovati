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
