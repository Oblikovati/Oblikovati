//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"errors"
	"fmt"
	"os"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// fileModal is the chrome's single path-entry dialog (UI state, not model state, so it
// lives here in the head). File-menu items arm it; drawFileDialog renders it.
var fileModal fileDialog

// drawFileDialog renders the path modal when armed and applies a confirmed Open/Save As
// to the session. DrawChrome calls it once per frame. Failures go to stderr — the head
// has no message box yet (a fast-follow when the notification surface lands).
func drawFileDialog(s *app.Session) {
	if !fileModal.isOpen() {
		return
	}
	var act fileAction
	if native.Begin(fileModal.title()) {
		native.Text("File path:")
		native.InputText("##file-path", fileModal.path[:])
		if native.Button("OK") {
			act = fileModal.confirm()
		}
		native.SameLine()
		if native.Button("Cancel") {
			fileModal.cancel()
		}
	}
	native.End()
	applyFileAction(s, act)
}

// applyFileAction performs the confirmed file operation. A successful Save As becomes
// the document's path, so a later File ▸ Save writes straight through.
func applyFileAction(s *app.Session, act fileAction) {
	switch act.Kind {
	case dialogOpen:
		if _, err := s.OpenDocument(act.Path); err != nil {
			fmt.Fprintf(os.Stderr, "open %q: %v\n", act.Path, err)
		}
	case dialogSaveAs:
		if err := s.SaveActiveDocumentAs(act.Path); err != nil {
			fmt.Fprintf(os.Stderr, "save as %q: %v\n", act.Path, err)
		}
	case dialogLoadHDR:
		s.LoadEnvironmentFile(act.Path) // the decode happens lazily on the next viewport frame
	}
}

// saveActive saves the active document in place, falling back to the Save As modal when
// it has never been saved (a freshly created "PartN" has no path yet — app.ErrNeedsPath).
func saveActive(s *app.Session) {
	err := s.SaveActiveDocument()
	if errors.Is(err, app.ErrNeedsPath) {
		fileModal.openFor(dialogSaveAs)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
	}
}
