// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// allEntityKeys derives the persistent reference keys of every entity across every sketch of
// a part, in order — the full durable-reference surface of the document.
func allEntityKeys(t *testing.T, d *doc.Document, part *compdef.PartComponentDefinition) []string {
	t.Helper()
	var keys []string
	for i := 0; i < part.Sketches().Count(); i++ {
		for _, e := range part.Sketches().Item(i).Entities() {
			keys = append(keys, entityKey(t, d, e))
		}
	}
	return keys
}

// TestMultipleDocumentsKeepDistinctSketchKeys opens two part documents at once, edits their
// sketches interleaved (the kernel juggling several sketches simultaneously), saves and
// reopens both, and proves: each document's keys are stable across its own round trip; the
// two documents' keys never collide (the #153 cross-document guarantee — derived from
// distinct document GUIDs); and editing one reopened document does not perturb the other's
// keys.
func TestMultipleDocumentsKeepDistinctSketchKeys(t *testing.T) {
	t.Parallel()
	store, ws, dir := assemblyWorkspace(t)
	pathA := filepath.Join(dir, "alpha.obk")
	pathB := filepath.Join(dir, "beta.obk")

	docA, partA := newKeyedPart(t, ws, pathA)
	docB, partB := newKeyedPart(t, ws, pathB)
	if docA.FileIdentity().InternalName == docB.FileIdentity().InternalName {
		t.Fatal("two documents share a GUID — cross-document keys could collide")
	}

	// Interleave edits across the two open documents, as a session juggling both would.
	editSketch(t, partA, math.P2(10, 0), math.P2(12, 0))
	editSketch(t, partB, math.P2(20, 0), math.P2(22, 0))
	editSketch(t, partA, math.P2(10, 5), math.P2(12, 5))

	wantA := allEntityKeys(t, docA, partA)
	wantB := allEntityKeys(t, docB, partB)
	assertNoCollision(t, wantA, wantB)

	for _, d := range []*doc.Document{docA, docB} {
		if err := ws.Save(d); err != nil {
			t.Fatalf("Save %q: %v", d.FullFileName(), err)
		}
	}

	rDocA, rPartA := openPartDoc(t, store, pathA)
	rDocB, rPartB := openPartDoc(t, store, pathB)
	assertSameKeys(t, "alpha", wantA, allEntityKeys(t, rDocA, rPartA))
	assertSameKeys(t, "beta", wantB, allEntityKeys(t, rDocB, rPartB))
	assertNoCollision(t, allEntityKeys(t, rDocA, rPartA), allEntityKeys(t, rDocB, rPartB))

	// Editing one reopened document must not disturb the other's keys.
	editSketch(t, rPartA, math.P2(30, 0), math.P2(31, 0))
	assertSameKeys(t, "beta after editing alpha", wantB, allEntityKeys(t, rDocB, rPartB))
}

// newKeyedPart adds a part at path with one rectangle+circle sketch and returns its doc/def.
func newKeyedPart(t *testing.T, ws *doc.Workspace, path string) (*doc.Document, *compdef.PartComponentDefinition) {
	t.Helper()
	d, err := compdef.AddPart(ws, path, true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", path, err)
	}
	part := d.Content().(*compdef.PartComponentDefinition)
	addRectangleSketch(t, part)
	return d, part
}

// editSketch appends a line to the part's first sketch — a representative in-place edit.
func editSketch(t *testing.T, part *compdef.PartComponentDefinition, a, b math.Point2) {
	t.Helper()
	sk := part.Sketches().Item(0)
	sk.Lines().Add(sk.NewPoint(a), sk.NewPoint(b))
}

// openPartDoc reopens a saved document from disk in a fresh workspace, returning doc + def.
func openPartDoc(t *testing.T, store *persistence.PackageStore, path string) (*doc.Document, *compdef.PartComponentDefinition) {
	t.Helper()
	d, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open %q: %v", path, err)
	}
	return d, d.Content().(*compdef.PartComponentDefinition)
}

// assertSameKeys fails when two key slices differ (order-sensitive).
func assertSameKeys(t *testing.T, label string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("%s: key count = %d, want %d", label, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: key %d = %q, want %q", label, i, got[i], want[i])
		}
	}
}

// assertNoCollision fails when any key appears in both slices.
func assertNoCollision(t *testing.T, a, b []string) {
	t.Helper()
	seen := make(map[string]bool, len(a))
	for _, k := range a {
		seen[k] = true
	}
	for _, k := range b {
		if seen[k] {
			t.Errorf("cross-document key collision: %q is in both documents", k)
		}
	}
}
