// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"testing"
)

// parametersButton finds the Manage ▸ Parameters button on a built ribbon, returning it and
// whether it was found.
func parametersButton(t *testing.T, s *Session) (RibbonButton, bool) {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab("Manage")
	if !ok {
		return RibbonButton{}, false
	}
	panel, ok := tab.Panel("Parameters")
	if !ok {
		return RibbonButton{}, false
	}
	for _, b := range panel.Buttons {
		if b.Command != nil && b.Command.ID() == "Manage.Parameters" {
			return b, true
		}
	}
	return RibbonButton{}, false
}

// TestParametersButtonOnAssemblyRibbon checks the Manage ▸ Parameters button now appears AND
// is enabled for an active assembly — the F04 ribbon gate (#1560). Before F04 the Manage tab
// was part-only, so an assembly showed no Parameters button.
func TestParametersButtonOnAssemblyRibbon(t *testing.T) {
	t.Parallel()
	b, ok := parametersButton(t, assemblySession(t))
	if !ok {
		t.Fatal("an active assembly should show the Manage ▸ Parameters button")
	}
	if !b.Enabled {
		t.Error("the Parameters button should be enabled for an active assembly")
	}
}

// TestParametersButtonStillOnPartRibbon guards against regressing the part: the Parameters
// button must still appear and be enabled for a part.
func TestParametersButtonStillOnPartRibbon(t *testing.T) {
	t.Parallel()
	b, ok := parametersButton(t, registeredSession(t))
	if !ok || !b.Enabled {
		t.Errorf("a part should still show an enabled Parameters button (found=%v enabled=%v)", ok, b.Enabled)
	}
}

// TestHasActiveParameterHolderGate checks the enable predicate: true for a part or an
// assembly, false with no parameter-holding document active.
func TestHasActiveParameterHolderGate(t *testing.T) {
	t.Parallel()
	if !hasActiveParameterHolder(assemblySession(t)) {
		t.Error("hasActiveParameterHolder should be true with an assembly active")
	}
	if !hasActiveParameterHolder(registeredSession(t)) {
		t.Error("hasActiveParameterHolder should be true with a part active")
	}
	if hasActiveParameterHolder(NewSession()) {
		t.Error("hasActiveParameterHolder should be false with no active document")
	}
}

// TestDerivedTableViewOnAssembly checks the dialog presentation accessors for an assembly
// deriving from a part: the part is listed as a linkable source, and after a link the derived
// table row reports the source and linked subset (M39-F04, #1560).
func TestDerivedTableViewOnAssembly(t *testing.T) {
	t.Parallel()
	s := NewSession()
	gears, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if err := s.AddNumericUserParameter("module", "2 mm"); err != nil {
		t.Fatalf("add module: %v", err)
	}
	asm, err := s.NewAssembly()
	if err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asm); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}

	sources := s.LinkableSourceDocuments()
	if len(sources) != 1 || sources[0].FullName != gears.FullDocumentName() {
		t.Fatalf("linkable sources = %+v, want [gears]", sources)
	}
	if !slices.Equal(sources[0].Parameters, []string{"module"}) {
		t.Errorf("source parameters = %v, want [module]", sources[0].Parameters)
	}

	if _, err := s.AddDerivedParameterTable(gears.FullDocumentName(), []string{"module"}); err != nil {
		t.Fatalf("AddDerivedParameterTable: %v", err)
	}
	rows := s.DerivedTableRows()
	if len(rows) != 1 {
		t.Fatalf("derived table rows = %d, want 1", len(rows))
	}
	if rows[0].SourceDocument != gears.FullDocumentName() || !slices.Equal(rows[0].Linked, []string{"module"}) {
		t.Errorf("row = %+v, want source gears linked [module]", rows[0])
	}
	if rows[0].Health != "" {
		t.Errorf("healthy link row should have empty health, got %q", rows[0].Health)
	}
}

// TestLinkableSourceExcludesActiveAndNonHolders checks the picker omits the active document
// (no self-derivation) and documents with no linkable parameters.
func TestLinkableSourceExcludesActiveAndNonHolders(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := s.NewAssembly(); err != nil { // active, empty → not its own source
		t.Fatalf("NewAssembly: %v", err)
	}
	if got := s.LinkableSourceDocuments(); len(got) != 0 {
		t.Errorf("a lone empty assembly should have no linkable sources, got %+v", got)
	}
}
