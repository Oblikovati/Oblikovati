// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// derivePartIntoPart creates a part that derives sourceDoc (another part) with transform,
// capturing the source identity link and recording the part→source reference — the model
// equivalent of a mirror-into-part (#717). Returns the deriving part document.
func derivePartIntoPart(t *testing.T, ws *doc.Workspace, dir, name string, sourceDoc *doc.Document, transform math.Matrix4) *doc.Document {
	t.Helper()
	partDoc, err := compdef.AddPart(ws, filepath.Join(dir, name), true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", name, err)
	}
	part := partDoc.Content().(*compdef.PartComponentDefinition)
	source := sourceDoc.Content().(feature.BodySource)
	feature.NewDerivedComponents(part.Features()).AddDerived(source, transform, assemblySourceLink(partDoc, sourceDoc))
	return partDoc
}

// derivedPartComponentOf returns the single derived-part component of a part.
func derivedPartComponentOf(t *testing.T, part *compdef.PartComponentDefinition) *feature.DerivedPartComponent {
	t.Helper()
	for i := 0; i < part.Features().Count(); i++ {
		if d, ok := part.Features().Item(i).Definition().(*feature.DerivedPartComponent); ok {
			return d
		}
	}
	t.Fatal("part has no derived-part component")
	return nil
}

// TestDerivedFromPartRebindsOnReopen derives one part into another with a reflection
// transform (the mirror-into-part shape), saves and reopens the deriving part, and checks
// the derive re-resolves its source through the part's reference graph — rebound, not
// stale, transform preserved — so a handed part survives a session (#717 foundation).
func TestDerivedFromPartRebindsOnReopen(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	srcDoc := savePartDoc(t, ws, dir, "source.obk")
	x, err := math.NewUnitVector3(1, 0, 0)
	if err != nil {
		t.Fatalf("NewUnitVector3: %v", err)
	}
	reflect := math.Reflection4(math.P3(0, 0, 0), x)
	partDoc := derivePartIntoPart(t, ws, dir, "derived.obk", srcDoc, reflect)
	if err := ws.Save(partDoc); err != nil {
		t.Fatalf("Save derived part: %v", err)
	}

	d := derivedPartComponentOf(t, openPart(t, store, partDoc.FullFileName()))
	if d.SourceVersion() == "" {
		t.Error("reopened derived-part should have rebound its source")
	}
	if d.OutOfDate() {
		t.Error("derive against an unchanged source should not be out of date")
	}
	if got := d.SourceLink().Document; got != srcDoc.FullFileName() {
		t.Errorf("restored source link document = %q, want %q", got, srcDoc.FullFileName())
	}
	if d.Transform().Cells() != reflect.Cells() {
		t.Error("restored derive transform does not match the saved reflection")
	}
}
