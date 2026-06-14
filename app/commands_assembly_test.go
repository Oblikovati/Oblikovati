// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// assemblySession returns a session with an active assembly document and the standard
// ribbon commands wired — so BuildRibbon resolves the AssemblyRibbon.
func assemblySession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	d, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	return s
}

// TestAssembleRibbonTabAndPanels checks an active assembly shows the Assemble tab with its
// Component/Pattern/BOM panels (the M11 scaffolding, #761).
func TestAssembleRibbonTabAndPanels(t *testing.T) {
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble ribbon tab")
	}
	for _, panel := range []string{"Component", "Pattern", "BOM"} {
		if _, ok := tab.Panel(panel); !ok {
			t.Errorf("Assemble tab has no %s panel", panel)
		}
	}
}

// TestAssembleTabAbsentForPart checks the Assemble tab is contextual to an assembly: a part
// document must not show it.
func TestAssembleTabAbsentForPart(t *testing.T) {
	if _, ok := BuildRibbon(registeredSession(t)).Tab("Assemble"); ok {
		t.Error("a part ribbon should not show the Assemble tab")
	}
}

// TestActiveAssemblyGate checks the active-assembly enable predicate and accessor.
func TestActiveAssemblyGate(t *testing.T) {
	if !hasActiveAssembly(assemblySession(t)) {
		t.Error("hasActiveAssembly should be true with an assembly active")
	}
	if hasActiveAssembly(registeredSession(t)) {
		t.Error("hasActiveAssembly should be false with a part active")
	}
	if _, err := activeAssembly(assemblySession(t)); err != nil {
		t.Errorf("activeAssembly should resolve the active assembly: %v", err)
	}
}
