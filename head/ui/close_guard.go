// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// closeGuard is the head's graceful-close state: whether the "save changes?" prompt
// is currently showing. It is cgo-free so the prompt/exit logic is unit-testable; the
// modal that drives it lives in close_guard_draw.go (the file_dialog split, ADR-0014).
type closeGuard struct {
	prompting bool
}

// documentCloseGuard tracks the document whose tab-close request is waiting for a
// Save / Don't Save / Cancel answer.
type documentCloseGuard struct {
	pending *doc.Document
}

func (g *documentCloseGuard) request(d *doc.Document) bool {
	if d == nil || !d.Dirty() {
		return true
	}
	g.pending = d
	return false
}

func (g *documentCloseGuard) cancel() { g.pending = nil }

func (g *documentCloseGuard) discard(s *app.Session) {
	if g.pending == nil {
		return
	}
	closeDocumentNow(s, g.pending, true)
	g.pending = nil
}

func (g *documentCloseGuard) closeIfClean(s *app.Session) bool {
	if g.pending == nil || g.pending.Dirty() {
		return false
	}
	closeDocumentNow(s, g.pending, false)
	g.pending = nil
	return true
}

// dirtyDocuments returns the open documents with unsaved changes — the ones a close
// must warn about. Order follows the workspace's stable enumeration.
func dirtyDocuments(s *app.Session) []*doc.Document {
	var out []*doc.Document
	for _, d := range s.Workspace().Documents() {
		if d.Dirty() {
			out = append(out, d)
		}
	}
	return out
}
