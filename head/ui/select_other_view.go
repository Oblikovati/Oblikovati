//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Select Other (#910): a right-click on stacked geometry begins a cycle through the occluded
// candidates instead of opening the marking menu. A small widget near the cursor steps through
// them (each is selected as you go), and committing keeps the chosen one. The app.Session owns the
// cycle; this file is just the trigger + the widget.

// selectOtherScreenX/Y anchor the cycle widget at the press position.
var selectOtherScreenX, selectOtherScreenY float32

// handleViewportRightClick routes a viewport right-click: on stacked geometry it digs through the
// occluded objects (Select Other, #910); otherwise it opens the radial marking menu (M05-F12).
// Must be called right after the viewport's InvisibleButton.
func handleViewportRightClick(s *app.Session) {
	if !native.IsItemClicked(native.MouseRight) {
		return
	}
	lx, ly := viewportCursor()
	if !beginSelectOtherAt(s, lx, ly) {
		openMarkingMenu()
	}
}

// beginSelectOtherAt starts the cycle at a viewport pixel and, when it starts (more than one
// object stacks up), remembers the screen position for the widget. Returns whether it started, so
// the caller falls back to the marking menu when there is nothing occluded.
func beginSelectOtherAt(s *app.Session, lx, ly float64) bool {
	if !s.BeginSelectOther(lx, ly) {
		return false
	}
	selectOtherScreenX, selectOtherScreenY = native.MousePos()
	return true
}

// drawSelectOtherWidget draws the cycle control near the cursor while Select Other is active:
// Next steps one object deeper (wrapping), Prev steps back, and Select commits the highlighted
// one. Esc (keyboard path) also commits.
func drawSelectOtherWidget(s *app.Session) {
	if !s.SelectOtherActive() {
		return
	}
	pos, count := s.SelectOtherStatus()
	native.SetNextWindowPos(selectOtherScreenX+12, selectOtherScreenY+12)
	if native.Begin("Select Other###select-other") {
		if native.Button("Prev##so-prev") {
			s.CycleSelectOther(-1)
		}
		native.SameLine()
		native.Text(strconv.Itoa(pos) + " / " + strconv.Itoa(count))
		native.SameLine()
		if native.Button("Next##so-next") {
			s.CycleSelectOther(1) // deeper into the occluded stack
		}
		native.SameLine()
		if native.Button("Select##so-ok") {
			s.CommitSelectOther()
		}
	}
	native.End()
}
