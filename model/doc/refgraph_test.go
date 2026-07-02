// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

func TestAssemblyReportsPartsAndPartReportsAssembly(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	asm, _ := ws.Add(Assembly, "top.iam.obk", true)
	p1, _ := ws.Add(Part, "a.obk", true)
	p2, _ := ws.Add(Part, "b.obk", true)

	if _, err := asm.AddReference("a.obk"); err != nil {
		t.Fatalf("AddReference a: %v", err)
	}
	if _, err := asm.AddReference("b.obk"); err != nil {
		t.Fatalf("AddReference b: %v", err)
	}

	refs := asm.ReferencedDocuments()
	if len(refs) != 2 || refs[0] != p1 || refs[1] != p2 {
		t.Fatalf("ReferencedDocuments = %v, want [a b]", refs)
	}
	referencing := p1.ReferencingDocuments()
	if len(referencing) != 1 || referencing[0] != asm {
		t.Errorf("a.ReferencingDocuments = %v, want [top]", referencing)
	}
	if !p1.Referenced() {
		t.Error("referenced part reports Referenced() = false")
	}
}

func TestBrokenReferenceFlaggedNotFatal(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	asm, _ := ws.Add(Assembly, "top.obk", true)
	present, _ := ws.Add(Part, "here.obk", true)
	_, _ = asm.AddReference("here.obk")
	desc, _ := asm.AddReference("ghost.obk") // not open, not in store

	refs := asm.ReferencedDocuments() // must not panic
	if len(refs) != 1 || refs[0] != present {
		t.Errorf("ReferencedDocuments = %v, want only the resolvable [here]", refs)
	}
	if !desc.IsBroken() {
		t.Error("unresolvable reference not flagged broken")
	}
}

func TestAllReferencedIsTransitive(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	top, _ := ws.Add(Assembly, "top.obk", true)
	sub, _ := ws.Add(Assembly, "sub.obk", true)
	leaf, _ := ws.Add(Part, "leaf.obk", true)
	_, _ = top.AddReference("sub.obk")
	_, _ = sub.AddReference("leaf.obk")

	all := top.AllReferencedDocuments()
	if len(all) != 2 || all[0] != sub || all[1] != leaf {
		t.Errorf("AllReferencedDocuments = %v, want [sub leaf]", all)
	}
}

func TestReferencedDocumentLazyLoadsFromStore(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store, nil)
	part, _ := ws.Add(Part, "part.obk", true)
	_ = ws.Save(part)
	_ = ws.Close(part, true) // gone from the collection, still on disk

	asm, _ := ws.Add(Assembly, "top.obk", true)
	desc, _ := asm.AddReference("part.obk")
	if loaded := store.loads; loaded != 0 {
		t.Fatalf("store loaded before resolution: %d", loaded)
	}

	refs := asm.ReferencedDocuments()
	if len(refs) != 1 || refs[0].FullDocumentName() != "part.obk" {
		t.Fatalf("lazy resolution = %v, want the part loaded from store", refs)
	}
	if store.loads != 1 {
		t.Errorf("store.loads = %d, want 1 lazy load", store.loads)
	}
	if d, ok := desc.ReferencedDocument(); !ok || !d.Open() {
		t.Error("descriptor did not cache an open resolved document")
	}
}

func TestRemoveReference(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	asm, _ := ws.Add(Assembly, "top.obk", true)
	part, _ := ws.Add(Part, "p.obk", true)
	_, _ = asm.AddReference("p.obk")
	if !part.Referenced() {
		t.Fatal("part not referenced after AddReference")
	}

	if !asm.RemoveReference("p.obk") {
		t.Error("RemoveReference returned false for an existing edge")
	}
	if len(asm.ReferencedDocuments()) != 0 {
		t.Error("reference survived RemoveReference")
	}
	if part.Referenced() {
		t.Error("part still referenced after RemoveReference")
	}
	if asm.RemoveReference("p.obk") {
		t.Error("RemoveReference returned true for a missing edge")
	}
}

func TestUnreferencedCloseKeepsReferencedPart(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	top, _ := ws.Add(Assembly, "top.obk", true)
	_, _ = ws.Add(Part, "p.obk", true)
	_, _ = top.AddReference("p.obk")

	if err := ws.CloseAll(true, true); err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	if _, ok := ws.ByName("p.obk"); !ok {
		t.Error("referenced part was closed by unreferenced-only CloseAll")
	}
	if _, ok := ws.ByName("top.obk"); ok {
		t.Error("unreferenced assembly was not closed")
	}
}

func TestStandaloneDocumentHasNoGraph(t *testing.T) {
	d := NewPartDocument("loose.obk")
	if _, err := d.AddReference("x.obk"); err == nil {
		t.Error("AddReference on a standalone document did not error")
	}
	if d.RemoveReference("x.obk") {
		t.Error("RemoveReference on a standalone document returned true")
	}
	if d.ReferencedDocuments() != nil || d.ReferencingDocuments() != nil ||
		d.AllReferencedDocuments() != nil || d.References() != nil {
		t.Error("standalone document returned non-nil reference queries")
	}
}

func TestDocumentReferencesListsDescriptors(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	if ws.References() == nil {
		t.Fatal("Workspace.References() is nil")
	}
	asm, _ := ws.Add(Assembly, "top.obk", true)
	_, _ = asm.AddReference("a.obk")
	_, _ = asm.AddReference("b.obk")

	descs := asm.References()
	if len(descs) != 2 {
		t.Fatalf("References len = %d, want 2", len(descs))
	}
	if descs[0].FullDocumentName() != "a.obk" || descs[1].FullDocumentName() != "b.obk" {
		t.Errorf("descriptor names = %q,%q, want a,b", descs[0].FullDocumentName(), descs[1].FullDocumentName())
	}
}

func TestDescriptorReferenceKey(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	asm, _ := ws.Add(Assembly, "top.obk", true)
	desc, _ := asm.AddReference("p.obk")
	if desc.ReferenceKey() != nil {
		t.Error("new descriptor already has a reference key")
	}
	desc.SetReferenceKey([]byte{1, 2, 3})
	if got := desc.ReferenceKey(); len(got) != 3 || got[0] != 1 {
		t.Errorf("ReferenceKey = %v, want [1 2 3]", got)
	}
	desc.SetNeedsUpdate(true)
	if !desc.NeedsUpdate() {
		t.Error("SetNeedsUpdate not reflected")
	}
}
