//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/head/internal/native"
)

// toggleFind opens or closes the find/replace bar. Opening it recomputes matches for whatever is
// already in the query box (and selects the first), so reopening after an edit is immediately useful.
func (e *codeEditor) toggleFind() {
	e.find.active = !e.find.active
	if e.find.active {
		e.recomputeMatches()
	}
}

// DrawFindBar renders the find/replace row above the editor when open: a query field (live
// match count + Next/Prev) and a replace field with Replace All. The console panel calls it
// before the editor body. It is a no-op when the bar is closed.
func (e *codeEditor) DrawFindBar() {
	if !e.find.active {
		return
	}
	native.SetNextItemWidth(220)
	if native.InputText("##find-query", e.find.query) {
		e.recomputeMatches()
	}
	native.SameLine()
	native.Text(fmt.Sprintf("%d/%d", e.matchOrdinal(), len(e.find.matches)))
	native.SameLine()
	if native.Button("Prev##find") {
		e.stepMatch(-1)
	}
	native.SameLine()
	if native.Button("Next##find") {
		e.stepMatch(1)
	}
	native.SetNextItemWidth(220)
	native.InputText("##find-replace", e.find.replace)
	native.SameLine()
	if native.Button("Replace All##find") {
		e.replaceAllMatches()
	}
}

// recomputeMatches re-runs the search for the current query and selects the first hit.
func (e *codeEditor) recomputeMatches() {
	e.find.matches = e.model.Find(bufString(e.find.query))
	e.find.index = 0
	if len(e.find.matches) > 0 {
		e.model.SelectMatch(e.find.matches[0])
		e.focused = true
	}
}

// stepMatch advances the selection to the next/previous match, wrapping around.
func (e *codeEditor) stepMatch(delta int) {
	n := len(e.find.matches)
	if n == 0 {
		return
	}
	e.find.index = (e.find.index + delta + n) % n
	e.model.SelectMatch(e.find.matches[e.find.index])
	e.focused = true
}

// matchOrdinal is the 1-based index of the current match for the "k/n" readout (0 when none).
func (e *codeEditor) matchOrdinal() int {
	if len(e.find.matches) == 0 {
		return 0
	}
	return e.find.index + 1
}

// replaceAllMatches replaces every occurrence of the query with the replacement text and
// refreshes the match list (now empty unless the replacement contains the query).
func (e *codeEditor) replaceAllMatches() {
	e.model.ReplaceAll(bufString(e.find.query), bufString(e.find.replace))
	e.recomputeMatches()
}
