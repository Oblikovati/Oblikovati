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

// drawMarkingMenu renders the popup when open: the ring for the active
// environment, then the overflow rows. Unknown command ids are skipped, so a
// customized menu can name commands before they register.
func drawMarkingMenu(s *app.Session) {
	if openMarkingMenuOnFirstFrame {
		openMarkingMenuOnFirstFrame = false
		openMarkingMenu()
	}
	if !native.BeginPopup(markingMenuPopupID) {
		return
	}
	menu := s.MarkingMenu(app.CurrentEnvironment(s))
	const size = 2*markingMenuRadius + 96 // ring + button width margin
	centerX, centerY := float32(size)/2, float32(size)/2
	native.InvisibleButton("##marking-area", size, size) // reserves the ring's space
	for _, item := range menu.Quadrants {
		drawMarkingSlot(s, centerX, centerY, item)
	}
	drawMarkingOverflow(s, menu.Overflow)
	native.EndPopup()
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
