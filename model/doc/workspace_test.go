// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

// fakeStore is an in-memory Store standing in for the real zip-package backend
// (M03-F03). It persists only the identity that exists today — kind and names —
// which is enough to prove the lifecycle round-trips (CLAUDE.md: named fakes, not
// inline stubs).
type fakeStore struct {
	saved map[string]storedDoc
	loads int // how many times Load actually hit the store
}

type storedDoc struct {
	docType     DocumentType
	displayName string
}

func newFakeStore() *fakeStore { return &fakeStore{saved: map[string]storedDoc{}} }

func (s *fakeStore) Save(d *Document) error {
	s.saved[d.FullDocumentName()] = storedDoc{docType: d.DocumentType(), displayName: d.DisplayName()}
	return nil
}

func (s *fakeStore) SaveCopy(d *Document, target string, meta CopyMetadata) error {
	displayName := d.DisplayName()
	if meta.DisplayName != "" {
		displayName = meta.DisplayName
	}
	s.saved[target] = storedDoc{docType: d.DocumentType(), displayName: displayName}
	return nil
}

func (s *fakeStore) Load(fullDocumentName string) (*Document, error) {
	rec, ok := s.saved[fullDocumentName]
	if !ok {
		return nil, errNotStored{fullDocumentName}
	}
	s.loads++
	return Restore(rec.docType, fullDocumentName, rec.displayName)
}

func (s *fakeStore) Exists(fullDocumentName string) bool {
	_, ok := s.saved[fullDocumentName]
	return ok
}

type errNotStored struct{ name string }

func (e errNotStored) Error() string { return "no document stored at " + e.name }

func TestAddCreatesFromTemplateAndAppears(t *testing.T) {
	ws := NewWorkspace(newFakeStore())

	d, err := ws.Add(Part, "/proj/bracket.obk", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if part, ok := AsPartDocument(d); !ok || part.ComponentDefinition() == nil {
		t.Fatal("Add(Part) did not yield part content")
	}
	if ws.Count() != 1 || ws.LoadedCount() != 1 {
		t.Errorf("Count=%d LoadedCount=%d, want 1 and 1", ws.Count(), ws.LoadedCount())
	}
	if !d.Dirty() {
		t.Error("freshly created document is not dirty")
	}
	if ws.ActiveDocument() != d {
		t.Error("created document is not active")
	}
	if got, ok := ws.ByName("/proj/bracket.obk"); !ok || got != d {
		t.Error("ByName did not find the created document")
	}
}

func TestVisibleVsHiddenOpen(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	vis, _ := ws.Add(Part, "shown.obk", true)
	hid, _ := ws.Add(Assembly, "hidden.obk", false)

	if got := len(ws.VisibleDocuments()); got != 1 {
		t.Fatalf("VisibleDocuments len = %d, want 1", got)
	}
	if ws.VisibleDocuments()[0] != vis {
		t.Error("wrong document reported visible")
	}
	if hid.Visible() {
		t.Error("hidden-open document reports Visible() = true")
	}
}

func TestDeferContentOpensStubWithoutLoad(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)

	stub, err := ws.OpenWithOptions("/lib/screw.obk", OpenOptions{DeferContent: true})
	if err != nil {
		t.Fatalf("OpenWithOptions deferred: %v", err)
	}
	if !stub.IsReferenceStub() {
		t.Error("DeferContent did not yield a reference stub")
	}
	if store.loads != 0 {
		t.Errorf("store.Load called %d times for a deferred open, want 0", store.loads)
	}
	if ws.Count() != 1 || ws.LoadedCount() != 0 {
		t.Errorf("Count=%d LoadedCount=%d, want 1 and 0", ws.Count(), ws.LoadedCount())
	}
}

