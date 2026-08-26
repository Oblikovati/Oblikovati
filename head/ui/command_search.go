//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The command search box (M05-F12): a query field in the menu bar; matches drop
// down beneath it and a click runs the command. Backed by the same
// session.SearchCommands the ui.search wire method serves.

// commandSearchBuf is the box's edit buffer (UI state, head-owned).
var commandSearchBuf [64]byte

// drawCommandSearch renders the search field inside the main menu bar and its
// results popup while a query is typed.
func drawCommandSearch(s *app.Session) {
	native.SetNextItemWidth(200)
	native.InputText("##command-search", commandSearchBuf[:])
	query := bufString(commandSearchBuf[:])
	if query == "" {
		return
	}
	hits := s.SearchCommands(query)
	if len(hits) == 0 {
		native.SetItemTooltip("No matching commands")
		return
	}
	drawSearchResults(s, hits)
}

// drawSearchResults lists the matches right under the field; clicking one runs it
// and clears the query.
func drawSearchResults(s *app.Session, hits []*app.CommandDefinition) {
	x, y := native.ItemRectMin()
	_, h := native.ItemRectMax()
	_ = h
	native.SetNextWindowPos(x, y+native.FrameHeight())
	if native.Begin("Search results###cmd-search-results") {
		limit := min(len(hits), 10)
		for _, cmd := range hits[:limit] {
			native.BeginDisabled(!cmd.IsEnabled(s))
			if native.Selectable(cmd.DisplayName()+"##sr-"+cmd.ID(), false) {
				_ = s.Execute(cmd.ID())
				commandSearchBuf = [64]byte{}
			}
			native.EndDisabled()
		}
	}
	native.End()
}
