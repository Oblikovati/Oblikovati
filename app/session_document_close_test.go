// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// addBackgroundPart opens a second part document and re-activates the first, so the returned
// document is open but NOT active — the case a close must leave the active edit alone for.
func addBackgroundPart(t *testing.T, s *Session) *doc.Document {
	t.Helper()
	active := s.ActiveDocument()
	other, err := s.Workspace().Add(doc.Part, "other.obk", true)
	if err != nil {
		t.Fatalf("Add second part: %v", err)
	}
	other.SetContent(compdef.NewPartComponentDefinition())
	if err := s.Workspace().SetActiveDocument(active); err != nil {
		t.Fatalf("re-activate first part: %v", err)
	}
	return other
}

// #2040: the close paths used to call Workspace().Close directly, leaving the tool armed and
// s.activeSketch pointing into the destroyed document.
func TestCloseDocumentLeavesSketchEnvironmentAndDisarmsTool(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.StartTool(NewRectangleTool())
	s.Click(40, 40) // a half-placed rectangle: the tool is armed with pending input

	if err := s.CloseDocument(s.ActiveDocument(), true); err != nil {
		t.Fatalf("CloseDocument: %v", err)
	}
	if s.InSketch() {
		t.Error("still in the sketch environment after closing the document")
	}
	if s.ActiveSketch() != nil {
		t.Errorf("ActiveSketch is %v, want nil — it points into the destroyed document", s.ActiveSketch())
	}
	if s.ActiveTool() != nil {
		t.Errorf("tool %q still armed after closing the document", s.ActiveTool().Name())
	}
}

func TestCloseDocumentLeavesSketch3DEnvironment(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	if err := s.CloseDocument(s.ActiveDocument(), true); err != nil {
		t.Fatalf("CloseDocument: %v", err)
	}
	if s.InSketch3D() {
		t.Error("still in the 3D sketch environment after closing the document")
	}
	if s.ActiveSketch3D() != nil {
		t.Error("ActiveSketch3D points into the destroyed document")
	}
}

// A fresh document after a close-all must be usable: nothing may still resolve to the closed
// document's sketch, which is what made Sketch.AutoDimension answer ok while changing nothing.
func TestNewDocumentAfterCloseStartsInBaseEnvironment(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if err := s.CloseDocument(s.ActiveDocument(), true); err != nil {
		t.Fatalf("CloseDocument: %v", err)
	}
	fresh, err := s.Workspace().Add(doc.Part, "fresh.obk", true)
	if err != nil {
		t.Fatalf("Add fresh part: %v", err)
	}
	fresh.SetContent(compdef.NewPartComponentDefinition())
	if s.InSketch() || s.InSketch3D() {
		t.Error("a brand-new document inherited the closed document's sketch environment")
	}
	if !s.CanCreateSketch() {
		t.Error("Create 2D Sketch is disabled on a fresh part — the stale edit is still open")
	}
}

func TestCloseBackgroundDocumentKeepsActiveEditState(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.StartTool(NewRectangleTool())
	background := addBackgroundPart(t, s)

	if err := s.CloseDocument(background, true); err != nil {
		t.Fatalf("CloseDocument(background): %v", err)
	}
	if !s.InSketch() {
		t.Error("closing a background document dropped the active document's sketch edit")
	}
	if s.ActiveTool() == nil {
		t.Error("closing a background document disarmed the active document's tool")
	}
}

func TestCloseDocumentClearsSelectionIntoTheClosedDocument(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketchOnOrigin(OriginXY)
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.Selection().Add(SketchHandle{Sketch: sk})
	if s.Selection().Count() == 0 {
		t.Fatal("fixture failed to select the sketch")
	}
	if err := s.CloseDocument(s.ActiveDocument(), true); err != nil {
		t.Fatalf("CloseDocument: %v", err)
	}
	if n := s.Selection().Count(); n != 0 {
		t.Errorf("selection holds %d refs into the destroyed document, want 0", n)
	}
}

func TestCloseDocumentRejectsNilDocument(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if err := s.CloseDocument(nil, true); err == nil {
		t.Error("CloseDocument(nil) should error rather than close nothing silently")
	}
}
