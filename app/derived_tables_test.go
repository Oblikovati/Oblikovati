// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// derivedSession opens two part documents: gears.obk with a numeric user
// parameter "module", and hub.obk (active) ready to derive from it.
func derivedSession(t *testing.T) (*Session, *doc.Document, *doc.Document) {
	t.Helper()
	s := NewSession()
	gears, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart(gears): %v", err)
	}
	if err := s.AddNumericUserParameter("module", "2 mm"); err != nil {
		t.Fatalf("add module: %v", err)
	}
	hub, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart(hub): %v", err)
	}
	if s.ActiveDocument() != hub {
		t.Fatal("hub must be the active document")
	}
	return s, gears, hub
}

// docPartOf unwraps a document's part definition.
func docPartOf(t *testing.T, d *doc.Document) *compdef.PartComponentDefinition {
	t.Helper()
	def, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		t.Fatalf("document %q holds no part", d.FullDocumentName())
	}
	return def
}

func TestDeriveParametersAcrossDocuments(t *testing.T) {
	s, gears, hub := derivedSession(t)

	table, err := s.AddDerivedParameterTable(gears.FullDocumentName(), []string{"module"})
	if err != nil {
		t.Fatalf("AddDerivedParameterTable: %v", err)
	}
	p, ok := docPartOf(t, hub).Parameters().ByName("module")
	if !ok || !approx3(p.Value().Value, 0.2) {
		t.Fatalf("derived module = %+v, want 0.2 (2 mm in db units)", p)
	}

	// The document reference is in the workspace graph.
	if refs := hub.ReferencedDocuments(); len(refs) != 1 || refs[0] != gears {
		t.Errorf("hub references = %v, want [gears]", refs)
	}

	// Editing the source pushes the new value into the deriving document
	// through the recordEdit seam — no manual sync.
	if err := s.Workspace().SetActiveDocument(gears); err != nil {
		t.Fatalf("activate gears: %v", err)
	}
	srcParam, _ := docPartOf(t, gears).Parameters().ByName("module")
	if err := s.SetParameterEquation(srcParam.ID(), "3 mm"); err != nil {
		t.Fatalf("edit source module: %v", err)
	}
	if !approx3(p.Value().Value, 0.3) {
		t.Errorf("derived module after source edit = %v, want 0.3", p.Value().Value)
	}

	// Deleting the source parameter sickens the derived one on the next edit.
	if err := s.DeleteParameter(srcParam.ID()); err != nil {
		t.Fatalf("delete source module: %v", err)
	}
	if p.Health().OK() {
		t.Error("derived parameter must go sick when its source is deleted")
	}
	if table.Health().OK() {
		t.Errorf("table health = %+v, want failed", table.Health())
	}
}

func TestDeriveRejectsSelfAndUnknownSource(t *testing.T) {
	s, _, hub := derivedSession(t)
	if _, err := s.AddDerivedParameterTable(hub.FullDocumentName(), nil); err == nil || !strings.Contains(err.Error(), "itself") {
		t.Errorf("self-derive err = %v, want a self-link rejection", err)
	}
	if _, err := s.AddDerivedParameterTable("missing.obk", nil); err == nil {
		t.Error("deriving from an unopened document must be rejected")
	}
}

func TestDerivedTableLifecycleOnSession(t *testing.T) {
	s, gears, hub := derivedSession(t)
	table, err := s.AddDerivedParameterTable(gears.FullDocumentName(), nil)
	if err != nil {
		t.Fatalf("AddDerivedParameterTable: %v", err)
	}

	if err := s.SetDerivedTableLinked(table.ID(), []string{"module"}); err != nil {
		t.Fatalf("SetDerivedTableLinked: %v", err)
	}
	if _, ok := docPartOf(t, hub).Parameters().ByName("module"); !ok {
		t.Error("relinking must produce the derived parameter")
	}

	if err := s.DeleteDerivedParameterTable(table.ID()); err != nil {
		t.Fatalf("DeleteDerivedParameterTable: %v", err)
	}
	if _, ok := docPartOf(t, hub).Parameters().ByName("module"); ok {
		t.Error("deleting the table must delete its derived parameters")
	}
}

