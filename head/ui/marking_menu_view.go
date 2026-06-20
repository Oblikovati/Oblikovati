//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The radial marking menu (M05-F12): right-clicking the viewport opens the active
// environment's eight-quadrant command ring (plus its linear overflow) as a popup
// at the cursor. Picking a slot runs the command; clicking away dismisses — the
// standard ImGui popup behavior.

// markingMenuPopupID names the popup; the viewport's right-click opens it.
const markingMenuPopupID = "##marking-menu"

// markingMenuRadius is the ring's pixel radius around the popup center.
const markingMenuRadius = 72

// openMarkingMenu arms the popup (called on a viewport right-click).
func openMarkingMenu() { native.OpenPopup(markingMenuPopupID) }

// openMarkingMenuOnFirstFrame lets the visual harness show the menu without a
// synthetic right-click; consumed on the first chrome frame.
var openMarkingMenuOnFirstFrame bool

// quadrantDirections maps each compass slot to its unit offset (screen y grows
// downward, so north is -y).
var quadrantDirections = map[types.ScreenQuadrant][2]float32{
	types.QuadrantNorth: {0, -1}, types.QuadrantNorthEast: {0.7, -0.7},
	types.QuadrantEast: {1, 0}, types.QuadrantSouthEast: {0.7, 0.7},
	types.QuadrantSouth: {0, 1}, types.QuadrantSouthWest: {-0.7, 0.7},
	types.QuadrantWest: {-1, 0}, types.QuadrantNorthWest: {-0.7, -0.7},
}

// drawMarkingMenu renders the right-click popup when open, in the user's chosen style: the radial
// ring (default) or the classic linear menu (#915 C8). Both lead with the idle "Repeat <command>"
// entry (#915 C5) and end with the style toggle. Unknown command ids are skipped, so a customized
// menu can name commands before they register.
func drawMarkingMenu(s *app.Session) {
	if openMarkingMenuOnFirstFrame {
		openMarkingMenuOnFirstFrame = false
		openMarkingMenu()
	}
	if !native.BeginPopup(markingMenuPopupID) {
		return
	}
	if s.ClassicContextMenu() {
		drawClassicContextMenu(s)
	} else {
		drawRadialMarkingMenu(s)
	}
	native.EndPopup()
}

// drawRadialMarkingMenu draws the eight-quadrant ring plus its overflow rows.
func drawRadialMarkingMenu(s *app.Session) {
	drawRepeatEntry(s)
	menu := s.MarkingMenu(app.CurrentEnvironment(s))
	const size = 2*markingMenuRadius + 96 // ring + button width margin
	centerX, centerY := float32(size)/2, float32(size)/2
	native.InvisibleButton("##marking-area", size, size) // reserves the ring's space
	for _, item := range menu.Quadrants {
		drawMarkingSlot(s, centerX, centerY, item)
	}
	drawMarkingOverflow(s, menu.Overflow)
	drawContextMenuStyleToggle(s)
}

// drawClassicContextMenu draws the same commands as a plain vertical list — the classic
// (non-radial) right-click menu (#915 C8).
func drawClassicContextMenu(s *app.Session) {
	drawRepeatEntry(s)
	menu := s.MarkingMenu(app.CurrentEnvironment(s))
	for _, item := range menu.Quadrants {
		drawLinearCommandEntry(s, item.CommandID)
	}
	for _, id := range menu.Overflow {
		drawLinearCommandEntry(s, id)
	}
	drawContextMenuStyleToggle(s)
}

// drawRepeatEntry renders the idle "Repeat <command>" entry at the top of the menu when a prior
// command exists and no tool is active (#915 C5).
func drawRepeatEntry(s *app.Session) {
	label, _, ok := s.RepeatMenuEntry()
	if !ok {
		return
	}
	if native.Selectable(label+"##mm-repeat", false) {
		_ = s.RepeatLastCommand()
		native.CloseCurrentPopup()
	}
	native.Separator()
}

// drawContextMenuStyleToggle renders the entry that switches between the radial and classic
// styles (#915 C8).
func drawContextMenuStyleToggle(s *app.Session) {
	native.Separator()
	label := "Use Classic Menu"
	if s.ClassicContextMenu() {
		label = "Use Marking Menu"
	}
	if native.Selectable(label+"##mm-style", false) {
		s.ToggleContextMenuStyle()
		native.CloseCurrentPopup()
	}
}

// drawLinearCommandEntry renders one command as a vertical menu row (greyed when disabled,
// skipped when unknown), running it and closing the popup on click.
func drawLinearCommandEntry(s *app.Session, id string) {
	cmd, ok := s.Commands().ByID(id)
	if !ok {
		return
	}
	native.BeginDisabled(!cmd.IsEnabled(s))
	if native.Selectable(cmd.DisplayName()+"##mmc-"+id, false) {
		_ = s.Execute(id)
		native.CloseCurrentPopup()
	}
	native.EndDisabled()
}

// drawMarkingSlot places one quadrant's command button on the ring.
func drawMarkingSlot(s *app.Session, centerX, centerY float32, item wire.MarkingMenuItem) {
	cmd, ok := s.Commands().ByID(item.CommandID)
	if !ok {
		return
	}
	dir := quadrantDirections[item.Quadrant]
	label := cmd.DisplayName()
	w := native.CalcTextWidth(label) + 16
	x := centerX + dir[0]*markingMenuRadius - w/2
	y := centerY + dir[1]*markingMenuRadius - 12
	native.SetCursorPos(x, y)
	native.BeginDisabled(!cmd.IsEnabled(s))
	if native.Button(label + "##mm-" + item.CommandID) {
		_ = s.Execute(item.CommandID)
		native.CloseCurrentPopup()
	}
	native.EndDisabled()
}

// drawMarkingOverflow renders the linear rows beneath the ring.
func drawMarkingOverflow(s *app.Session, overflow []string) {
	if len(overflow) == 0 {
		return
	}
	native.Separator()
	for _, id := range overflow {
		cmd, ok := s.Commands().ByID(id)
		if !ok {
			continue
		}
		if native.Selectable(cmd.DisplayName()+"##mmo-"+id, false) {
			_ = s.Execute(id)
		}
	}
}
