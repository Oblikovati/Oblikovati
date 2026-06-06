// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/model/doc"
)

// closeGuard is the head's graceful-close state: whether the "save changes?" prompt
// is currently showing. It is cgo-free so the prompt/exit logic is unit-testable; the
// modal that drives it lives in close_guard_draw.go (the file_dialog split, ADR-0014).
type closeGuard struct {
	prompting bool
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
