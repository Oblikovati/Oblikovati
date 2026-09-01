// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// TestContactPanelAndActions: the Contact panel exposes its commands, creating a contact set
// from the selection groups them, the solver toggles, and Analyze Interference reports the
// overlap (M12-F05).
func TestContactPanelAndActions(t *testing.T) {
	t.Parallel()
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("assembly should show the Assemble tab")
	}
	panel, ok := tab.Panel("Contact")
	if !ok {
		t.Fatal("Assemble tab has no Contact panel")
	}
	for _, name := range []string{"Contact Set", "Enable Contact", "Interference"} {
		if !hasButton(panel, name) {
			t.Errorf("Contact panel missing %q", name)
		}
	}

	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a, b := asm.Occurrences().Item(0), asm.Occurrences().Item(1)
	s.Selection().Add(OccurrenceHandle{Occurrence: a})
	s.Selection().Add(OccurrenceHandle{Occurrence: b})

	if err := s.CreateContactSet(); err != nil {
		t.Fatalf("CreateContactSet: %v", err)
	}
	if asm.ContactSolver().Count() != 1 || asm.ContactSolver().All()[0].MemberCount() != 2 {
		t.Errorf("contact set = %d sets / %d members, want 1/2", asm.ContactSolver().Count(), len(asm.ContactSolver().All()))
	}
	if folder := findBrowserNode(BuildBrowser(s), "contactSets", "Contact Sets"); folder == nil || len(folder.Children) != 1 {
		t.Errorf("Contact Sets browser folder = %v, want one row", folder)
	}

	if err := s.ToggleContactSolver(); err != nil || !asm.ContactSolver().Enabled() {
		t.Errorf("ToggleContactSolver: err=%v enabled=%v", err, asm.ContactSolver().Enabled())
	}

	// Overlap the two widgets, then Analyze Interference reports a positive volume.
	b.SetTransform(math.Identity4()) // coincident with a → full overlap
	if err := s.AnalyzeInterference(); err != nil {
		t.Fatalf("AnalyzeInterference: %v", err)
	}
	if !strings.Contains(s.Notice(), "interference") {
		t.Errorf("interference notice = %q, want an interference report", s.Notice())
	}
}