func TestSavedDocumentReopensIdentically(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	d, _ := ws.Add(Assembly, "/proj/top.obk", true)
	d.SetDisplayName("Top Assembly")
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if d.Dirty() {
		t.Error("document still dirty after Save")
	}

	// A fresh workspace forces the load path (not the already-open shortcut).
	reopened, err := NewWorkspace(store).Open("/proj/top.obk", true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if reopened.DocumentType() != Assembly {
		t.Errorf("reopened type = %v, want Assembly", reopened.DocumentType())
	}
	if reopened.FullDocumentName() != "/proj/top.obk" || reopened.DisplayName() != "Top Assembly" {
		t.Errorf("reopened identity = %q/%q, want preserved", reopened.FullDocumentName(), reopened.DisplayName())
	}
	if !reopened.Open() || reopened.Dirty() {
		t.Error("reopened document should be open and clean")
	}
}

func TestSaveAsRenamesAndPersists(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	d, _ := ws.Add(Part, "draft.obk", true)

	if err := ws.SaveAs(d, "final.obk"); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	if d.FullDocumentName() != "final.obk" {
		t.Errorf("name after SaveAs = %q, want final.obk", d.FullDocumentName())
	}
	if _, ok := ws.ByName("draft.obk"); ok {
		t.Error("old name still resolves after SaveAs")
	}
	if got, ok := ws.ByName("final.obk"); !ok || got != d {
		t.Error("new name does not resolve after SaveAs")
	}
	if !store.Exists("final.obk") || store.Exists("draft.obk") {
		t.Error("store not updated to the new name")
	}
}

func TestOpenAlreadyOpenReturnsSameInstance(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	d, _ := ws.Add(Part, "p.obk", true)
	_ = ws.Save(d)

	again, err := ws.Open("p.obk", true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if again != d {
		t.Error("re-opening an open document returned a different instance")
	}
	if store.loads != 0 {
		t.Errorf("store.Load called %d times re-opening an open doc, want 0", store.loads)
	}
}

func TestCloseSavesDirtyUnlessSkipped(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	d, _ := ws.Add(Part, "p.obk", true) // dirty

	if err := ws.Close(d, false); err != nil { // should save on the way out
		t.Fatalf("Close: %v", err)
	}
	if !store.Exists("p.obk") {
		t.Error("Close(skipSave=false) did not save the dirty document")
	}
	if ws.Count() != 0 {
		t.Errorf("Count after close = %d, want 0", ws.Count())
	}

	other, _ := ws.Add(Part, "q.obk", true)
	if err := ws.Close(other, true); err != nil { // discard
		t.Fatalf("Close skipSave: %v", err)
	}
	if store.Exists("q.obk") {
		t.Error("Close(skipSave=true) saved a discarded document")
	}
}

func TestCloseAllUnreferencedKeepsReferenced(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	top, _ := ws.Add(Assembly, "top.obk", true)
	part, _ := ws.Add(Part, "part.obk", true)
	part.acquireRef() // pretend top references part (graph lands in F04)

	if err := ws.CloseAll(true, true); err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	if _, ok := ws.ByName("part.obk"); !ok {
		t.Error("referenced document was closed by unreferenced-only CloseAll")
	}
	if _, ok := ws.ByName("top.obk"); ok {
		t.Error("unreferenced top document was not closed")
	}
	_ = top

	part.releaseRef()
	if part.Referenced() {
		t.Error("part still referenced after releaseRef")
	}
	if err := ws.CloseAll(true, true); err != nil {
		t.Fatalf("CloseAll second pass: %v", err)
	}
	if ws.Count() != 0 {
		t.Errorf("Count after final CloseAll = %d, want 0", ws.Count())
	}
}

func TestActiveDocumentReassignedOnClose(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	a, _ := ws.Add(Part, "a.obk", true)
	b, _ := ws.Add(Part, "b.obk", true)
	if ws.ActiveDocument() != b {
		t.Fatal("most recently added document should be active")
	}
	if err := ws.SetActiveDocument(a); err != nil {
		t.Fatalf("SetActiveDocument: %v", err)
	}
	_ = ws.Close(a, true)
	if ws.ActiveDocument() != b {
		t.Errorf("active after closing active = %v, want remaining doc b", ws.ActiveDocument())
	}
	_ = ws.Close(b, true)
	if ws.ActiveDocument() != nil {
		t.Error("active should be nil when no documents remain")
	}
}

func TestAddRejectsDuplicateNameAndBadType(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	if _, err := ws.Add(Part, "dup.obk", true); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if _, err := ws.Add(Assembly, "dup.obk", true); err == nil {
		t.Error("Add accepted a duplicate name")
	}
	if _, err := ws.Add(Unknown, "weird.obk", true); err == nil {
		t.Error("Add accepted an invalid document type")
	}
}

// TestBackgroundOpenKeepsActiveAndHidden pins the place-component fix (#764): loading a
// component in the background must reference it in memory without making it visible or
// stealing the active document, so placing a part never switches the tab away from the
// assembly.
func TestBackgroundOpenKeepsActiveAndHidden(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	asm, _ := ws.Add(Assembly, "asm.obk", true)
	part, _ := ws.Add(Part, "part.obk", true) // stage in the store
	if err := ws.Save(part); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := ws.Close(part, false); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := ws.SetActiveDocument(asm); err != nil {
		t.Fatalf("SetActiveDocument: %v", err)
	}

	got, err := ws.OpenWithOptions("part.obk", OpenOptions{Visible: false, Background: true})
	if err != nil {
		t.Fatalf("background open: %v", err)
	}
	if ws.ActiveDocument() != asm {
		t.Error("background open stole the active document; the assembly must stay active")
	}
	if got.Visible() {
		t.Error("background-opened component reports Visible() = true")
	}
	if vis := ws.VisibleDocuments(); len(vis) != 1 || vis[0] != asm {
		t.Errorf("visible documents = %d (want only the assembly); a background component must show no tab", len(vis))
	}
}

// TestBackgroundOpenOfOpenDocPreservesIt checks that re-placing a part the user already has
// open in a tab does not hide it or steal focus.
func TestBackgroundOpenOfOpenDocPreservesIt(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	asm, _ := ws.Add(Assembly, "asm.obk", true)
	part, _ := ws.Add(Part, "part.obk", true) // user has it open, visible, active
	if _, err := ws.OpenWithOptions("part.obk", OpenOptions{Visible: false, Background: true}); err != nil {
		t.Fatalf("background re-open: %v", err)
	}
	if !part.Visible() {
		t.Error("re-placing a part the user has open in a tab hid it")
	}
	if ws.ActiveDocument() != part {
		t.Error("background re-open changed the active document")
	}
	_ = asm
}
