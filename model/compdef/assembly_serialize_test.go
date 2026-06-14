// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/persistence"
)

// assemblyWorkspace returns a package-store-backed workspace and its temp directory —
// the fixture every assembly round-trip test builds on.
func assemblyWorkspace(t *testing.T) (*persistence.PackageStore, *doc.Workspace, string) {
	t.Helper()
	store := persistence.NewPackageStore()
	return store, doc.NewWorkspace(store), t.TempDir()
}

// newAssembly adds and returns an assembly document and its definition at name in dir.
func newAssembly(t *testing.T, ws *doc.Workspace, dir, name string) (*doc.Document, *compdef.AssemblyComponentDefinition) {
	t.Helper()
	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, name), true)
	if err != nil {
		t.Fatalf("AddAssembly %q: %v", name, err)
	}
	return asm, asm.Content().(*compdef.AssemblyComponentDefinition)
}

// savePartDoc creates a part document at name in dir/ws and saves it, so an assembly can
// place it from a real, resolvable file. The part is empty — occurrence persistence
// rebinds by document, independent of the component's geometry, so a body is unneeded
// here (geometry round-tripping is covered by the part recipe tests).
func savePartDoc(t *testing.T, ws *doc.Workspace, dir, name string) *doc.Document {
	t.Helper()
	d, err := compdef.AddPart(ws, filepath.Join(dir, name), true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", name, err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save part %q: %v", name, err)
	}
	return d
}

// placedAssembly builds the common fixture: a store-backed workspace with a saved
// "widget.obk" component part and a new empty "asm.obk" assembly, returning the pieces a
// placement round-trip needs.
func placedAssembly(t *testing.T) (store *persistence.PackageStore, ws *doc.Workspace, asm, widget *doc.Document, asmDef *compdef.AssemblyComponentDefinition) {
	t.Helper()
	store, ws, dir := assemblyWorkspace(t)
	widget = savePartDoc(t, ws, dir, "widget.obk")
	asm, asmDef = newAssembly(t, ws, dir, "asm.obk")
	return store, ws, asm, widget, asmDef
}

// placeFromFile places componentDoc into asmDef under name at transform, failing the test
// on error — the persisting place path the round-trip exercises.
func placeFromFile(t *testing.T, asm, componentDoc *doc.Document, asmDef *compdef.AssemblyComponentDefinition, name string, transform math.Matrix4) *occurrence.Occurrence {
	t.Helper()
	o, err := asmDef.PlaceComponentFromFile(asm, componentDoc, name, transform)
	if err != nil {
		t.Fatalf("place %q: %v", name, err)
	}
	return o
}

// reopenAssembly saves the assembly through the store and reopens it in a fresh workspace
// backed by the same store, so its occurrences resolve their component documents from disk
// through the reference graph — the real #715 round trip.
func reopenAssembly(t *testing.T, store *persistence.PackageStore, ws *doc.Workspace, asm *doc.Document) *compdef.AssemblyComponentDefinition {
	t.Helper()
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save assembly: %v", err)
	}
	return openAssembly(t, store, asm.FullFileName())
}

// openAssembly opens name in a fresh store-backed workspace (forcing the on-disk load and
// reference-resolution path) and returns its assembly definition.
func openAssembly(t *testing.T, store *persistence.PackageStore, name string) *compdef.AssemblyComponentDefinition {
	t.Helper()
	reopened, err := doc.NewWorkspace(store).Open(name, true)
	if err != nil {
		t.Fatalf("Open assembly %q: %v", name, err)
	}
	def, ok := reopened.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		t.Fatalf("reopened content is %T, want *AssemblyComponentDefinition", reopened.Content())
	}
	return def
}

// TestEmptyAssemblyRoundTrips checks a placed-component-free assembly round-trips with
// zero occurrences.
func TestEmptyAssemblyRoundTrips(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	asm, _ := newAssembly(t, ws, dir, "empty.obk")

	if def := reopenAssembly(t, store, ws, asm); def.Occurrences().Count() != 0 {
		t.Errorf("reopened empty assembly has %d occurrences, want 0", def.Occurrences().Count())
	}
}

