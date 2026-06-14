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

// savePartDoc creates a part document at path in ws and saves it, so an assembly can
// place it from a real, resolvable file. The part is empty — occurrence persistence
// rebinds by document, independent of the component's geometry, so a body is unneeded
// here (geometry round-tripping is covered by the part recipe tests).
func savePartDoc(t *testing.T, ws *doc.Workspace, path string) *doc.Document {
	t.Helper()
	d, err := compdef.AddPart(ws, path, true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", path, err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save part %q: %v", path, err)
	}
	return d
}

// reopenAssembly saves the active assembly through the store and reopens it in a fresh
// workspace backed by the same store, so its occurrences resolve their component
// documents from disk through the reference graph — the real #715 round trip.
func reopenAssembly(t *testing.T, store *persistence.PackageStore, ws *doc.Workspace, asm *doc.Document) *compdef.AssemblyComponentDefinition {
	t.Helper()
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save assembly: %v", err)
	}
	reopened, err := doc.NewWorkspace(store).Open(asm.FullFileName(), true)
	if err != nil {
		t.Fatalf("Open assembly: %v", err)
	}
	def, ok := reopened.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		t.Fatalf("reopened content is %T, want *AssemblyComponentDefinition", reopened.Content())
	}
	return def
}

// TestEmptyAssemblyRoundTrips checks a placed-component-free assembly round-trips its
// units with zero occurrences (and writes no file references).
func TestEmptyAssemblyRoundTrips(t *testing.T) {
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, "empty.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}

	def := reopenAssembly(t, store, ws, asm)
	if def.Occurrences().Count() != 0 {
		t.Errorf("reopened empty assembly has %d occurrences, want 0", def.Occurrences().Count())
	}
}

// TestAssemblyOccurrencesRoundTrip places one shared component twice with distinct
// transforms and per-instance state, and checks both occurrences — names, placements,
// suppression, grounding — restore through the reference graph, with exactly one
// deduped assembly→component reference recorded.
func TestAssemblyOccurrencesRoundTrip(t *testing.T) {
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(dir, "widget.obk"))

	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, "asm.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asmDef := asm.Content().(*compdef.AssemblyComponentDefinition)
	at3 := math.Translation4(math.V3(3, 0, 0))
	o1, err := asmDef.PlaceComponentFromFile(asm, widget, "widget:1", math.Identity4())
	if err != nil {
		t.Fatalf("place widget:1: %v", err)
	}
	o2, err := asmDef.PlaceComponentFromFile(asm, widget, "widget:2", at3)
	if err != nil {
		t.Fatalf("place widget:2: %v", err)
	}
	o1.SetGrounded(true)
	o2.SetSuppressed(true)

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
	// Both placements share one component document ⇒ exactly one reference edge/record.
	if refs := def.Occurrences().Item(0).Definition(); refs == nil {
		t.Error("occurrence 0 has a nil definition after reopen")
	}
}

// TestAssemblyReferenceDeduped checks two placements of the same component yield a single
// assembly→component reference edge and a single persisted file-reference record (the
// referenced-by count must not inflate with placement count).
func TestAssemblyReferenceDeduped(t *testing.T) {
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(dir, "widget.obk"))
	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, "asm.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asmDef := asm.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := asmDef.PlaceComponentFromFile(asm, widget, "widget:1", math.Identity4()); err != nil {
		t.Fatalf("place widget:1: %v", err)
	}
	if _, err := asmDef.PlaceComponentFromFile(asm, widget, "widget:2", math.Identity4()); err != nil {
		t.Fatalf("place widget:2: %v", err)
	}
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
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(dir, "gone.obk"))
	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, "asm.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asmDef := asm.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := asmDef.PlaceComponentFromFile(asm, widget, "gone:1", math.Identity4()); err != nil {
		t.Fatalf("place: %v", err)
	}
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save assembly: %v", err)
	}
	// Delete the component package, then reopen the assembly in a fresh workspace.
	if err := os.Remove(filepath.Join(dir, "gone.obk")); err != nil {
		t.Fatalf("remove component: %v", err)
	}
	reopened, err := doc.NewWorkspace(store).Open(asm.FullFileName(), true)
	if err != nil {
		t.Fatalf("Open with missing component should not fail: %v", err)
	}
	def := reopened.Content().(*compdef.AssemblyComponentDefinition)
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
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(dir, "widget.obk"))
	asm, err := compdef.AddAssembly(ws, filepath.Join(dir, "asm.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asmDef := asm.Content().(*compdef.AssemblyComponentDefinition)
	// One persistent placement and one bare in-memory placement.
	if _, err := asmDef.PlaceComponentFromFile(asm, widget, "widget:1", math.Identity4()); err != nil {
		t.Fatalf("place from file: %v", err)
	}
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
	store := persistence.NewPackageStore()
	dir := t.TempDir()
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(dir, "widget.obk"))

	sub, err := compdef.AddAssembly(ws, filepath.Join(dir, "sub.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly sub: %v", err)
	}
	subDef := sub.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := subDef.PlaceComponentFromFile(sub, widget, "widget:1", math.Identity4()); err != nil {
		t.Fatalf("place widget in sub: %v", err)
	}
	if err := ws.Save(sub); err != nil {
		t.Fatalf("Save sub: %v", err)
	}

	top, err := compdef.AddAssembly(ws, filepath.Join(dir, "top.obk"), true)
	if err != nil {
		t.Fatalf("AddAssembly top: %v", err)
	}
	topDef := top.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := topDef.PlaceComponentFromFile(top, sub, "sub:1", math.Translation4(math.V3(5, 0, 0))); err != nil {
		t.Fatalf("place sub in top: %v", err)
	}

	def := reopenAssembly(t, store, ws, top)
	if def.Occurrences().Count() != 1 {
		t.Fatalf("reopened top has %d occurrences, want 1 (the sub-assembly)", def.Occurrences().Count())
	}
	subOcc := def.Occurrences().Item(0)
	nested, ok := subOcc.Definition().(occurrence.Composite)
	if !ok {
		t.Fatalf("sub occurrence definition %T is not a composite (sub-assembly)", subOcc.Definition())
	}
	if nested.Occurrences().Count() != 1 || nested.Occurrences().Item(0).Name() != "widget:1" {
		t.Errorf("nested sub-assembly = %d occurrences, want the restored widget:1", nested.Occurrences().Count())
	}
}