// docAssemblyOf unwraps a document's assembly definition.
func docAssemblyOf(t *testing.T, d *doc.Document) *compdef.AssemblyComponentDefinition {
	t.Helper()
	def, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		t.Fatalf("document %q holds no assembly", d.FullDocumentName())
	}
	return def
}

// TestAssemblyDerivesParametersFromPart is the M39-F02 forward direction: an ASSEMBLY (not a
// part) derives a numeric user parameter from a part document, and a later source edit
// resyncs into the assembly through the now-generalized seam (#1558). Before F02 the active
// target and the resync target were both cast to *PartComponentDefinition, so an assembly
// could neither derive nor receive a resync.
func TestAssemblyDerivesParametersFromPart(t *testing.T) {
	s := NewSession()
	gears, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart(gears): %v", err)
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

	if _, err := s.AddDerivedParameterTable(gears.FullDocumentName(), []string{"module"}); err != nil {
		t.Fatalf("AddDerivedParameterTable into assembly: %v", err)
	}
	p, ok := docAssemblyOf(t, asm).Parameters().ByName("module")
	if !ok || !approx3(p.Value().Value, 0.2) {
		t.Fatalf("derived module on assembly = %+v, want 0.2 (2 mm in db units)", p)
	}

	// Edit the source part; the resync must flow into the deriving assembly.
	if err := s.Workspace().SetActiveDocument(gears); err != nil {
		t.Fatalf("activate gears: %v", err)
	}
	src, _ := docPartOf(t, gears).Parameters().ByName("module")
	if err := s.SetParameterEquation(src.ID(), "3 mm"); err != nil {
		t.Fatalf("edit source module: %v", err)
	}
	if !approx3(p.Value().Value, 0.3) {
		t.Errorf("assembly derived module after source edit = %v, want 0.3 (resync to assembly)", p.Value().Value)
	}
}

// TestPartDerivesParametersFromAssembly is the M39-F02 reverse direction: a PART derives from
// an ASSEMBLY's numeric user parameter. The source-document resolution (LinkableSourceParameters)
// must treat an assembly as a valid parameter source (#1558).
func TestPartDerivesParametersFromAssembly(t *testing.T) {
	s := NewSession()
	asm, err := s.NewAssembly()
	if err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	// The session parameter-edit verbs are still part-only (F03), so seed the assembly source
	// parameter directly on its definition.
	if _, err := docAssemblyOf(t, asm).Parameters().AddUserParameter("spacing", "5 mm"); err != nil {
		t.Fatalf("seed assembly spacing: %v", err)
	}
	hub, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart(hub): %v", err)
	}
	if s.ActiveDocument() != hub {
		t.Fatalf("hub must be the active document")
	}

	if _, err := s.AddDerivedParameterTable(asm.FullDocumentName(), []string{"spacing"}); err != nil {
		t.Fatalf("AddDerivedParameterTable from assembly: %v", err)
	}
	p, ok := docPartOf(t, hub).Parameters().ByName("spacing")
	if !ok || !approx3(p.Value().Value, 0.5) {
		t.Fatalf("part derived spacing = %+v, want 0.5 (5 mm in db units)", p)
	}
}

// approx3 absorbs float noise at three decimals.
func approx3(got, want float64) bool {
	const eps = 1e-9
	return got-want < eps && want-got < eps
}

// TestLinkAutoExportsSourceParameter checks the Add2 auto-export-on-link behavior (M39-F05,
// #1561): linking a source parameter that was not exported marks it exported on the source
// document, so the source advertises it.
func TestLinkAutoExportsSourceParameter(t *testing.T) {
	s, gears, _ := derivedSession(t)
	src, ok := docPartOf(t, gears).Parameters().ByName("module")
	if !ok || src.ExposedAsProperty {
		t.Fatalf("precondition: source module should exist and start unexported (ok=%v)", ok)
	}
	if _, err := s.AddDerivedParameterTable(gears.FullDocumentName(), []string{"module"}); err != nil {
		t.Fatalf("AddDerivedParameterTable: %v", err)
	}
	if !src.ExposedAsProperty {
		t.Error("linking should auto-export the source parameter (Add2 semantics)")
	}
}
