//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"slices"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/doc"
)

// historyBrowserSelection remembers which documents' timelines the History Browser shows side
// by side, keyed by document id. It is UI-only state (the model owns the timelines), kept
// across frames so the user's column choices stick while the window is open.
var historyBrowserSelection = map[doc.ID]bool{}

// historyDoc is one open document offered in the History Browser: its id and display name.
type historyDoc struct {
	id   doc.ID
	name string
}

// drawHistoryBrowserWindow renders the Edit ▸ History Browser while it is open. It complements
// the undo/redo buttons for long histories and multi-document assemblies: a checkbox list picks
// documents, then one timeline column per picked document shows every step since the document
// opened — the current state highlighted, save checkpoints flagged "*" — with click-to-jump.
// Reads go through DocumentHistoryView and jumps through JumpDocumentTo (the same surface the
// API uses), so a background part's timeline navigates without activating it.
func drawHistoryBrowserWindow(s *app.Session) {
	if !s.HistoryBrowserOpen() {
		return
	}
	native.SetNextWindowSizeOnce(560, 440)
	if native.Begin("History Browser") {
		docs := historyDocuments(s)
		if len(docs) == 0 {
			native.Text("No open documents with an editable history.")
		} else {
			drawHistoryDocPicker(docs)
			native.Separator()
			native.Text("* = saved to disk    (highlighted row = current state)")
			drawHistoryColumns(s, selectedHistoryDocs(docs))
		}
		native.Separator()
		if native.Button("Close") {
			s.CloseHistoryBrowser()
		}
	}
	native.End()
}

// historyDocuments lists the open documents that have a navigable timeline (parts, assemblies,
// drawings — anything DocumentHistoryView recognises), in tab order.
func historyDocuments(s *app.Session) []historyDoc {
	var out []historyDoc
	for _, d := range s.Workspace().VisibleDocuments() {
		if _, ok := s.DocumentHistoryView(d.ID()); ok {
			out = append(out, historyDoc{id: d.ID(), name: d.DisplayName()})
		}
	}
	return out
}

// drawHistoryDocPicker renders the per-document checkboxes, defaulting to one selection so a
// column is always shown.
func drawHistoryDocPicker(docs []historyDoc) {
	ensureHistorySelection(docs)
	for _, d := range docs {
		sel := historyBrowserSelection[d.id]
		native.Checkbox(fmt.Sprintf("%s##histsel%d", d.name, uint64(d.id)), &sel)
		historyBrowserSelection[d.id] = sel
	}
}

// ensureHistorySelection selects the first document when none of the currently open documents
// is selected, so the browser never shows an empty body.
func ensureHistorySelection(docs []historyDoc) {
	for _, d := range docs {
		if historyBrowserSelection[d.id] {
			return
		}
	}
	if len(docs) > 0 {
		historyBrowserSelection[docs[0].id] = true
	}
}

// selectedHistoryDocs returns the picked documents among those currently open.
func selectedHistoryDocs(docs []historyDoc) []historyDoc {
	out := make([]historyDoc, 0, len(docs))
	for _, d := range docs {
		if historyBrowserSelection[d.id] {
			out = append(out, d)
		}
	}
	return out
}

// drawHistoryColumns lays the selected documents' timelines side by side, one table column each.
func drawHistoryColumns(s *app.Session, docs []historyDoc) {
	if len(docs) == 0 {
		native.Text("Tick a document above to show its timeline.")
		return
	}
	if !native.BeginTable("##histcols", len(docs), 0, 0) {
		return
	}
	for _, d := range docs {
		native.TableSetupColumn(d.name)
	}
	native.TableHeadersRow()
	native.TableNextRow()
	for _, d := range docs {
		native.TableNextColumn()
		drawTimelineColumn(s, d.id)
	}
	native.EndTable()
}

// drawTimelineColumn renders one document's whole stream in a scrollable child: row 0 is the
// open/baseline state and row k the state after step k, so clicking row k jumps the cursor to
// position k. The current position is highlighted; saved checkpoints are prefixed "*".
func drawTimelineColumn(s *app.Session, id doc.ID) {
	tl, ok := s.DocumentHistoryView(id)
	if !ok {
		native.Text("(closed)")
		return
	}
	native.PushID(fmt.Sprintf("histcol%d", uint64(id)))
	defer native.PopID()
	if native.BeginChild("##timeline", 0, 300, true) {
		if native.Selectable("Open##pos0", tl.Position == 0) {
			_ = s.JumpDocumentTo(id, 0)
		}
		for i, label := range tl.Labels {
			pos := i + 1
			if native.Selectable(historyRowLabel(label, pos, tl), tl.Position == pos) {
				_ = s.JumpDocumentTo(id, pos)
			}
		}
	}
	native.EndChild()
}

// historyRowLabel builds a row label: a "*" for a save checkpoint, the step name, and a unique
// "##pos" id (step names repeat, e.g. several "Edit Parameters", so the visible label is not a
// usable widget id on its own).
func historyRowLabel(label string, pos int, tl app.DocumentTimeline) string {
	mark := "  "
	if slices.Contains(tl.SavedDepths, pos) {
		mark = "* "
	}
	return fmt.Sprintf("%s%s##pos%d", mark, label, pos)
}
