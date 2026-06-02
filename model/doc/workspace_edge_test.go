// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

func TestTypedViewHelpers(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	part, _ := ws.Add(Part, "p.obk", true)
	asm, _ := ws.Add(Assembly, "a.obk", true)
	dwg, _ := ws.Add(Drawing, "d.obk", true)
	prn, _ := ws.Add(Presentation, "n.obk", true)

	if v, ok := AsPartDocument(part); !ok || v.ComponentDefinition() == nil {
		t.Error("AsPartDocument failed on a part")
	}
	if v, ok := AsAssemblyDocument(asm); !ok || v.ComponentDefinition() == nil {
		t.Error("AsAssemblyDocument failed on an assembly")
	}
	if v, ok := AsDrawingDocument(dwg); !ok || v.DrawingContent() == nil {
		t.Error("AsDrawingDocument failed on a drawing")
	}
	if v, ok := AsPresentationDocument(prn); !ok || v.PresentationContent() == nil {
		t.Error("AsPresentationDocument failed on a presentation")
	}
	// Wrong-kind conversions must report false, not panic.
	if _, ok := AsAssemblyDocument(part); ok {
		t.Error("AsAssemblyDocument succeeded on a part")
	}
	if _, ok := AsPartDocument(asm); ok {
		t.Error("AsPartDocument succeeded on an assembly")
	}
}

func TestCompactedIsFalse(t *testing.T) {
	if NewPartDocument("p.obk").Compacted() {
		t.Error("Compacted() = true; the atomic-save format never compacts")
	}
}

func TestNilStoreErrorsOnSaveAndOpen(t *testing.T) {
	ws := NewWorkspace(nil)
	d, _ := ws.Add(Part, "p.obk", true)
	if err := ws.Save(d); err == nil {
		t.Error("Save with nil store did not error")
	}
	if _, err := ws.Open("missing.obk", true); err == nil {
		t.Error("Open with nil store did not error")
	}
	// A deferred open needs no store and must still succeed.
	if _, err := ws.OpenWithOptions("stub.obk", OpenOptions{DeferContent: true}); err != nil {
		t.Errorf("deferred open with nil store errored: %v", err)
	}
}

func TestOpenMissingDocumentErrors(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	if _, err := ws.Open("/nowhere.obk", true); err == nil {
		t.Error("Open of an unstored document did not error")
	}
}

func TestByIDAndSetActiveErrors(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	d, _ := ws.Add(Part, "p.obk", true)
	if got, ok := ws.ByID(d.ID()); !ok || got != d {
		t.Error("ByID did not return the document")
	}
	if _, ok := ws.ByID(ID(999999)); ok {
		t.Error("ByID returned a document for an unknown id")
	}
	stray := NewPartDocument("not-registered.obk")
	if err := ws.SetActiveDocument(stray.Document); err == nil {
		t.Error("SetActiveDocument accepted a document not in the workspace")
	}
}

func TestSaveAsRejectsCollision(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	a, _ := ws.Add(Part, "a.obk", true)
	_, _ = ws.Add(Part, "b.obk", true)
	if err := ws.SaveAs(a, "b.obk"); err == nil {
		t.Error("SaveAs onto an open document's name did not error")
	}
	// SaveAs to a document's own name is a plain save.
	if err := ws.SaveAs(a, "a.obk"); err != nil {
		t.Errorf("SaveAs to same name errored: %v", err)
	}
}

func TestNewContentRejectsUnknown(t *testing.T) {
	if _, err := newContent(Unknown); err == nil {
		t.Error("newContent(Unknown) did not error")
	}
	if _, err := Restore(DocumentType(42), "x.obk", "x"); err == nil {
		t.Error("Restore with bad type did not error")
	}
}
