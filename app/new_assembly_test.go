// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// TestNewAssemblyCreatesActiveAssemblyDocument: Session.NewAssembly mints a realized assembly
// document (assembly content installed, not the bare placeholder), makes it active, and names it
// "Assembly1" — the assembly counterpart of NewPart (#762).
func TestNewAssemblyCreatesActiveAssemblyDocument(t *testing.T) {
	t.Parallel()
	s := NewSession()
	d, err := s.NewAssembly()
	if err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	if s.ActiveDocument() != d {
		t.Error("a new assembly should become the active document")
	}
	if d.DocumentType() != doc.Assembly {
		t.Errorf("document type = %v, want Assembly", d.DocumentType())
	}
	if _, ok := d.Content().(*compdef.AssemblyComponentDefinition); !ok {
		t.Errorf("content = %T, want *AssemblyComponentDefinition (realized, not placeholder)", d.Content())
	}
	if d.DisplayName() != "Assembly1" {
		t.Errorf("name = %q, want Assembly1", d.DisplayName())
	}
}

// TestNewAssemblyNamesAreUnique: two New Assembly commands in a row never clash — the second is
// "Assembly2" — because uniqueDocumentName skips names already open.
func TestNewAssemblyNamesAreUnique(t *testing.T) {
	t.Parallel()
	s := NewSession()
	first, err := s.NewAssembly()
	if err != nil {
		t.Fatalf("first NewAssembly: %v", err)
	}
	second, err := s.NewAssembly()
	if err != nil {
		t.Fatalf("second NewAssembly: %v", err)
	}
	if first.DisplayName() == second.DisplayName() {
		t.Errorf("both assemblies named %q; the second must be unique", first.DisplayName())
	}
	if second.DisplayName() != "Assembly2" {
		t.Errorf("second assembly = %q, want Assembly2", second.DisplayName())
	}
}

// TestNewAssemblyAndNewPartShareCounterSpace: the Part and Assembly counters are independent —
// a New Part after a New Assembly is "Part1", not "Part2" (each prefix counts its own kind).
func TestNewAssemblyAndNewPartShareCounterSpace(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := s.NewAssembly(); err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	part, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if part.DisplayName() != "Part1" {
		t.Errorf("first part after an assembly = %q, want Part1", part.DisplayName())
	}
}

// TestGetStartedTabOffersNewAssembly: the ZeroDoc ribbon's Get Started ▸ Launch panel offers
// New Assembly alongside New Part, so a user can start an assembly from an empty session (#762).
func TestGetStartedTabOffersNewAssembly(t *testing.T) {
	t.Parallel()
	s := zeroDocSession(t)
	cmd, ok := s.Commands().ByID("GetStarted.NewAssembly")
	if !ok {
		t.Fatal("GetStarted.NewAssembly command is not registered")
	}
	if !cmd.IsEnabled(s) {
		t.Error("New Assembly should be enabled on the ZeroDoc ribbon")
	}
	r := BuildRibbon(s)
	tab, ok := r.Tab("Get Started")
	if !ok {
		t.Fatal("ZeroDoc ribbon has no Get Started tab")
	}
	panel, ok := tab.Panel("Launch")
	if !ok {
		t.Fatal("Get Started tab has no Launch panel")
	}
	if !panelHasCommand(panel, "GetStarted.NewAssembly") {
		t.Error("Launch panel does not offer New Assembly")
	}
}

// TestExecuteNewAssemblyOpensAssemblyRibbon: executing the command by id creates the assembly and
// switches the active ribbon to the Assemble environment (proves the command is wired end to end).
func TestExecuteNewAssemblyOpensAssemblyRibbon(t *testing.T) {
	t.Parallel()
	s := zeroDocSession(t)
	if err := s.Execute("GetStarted.NewAssembly"); err != nil {
		t.Fatalf("execute GetStarted.NewAssembly: %v", err)
	}
	if BuildRibbon(s).Key != AssemblyRibbon {
		t.Errorf("after New Assembly the ribbon is %q, want AssemblyRibbon", BuildRibbon(s).Key)
	}
}

// panelHasCommand reports whether a ribbon panel carries a button for the command id.
func panelHasCommand(p RibbonPanel, id string) bool {
	for _, b := range p.Buttons {
		if b.Command != nil && b.Command.ID() == id {
			return true
		}
	}
	return false
}
