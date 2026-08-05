// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/doc"
)

// CloseDocument closes d after releasing the session state that points into it. Every close
// path goes through here rather than calling Workspace().Close directly: closing on the
// workspace alone left the armed tool running and s.activeSketch dangling into the destroyed
// document, so the commit bar and the contextual Sketch tab survived the close and sketch
// commands kept answering ok while mutating an orphaned sketch (#2040).
//
//	if err := s.CloseDocument(s.ActiveDocument(), true); err != nil { ... }
func (s *Session) CloseDocument(d *doc.Document, skipSave bool) error {
	if d == nil {
		return errors.New("app: CloseDocument: nil document, expected an open document")
	}
	// Save before releasing edit state, not after: a failed save leaves the document open, and
	// the user's armed tool and sketch edit have to survive with it.
	if !skipSave && d.Dirty() {
		if err := s.workspace.Save(d); err != nil {
			return err
		}
	}
	s.releaseEditStateFor(d)
	return s.workspace.Close(d, skipSave)
}

// releaseEditStateFor abandons the armed tool, leaves the sketch environment and drops the
// selection when they belong to the document being closed. Closing a BACKGROUND document must
// leave a tool armed against the still-open document alone, so this is a no-op unless d is the
// active document — the only document a tool can be armed in or a sketch edit open in.
func (s *Session) releaseEditStateFor(d *doc.Document) {
	if d != s.ActiveDocument() {
		return
	}
	s.abandonTool()
	// The end-of-part marker the sketch edit rolled back dies with the document, so drop the
	// scope before exiting rather than letting endEditScope restore it onto whichever document
	// the workspace activates next.
	s.editScope = editScope{}
	s.ExitSketch()
	s.ExitSketch3D()
	// Selected refs point into the document being destroyed; closing emits no DocumentActivate,
	// so nothing else would clear them (compare watchDocumentSwitches, #1105).
	s.selection.Clear()
}
