// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// partAndSourceAssembly returns a session whose active document is a fresh empty PART (the derive
// target), with an open assembly (one box component placed) as the derive source (#767).
func partAndSourceAssembly(t *testing.T) (*Session, *doc.Document) {
	t.Helper()
	s := extrudedBox(t, 2, 4) // active part = a box component with a real body
	box := s.ActiveDocument()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := asm.PlaceComponentFromFile(asmDoc, box, "box:1", math.Identity4()); err != nil {
		t.Fatalf("place box: %v", err)
	}
	if _, err := s.NewPart(); err != nil { // active becomes a fresh empty part to derive into
		t.Fatalf("NewPart: %v", err)
	}
	return s, asmDoc
}

// activeTargetPart returns the active part definition (the derive target).
func activeTargetPart(t *testing.T, s *Session) *compdef.PartComponentDefinition {
	t.Helper()
	part, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if !ok {
		t.Fatal("active document is not the target part")
	}
	return part
}

// TestDeriveAssemblyCreatesBaseBody: Derive merges the source assembly into the active part as a
// derivedAssembly feature producing a base body (#767).
func TestDeriveAssemblyCreatesBaseBody(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	target := activeTargetPart(t, s)

	f, err := s.DeriveAssembly(source)
	if err != nil {
		t.Fatalf("DeriveAssembly: %v", err)
	}
	if f.Kind() != "derivedAssembly" {
		t.Errorf("feature kind = %q, want derivedAssembly", f.Kind())
	}
	if target.SurfaceBodies().Count() == 0 {
		t.Error("derive should produce a base body in the target part")
	}
}

// TestShrinkwrapCreatesBaseBody: Shrinkwrap merges the source into the part as a simplified base
// body (a single whole-assembly bounding box here).
func TestShrinkwrapCreatesBaseBody(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	target := activeTargetPart(t, s)

	def := feature.ShrinkwrapDefinition{EnvelopeStyle: feature.EnvelopeWhole}
	f, err := s.ShrinkwrapAssembly(source, def)
	if err != nil {
		t.Fatalf("ShrinkwrapAssembly: %v", err)
	}
	if f.Kind() != "shrinkwrap" {
		t.Errorf("feature kind = %q, want shrinkwrap", f.Kind())
	}
	if target.SurfaceBodies().Count() == 0 {
		t.Error("shrinkwrap should produce a base body in the target part")
	}
}

// TestDeriveCommandsGatedOnOpenAssembly: the Simplify commands are enabled only when a part is
// active AND an assembly is open to derive.
func TestDeriveCommandsGatedOnOpenAssembly(t *testing.T) {
	s := registeredSession(t) // a part is active, no assembly open
	derive, _ := s.Commands().ByID("Manage.Derive")
	if derive == nil {
		t.Fatal("Manage.Derive command not registered")
	}
	if derive.IsEnabled(s) {
		t.Error("Derive should be disabled with no assembly open")
	}
	// Open an assembly, then return focus to the part — now there is a source to derive.
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	part := s.Workspace().Documents()[0]
	if err := s.Workspace().SetActiveDocument(part); err != nil {
		t.Fatalf("activate part: %v", err)
	}
	_ = asmDoc
	if !derive.IsEnabled(s) {
		t.Error("Derive should be enabled with a part active and an assembly open")
	}
}

// TestDerivedFeatureMenuHasUpdateAndBreakLink: a derived feature's browser menu carries Update and
// Break Link; a plain feature does not.
func TestDerivedFeatureMenuHasUpdateAndBreakLink(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	f, err := s.DeriveAssembly(source)
	if err != nil {
		t.Fatalf("DeriveAssembly: %v", err)
	}
	labels := menuLabels(BrowserMenu(s, BrowserNode{Kind: "feature", Select: FeatureHandle{Feature: f}}))
	joined := strings.Join(labels, "|")
	if !strings.Contains(joined, "Update") || !strings.Contains(joined, "Break Link") {
		t.Errorf("derive feature menu = %q, want Update + Break Link", joined)
	}
}

// TestDeriveOutOfDateBadgeAndUpdate: a source edited after the derive marks the node out of date
// (badge in the label, Update enabled); Update re-syncs and clears it (#767).
func TestDeriveOutOfDateBadgeAndUpdate(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	stampRevision(source, "rev-A")
	f, err := s.DeriveAssembly(source)
	if err != nil {
		t.Fatalf("DeriveAssembly: %v", err)
	}
	if _, outOfDate, _ := s.DerivedFeatureStatus(f); outOfDate {
		t.Fatal("a fresh in-session derive should not be out of date")
	}

	// The source is edited (new revision) and the derive rebinds to it — the reopen-with-changed-
	// source case the drive state models.
	stampRevision(source, "rev-B")
	asm := source.Content().(*compdef.AssemblyComponentDefinition)
	f.Definition().(*feature.DerivedAssemblyComponent).BindSource(asm, "rev-B")

	if _, outOfDate, _ := s.DerivedFeatureStatus(f); !outOfDate {
		t.Fatal("editing the source should mark the derive out of date")
	}
	if lbl := featureLabel(f); !strings.Contains(lbl, "out of date") {
		t.Errorf("out-of-date label = %q, want an (out of date) badge", lbl)
	}
	if err := s.UpdateDerivedFeature(f); err != nil {
		t.Fatalf("UpdateDerivedFeature: %v", err)
	}
	if _, outOfDate, _ := s.DerivedFeatureStatus(f); outOfDate {
		t.Error("Update should clear the out-of-date state")
	}
}

// TestBreakDerivedLinkFreezes: Break Link severs the source link so the part keeps its geometry.
func TestBreakDerivedLinkFreezes(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	f, err := s.DeriveAssembly(source)
	if err != nil {
		t.Fatalf("DeriveAssembly: %v", err)
	}
	if _, _, linked := s.DerivedFeatureStatus(f); !linked {
		t.Fatal("a fresh derive should be linked")
	}
	if err := s.BreakDerivedLink(f); err != nil {
		t.Fatalf("BreakDerivedLink: %v", err)
	}
	if _, _, linked := s.DerivedFeatureStatus(f); linked {
		t.Error("Break Link should leave the feature unlinked")
	}
}

// TestDeriveIsUndoable: deriving an assembly is one undo step — undo removes the derived feature.
func TestDeriveIsUndoable(t *testing.T) {
	s, source := partAndSourceAssembly(t)
	target := activeTargetPart(t, s)
	trackFromHere(s)

	if _, err := s.DeriveAssembly(source); err != nil {
		t.Fatalf("DeriveAssembly: %v", err)
	}
	if target.Features().Count() != 1 {
		t.Fatalf("after derive: feature count = %d, want 1", target.Features().Count())
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if target.Features().Count() != 0 {
		t.Errorf("undo should remove the derived feature: count = %d, want 0", target.Features().Count())
	}
}

// stampRevision sets a document's model revision id, so a test can drive the derive drive-state.
func stampRevision(d *doc.Document, rev string) {
	id := d.FileIdentity()
	id.DatabaseRevisionID = rev
	d.SetFileIdentity(id)
}
