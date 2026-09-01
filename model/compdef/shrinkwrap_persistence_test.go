// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// assemblySourceLink records partDoc's reference to sourceDoc and returns the source's
// identity link — the shared front half of the derive/shrinkwrap-from-document test
// helpers, exactly as the router does at create time.
func assemblySourceLink(partDoc, sourceDoc *doc.Document) feature.DeriveSourceLink {
	partDoc.OpenReference(sourceDoc.FullDocumentName())
	id := sourceDoc.FileIdentity()
	return feature.DeriveSourceLink{
		Document:           sourceDoc.FullDocumentName(),
		InternalName:       id.InternalName,
		DatabaseRevisionID: id.DatabaseRevisionID,
	}
}

// shrinkwrapSourceIntoPart creates a part that shrinkwraps sourceDoc (an assembly),
// recording the source link and reference, and returns the part document.
func shrinkwrapSourceIntoPart(t *testing.T, ws *doc.Workspace, dir, name string, sourceDoc *doc.Document) *doc.Document {
	t.Helper()
	partDoc, err := compdef.AddPart(ws, filepath.Join(dir, name), true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", name, err)
	}
	part := partDoc.Content().(*compdef.PartComponentDefinition)
	source := sourceDoc.Content().(feature.AssemblyBodySource)
	feature.NewShrinkwrapComponents(part.Features()).AddShrinkwrap(source, feature.ShrinkwrapDefinition{}, assemblySourceLink(partDoc, sourceDoc))
	return partDoc
}

// shrinkwrapComponentOf returns the single shrinkwrap component of a part.
func shrinkwrapComponentOf(t *testing.T, part *compdef.PartComponentDefinition) *feature.ShrinkwrapComponent {
	t.Helper()
	for i := 0; i < part.Features().Count(); i++ {
		if sw, ok := part.Features().Item(i).Definition().(*feature.ShrinkwrapComponent); ok {
			return sw
		}
	}
	t.Fatal("part has no shrinkwrap component")
	return nil
}

// TestShrinkwrapPartRebindsOnReopen shrinkwraps a saved source assembly into a part, saves
// and reopens the part, and checks the shrinkwrap re-resolves its source through the part's
// reference graph — rebound and not stale (#715/#749 acceptance).
func TestShrinkwrapPartRebindsOnReopen(t *testing.T) {
	t.Parallel()
	store, ws, dir := assemblyWorkspace(t)
	srcDoc, _ := newAssembly(t, ws, dir, "src.obk")
	if err := ws.Save(srcDoc); err != nil {
		t.Fatalf("Save source: %v", err)
	}
	partDoc := shrinkwrapSourceIntoPart(t, ws, dir, "wrap.obk", srcDoc)
	if err := ws.Save(partDoc); err != nil {
		t.Fatalf("Save shrinkwrap part: %v", err)
	}

	sw := shrinkwrapComponentOf(t, openPart(t, store, partDoc.FullFileName()))
	if sw.SourceVersion() == "" {
		t.Error("reopened shrinkwrap should have rebound its source")
	}
	if sw.OutOfDate() {
		t.Error("shrinkwrap against an unchanged source should not be out of date")
	}
	if got := sw.SourceLink().Document; got != srcDoc.FullFileName() {
		t.Errorf("restored source link document = %q, want %q", got, srcDoc.FullFileName())
	}
}
