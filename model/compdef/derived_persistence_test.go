// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/persistence"
)

// deriveSourceIntoPart creates a part document deriving sourceDoc — capturing the source's
// identity link and recording the part→source reference exactly as the derive router does —
// and returns the part document (now active).
func deriveSourceIntoPart(t *testing.T, ws *doc.Workspace, dir, name string, sourceDoc *doc.Document) *doc.Document {
	t.Helper()
	partDoc, err := compdef.AddPart(ws, filepath.Join(dir, name), true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", name, err)
	}
	part := partDoc.Content().(*compdef.PartComponentDefinition)
	source := sourceDoc.Content().(feature.AssemblyBodySource)
	id := sourceDoc.FileIdentity()
	link := feature.DeriveSourceLink{
		Document:           sourceDoc.FullDocumentName(),
		InternalName:       id.InternalName,
		DatabaseRevisionID: id.DatabaseRevisionID,
	}
	feature.NewDerivedAssemblyComponents(part.Features()).AddDerived(source, link)
	partDoc.OpenReference(sourceDoc.FullDocumentName())
	return partDoc
}

// openPart opens name in a fresh store-backed workspace (forcing the on-disk load and
// reference-resolution path) and returns its part definition.
func openPart(t *testing.T, store *persistence.PackageStore, name string) *compdef.PartComponentDefinition {
	t.Helper()
	reopened, err := doc.NewWorkspace(store).Open(name, true)
	if err != nil {
		t.Fatalf("Open part %q: %v", name, err)
	}
	return reopened.Content().(*compdef.PartComponentDefinition)
}

// derivedComponentOf returns the single derived-assembly component of a part.
func derivedComponentOf(t *testing.T, part *compdef.PartComponentDefinition) *feature.DerivedAssemblyComponent {
	t.Helper()
	for i := 0; i < part.Features().Count(); i++ {
		if d, ok := part.Features().Item(i).Definition().(*feature.DerivedAssemblyComponent); ok {
			return d
		}
	}
	t.Fatal("part has no derived-assembly component")
	return nil
}

// savedDerivedPart builds and saves a source assembly and a part that derives it (both in
// the same store), returning the store, workspace, dir, and the two saved documents — the
// fixture the derived-part reopen tests start from.
func savedDerivedPart(t *testing.T) (store *persistence.PackageStore, ws *doc.Workspace, dir string, srcDoc, partDoc *doc.Document) {
	t.Helper()
	store, ws, dir = assemblyWorkspace(t)
	srcDoc, _ = newAssembly(t, ws, dir, "src.obk")
	if err := ws.Save(srcDoc); err != nil {
		t.Fatalf("Save source: %v", err)
	}
	partDoc = deriveSourceIntoPart(t, ws, dir, "derived.obk", srcDoc)
	if err := ws.Save(partDoc); err != nil {
		t.Fatalf("Save derived part: %v", err)
	}
	return store, ws, dir, srcDoc, partDoc
}

// TestDerivedPartRebindsAndIsCurrent derives a saved source assembly into a part, saves and
// reopens the part, and checks the derive re-resolves its source through the reference graph
// (rebound, non-empty source version) and is not flagged out of date (source unchanged).
func TestDerivedPartRebindsAndIsCurrent(t *testing.T) {
	store, _, _, srcDoc, partDoc := savedDerivedPart(t)

	d := derivedComponentOf(t, openPart(t, store, partDoc.FullFileName()))
	if d.SourceVersion() == "" {
		t.Error("reopened derive should have rebound its source (empty source version)")
	}
	if d.OutOfDate() {
		t.Error("derive against an unchanged source should not be out of date")
	}
	if got := d.SourceLink().Document; got != srcDoc.FullFileName() {
		t.Errorf("restored source link document = %q, want %q", got, srcDoc.FullFileName())
	}
}

// TestDerivedPartFlagsStaleSourceAcrossSessions saves a derived part, then edits and re-saves
// the source assembly (re-minting its recipe revision), and checks the part reopened in a
// fresh session flags the derive out of date by the revision mismatch — #715 acceptance 2.
func TestDerivedPartFlagsStaleSourceAcrossSessions(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	widget := savePartDoc(t, ws, dir, "widget.obk")
	srcDoc, srcDef := newAssembly(t, ws, dir, "src.obk")
	if err := ws.Save(srcDoc); err != nil { // revision 1
		t.Fatalf("Save source: %v", err)
	}
	partDoc := deriveSourceIntoPart(t, ws, dir, "derived.obk", srcDoc)
	if err := ws.Save(partDoc); err != nil {
		t.Fatalf("Save derived part: %v", err)
	}

	// Edit the source after the derive was saved and re-save it — re-minting its revision.
	placeFromFile(t, srcDoc, widget, srcDef, "widget:1", math.Identity4())
	if err := ws.Save(srcDoc); err != nil { // revision 2
		t.Fatalf("Re-save edited source: %v", err)
	}

	d := derivedComponentOf(t, openPart(t, store, partDoc.FullFileName()))
	if !d.OutOfDate() {
		t.Error("source edited and saved after the derive ⇒ reopened derive should be out of date")
	}
}

// TestDerivedPartMissingSourceIsNotFatal checks reopening a derived part whose source file
// is gone still succeeds, leaving the derive unbound (no source) rather than panicking.
func TestDerivedPartMissingSourceIsNotFatal(t *testing.T) {
	store, _, dir, _, partDoc := savedDerivedPart(t)
	if err := os.Remove(filepath.Join(dir, "src.obk")); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	d := derivedComponentOf(t, openPart(t, store, partDoc.FullFileName()))
	if d.SourceVersion() != "" {
		t.Error("a missing source should leave the derive unbound (empty source version)")
	}
}
