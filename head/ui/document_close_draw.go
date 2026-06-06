//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// drawDocumentClosePrompt asks how to handle one dirty document whose tab close button
// was clicked. The prompt remains open while Save As is resolving a never-saved file.
func drawDocumentClosePrompt(s *app.Session) {
	if closeDocumentModal.closeIfClean(s) || closeDocumentModal.pending == nil {
		return
	}
	d := closeDocumentModal.pending
	if native.Begin("Save document changes?##document-close") {
		native.Text("Save changes to " + d.FullDocumentName() + " before closing?")
		if native.Button("Save") {
			savePendingDocumentClose(s)
		}
		native.SameLine()
		if native.Button("Don't Save") {
			closeDocumentModal.discard(s)
		}
		native.SameLine()
		if native.Button("Cancel") {
			closeDocumentModal.cancel()
		}
	}
	native.End()
}

func savePendingDocumentClose(s *app.Session) {
	d := closeDocumentModal.pending
	if d == nil {
		return
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		fmt.Fprintf(os.Stderr, "activate %q for close save: %v\n", d.FullDocumentName(), err)
		return
	}
	if documentNeedsSaveAs(d) {
		fileModal.openFor(dialogSaveAs)
		return
	}
	saveActive(s)
	closeDocumentModal.closeIfClean(s)
}
