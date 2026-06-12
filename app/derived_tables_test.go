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

// approx3 absorbs float noise at three decimals.
func approx3(got, want float64) bool {
	const eps = 1e-9
	return got-want < eps && want-got < eps
}
