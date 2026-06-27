//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// dirtyPromptSession returns a session holding one unsaved (dirty) part, so the graceful-close
// and per-tab close prompts both have something to warn about.
func dirtyPromptSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "close-prompt-test.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	pd.MarkDirty()
	return s
}

// TestCloseSavePromptDrawsWhileDirty is the #1474 regression guard for the window-close
// "Save changes?" prompt: it drives a real frame through drawCloseModal (which now centres the
// window via native.CenterNextWindow) with a dirty document open, and asserts the close does not
// proceed while work is unsaved. A panic in the centring/Begin path would fail here.
func TestCloseSavePromptDrawsWhileDirty(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil // rebind the icon cache to this fresh window
	s := dirtyPromptSession(t)
	if len(dirtyDocuments(s)) == 0 {
		t.Fatal("expected the part to be dirty so the prompt has something to show")
	}

	closeModal.prompting = true
	defer func() { closeModal.prompting = false }()
	win.BeginFrame()
	exit := drawCloseModal(s)
	win.EndFrame(0.1, 0.1, 0.1)
	if exit {
		t.Error("drawCloseModal proceeded with the close while a document was still dirty")
	}
}

// TestDocumentClosePromptDrawsWhileDirty is the same guard for the per-tab "Save document
// changes?" prompt (drawDocumentClosePrompt), which is also centred now (#1474). With a dirty
// pending document it must keep the prompt up rather than dismiss itself.
func TestDocumentClosePromptDrawsWhileDirty(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil
	s := dirtyPromptSession(t)

	closeDocumentModal.pending = s.ActiveDocument()
	defer func() { closeDocumentModal.pending = nil }()
	win.BeginFrame()
	drawDocumentClosePrompt(s)
	win.EndFrame(0.1, 0.1, 0.1)
	if closeDocumentModal.pending == nil {
		t.Error("drawDocumentClosePrompt dismissed itself while the document was still dirty")
	}
}