// TestAssemblyOccurrencesRoundTrip places one shared component twice with distinct
// transforms and per-instance state, and checks both occurrences — names, placements,
// suppression, grounding — restore through the reference graph.
func TestAssemblyOccurrencesRoundTrip(t *testing.T) {
	store, ws, asm, widget, asmDef := placedAssembly(t)
	at3 := math.Translation4(math.V3(3, 0, 0))
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4()).SetGrounded(true)
	placeFromFile(t, asm, widget, asmDef, "widget:2", at3).SetSuppressed(true)

	def := reopenAssembly(t, store, ws, asm)
	if def.Occurrences().Count() != 2 {
		t.Fatalf("reopened assembly has %d occurrences, want 2", def.Occurrences().Count())
	}
	r1, r2 := def.Occurrences().Item(0), def.Occurrences().Item(1)
	if r1.Name() != "widget:1" || !r1.Grounded() || r1.Suppressed() {
		t.Errorf("occurrence 0 = {name:%q grounded:%v suppressed:%v}, want widget:1 grounded", r1.Name(), r1.Grounded(), r1.Suppressed())
	}
	if r2.Name() != "widget:2" || !r2.Suppressed() {
		t.Errorf("occurrence 1 = {name:%q suppressed:%v}, want widget:2 suppressed", r2.Name(), r2.Suppressed())
	}
	if got := r2.Transform().Cells(); got != at3.Cells() {
		t.Errorf("occurrence 1 transform = %v, want the x=3 translation", got)
	}
	if r1.Definition() == nil {
		t.Error("occurrence 0 has a nil definition after reopen")
	}
}

// TestAssemblyReferenceDeduped checks two placements of the same component yield a single
// assembly→component reference edge and a single persisted file-reference record (the
// referenced-by count must not inflate with placement count).
func TestAssemblyReferenceDeduped(t *testing.T) {
	_, ws, asm, widget, asmDef := placedAssembly(t)
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	placeFromFile(t, asm, widget, asmDef, "widget:2", math.Identity4())

	if got := len(asm.References()); got != 1 {
		t.Errorf("assembly has %d reference edges, want 1 (deduped)", got)
	}
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := len(asm.FileReferenceRecords()); got != 1 {
		t.Errorf("assembly persisted %d file-reference records, want 1", got)
	}
}

// TestAssemblyMissingComponentIsNotFatal checks reopening an assembly whose component
// file is gone still succeeds: the occurrence restores as an unresolved placeholder
// (empty range box), never a panic or a failed open.
func TestAssemblyMissingComponentIsNotFatal(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	widget := savePartDoc(t, ws, dir, "gone.obk")
	asm, asmDef := newAssembly(t, ws, dir, "asm.obk")
	placeFromFile(t, asm, widget, asmDef, "gone:1", math.Identity4())
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save assembly: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "gone.obk")); err != nil {
		t.Fatalf("remove component: %v", err)
	}

	def := openAssembly(t, store, asm.FullFileName())
	if def.Occurrences().Count() != 1 {
		t.Fatalf("reopened assembly has %d occurrences, want 1 placeholder", def.Occurrences().Count())
	}
	if !def.Occurrences().Item(0).Definition().RangeBox().IsEmpty() {
		t.Error("missing-component placeholder should contribute an empty range box")
	}
}

// TestInMemoryPlaceNotPersisted checks an occurrence placed from a bare definition (no
// document, via Place) is omitted from the recipe — it has no file to resolve on reopen,
// so persisting it would yield an unrestorable record.
func TestInMemoryPlaceNotPersisted(t *testing.T) {
	store, ws, asm, widget, asmDef := placedAssembly(t)
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	asmDef.Place("loose:1", widget.Content().(occurrence.Definition), math.Identity4())

	def := reopenAssembly(t, store, ws, asm)
	if def.Occurrences().Count() != 1 || def.Occurrences().Item(0).Name() != "widget:1" {
		t.Errorf("reopened occurrences = %d, want only the file-backed widget:1", def.Occurrences().Count())
	}
}

// TestNestedAssemblyRoundTrip checks an assembly that places a sub-assembly (which itself
// places a part) restores recursively: opening the top assembly hidden-opens the
// sub-assembly, which hidden-opens the part, and every placement is rebound.
func TestNestedAssemblyRoundTrip(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	widget := savePartDoc(t, ws, dir, "widget.obk")
	sub, subDef := newAssembly(t, ws, dir, "sub.obk")
	placeFromFile(t, sub, widget, subDef, "widget:1", math.Identity4())
	if err := ws.Save(sub); err != nil {
		t.Fatalf("Save sub: %v", err)
	}
	top, topDef := newAssembly(t, ws, dir, "top.obk")
	placeFromFile(t, top, sub, topDef, "sub:1", math.Translation4(math.V3(5, 0, 0)))

	def := reopenAssembly(t, store, ws, top)
	if def.Occurrences().Count() != 1 {
		t.Fatalf("reopened top has %d occurrences, want 1 (the sub-assembly)", def.Occurrences().Count())
	}
	nested, ok := def.Occurrences().Item(0).Definition().(occurrence.Composite)
	if !ok {
		t.Fatalf("sub occurrence definition %T is not a composite (sub-assembly)", def.Occurrences().Item(0).Definition())
	}
	if nested.Occurrences().Count() != 1 || nested.Occurrences().Item(0).Name() != "widget:1" {
		t.Errorf("nested sub-assembly = %d occurrences, want the restored widget:1", nested.Occurrences().Count())
	}
}
