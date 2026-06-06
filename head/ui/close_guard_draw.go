//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// closeModal is the chrome's single graceful-close prompt (UI state, head-local).
var closeModal closeGuard

// HandleClose runs the graceful-close flow once per frame and reports whether the app
// should exit. When the window close is requested with unsaved documents it cancels
// the close and shows a "save changes?" prompt; the app only exits on Don't Save, or
// on Save once every document is clean. With no unsaved work, a close exits at once.
// Call it after DrawChrome and before EndFrame (the prompt is an ImGui window).
func HandleClose(win *native.Window, s *app.Session) bool {
	if !closeModal.prompting {
		if !win.ShouldClose() {
			return false
		}
		if len(dirtyDocuments(s)) == 0 {
			return true // nothing unsaved → close immediately
		}
		win.SetShouldClose(false) // cancel the OS close; resolve via the prompt
		closeModal.prompting = true
	}
	return drawCloseModal(s)
}

// drawCloseModal renders the prompt and returns true when the app should exit.
func drawCloseModal(s *app.Session) bool {
	dirty := dirtyDocuments(s)
	if len(dirty) == 0 { // a Save cleared the last one → proceed with the close
		closeModal.prompting = false
		return true
	}
	exit := false
	if native.Begin("Save changes?") {
		native.Text(fmt.Sprintf("%d document(s) have unsaved changes:", len(dirty)))
		for _, d := range dirty {
			native.Text("   - " + d.FullDocumentName())
		}
		if native.Button("Save") {
			saveAllDirty(s) // never-saved docs route to the Save As modal below
		}
		native.SameLine()
		if native.Button("Don't Save") {
			closeModal.prompting = false
			exit = true
		}
		native.SameLine()
		if native.Button("Cancel") {
			closeModal.prompting = false
		}
	}
	native.End()
	drawFileDialog(s) // render the Save As modal if saveAllDirty armed it
	return exit
}

// saveAllDirty saves the active document through the session (a never-saved document
// routes to the Save As modal), then best-effort saves any other dirty documents.
// The close prompt stays up until everything is clean (then the close proceeds) or
// the user chooses Don't Save / Cancel.
func saveAllDirty(s *app.Session) {
	saveActive(s) // Save, or open Save As for a path-less active document
	active := s.Workspace().ActiveDocument()
	for _, d := range dirtyDocuments(s) {
		if d == active {
			continue // handled (or awaiting Save As) above
		}
		if err := s.Workspace().Save(d); err != nil {
			fmt.Fprintf(os.Stderr, "save %s: %v\n", d.FullDocumentName(), err)
		}
	}
}
