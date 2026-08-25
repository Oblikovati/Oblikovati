//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
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

// openMarkingMenu arms the popup (called on a viewport right-click).
func openMarkingMenu() { native.OpenPopup(markingMenuPopupID) }

// markingMenuRequested is set by the viewport's right-click handler (inside the Viewport
// window) and consumed here at the top level, so OpenPopup and BeginPopup run in the same
// ImGui window context (the appMenuRequested pattern, chrome.go). Without this deferral the
// popup opens in the Viewport's popup stack and the top-level BeginPopup never sees it.
var markingMenuRequested bool

// openMarkingMenuOnFirstFrame lets the visual harness show the menu without a
// synthetic right-click; consumed on the first chrome frame.
var openMarkingMenuOnFirstFrame bool

// drawMarkingMenu renders the right-click popup when open, in the user's chosen style: the radial
// ring (default) or the classic linear menu (#915 C8). Both lead with the idle "Repeat <command>"
// entry (#915 C5) and end with the style toggle. Unknown command ids are skipped, so a customized
// menu can name commands before they register.
func drawMarkingMenu(s *app.Session) {
	if openMarkingMenuOnFirstFrame || markingMenuRequested {
		openMarkingMenuOnFirstFrame = false
		markingMenuRequested = false
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
	drawActiveToolMenuOptions(s)
	menu := s.MarkingMenu(app.CurrentEnvironment(s))
	layout := markingRingLayoutForMenu(s, menu)
	ringX, ringY := native.GetCursorPos()
	native.Dummy(layout.size, layout.size) // reserves space without capturing slot clicks
	for _, item := range menu.Quadrants {
		drawMarkingSlot(s, ringX, ringY, layout, item)
	}
	// Slots are positioned manually, so restore the layout cursor after the reserved ring before
	// drawing overflow and the style row. Otherwise those rows start at whichever slot happened
	// to be last in the persisted slice and can overlap the ring (the observed right-click defect).
	m := native.Metrics()
	native.SetCursorPos(ringX, ringY+layout.size+m.ItemSpacingY)
	drawMarkingOverflow(s, menu.Overflow)
	drawContextMenuStyleToggle(s)
}

// markingMenuCommandHost is the narrow command lookup seam used while measuring the ring.
// Keeping the measurement helper on this interface preserves the head/ui coupling ratchet.
type markingMenuCommandHost interface {
	Commands() *app.CommandManager
}

var _ markingMenuCommandHost = (*app.Session)(nil)

// markingRingLayoutForMenu measures the commands that will actually render. Unknown
// command ids are intentionally ignored, matching drawMarkingSlot's fallback.
func markingRingLayoutForMenu(s markingMenuCommandHost, menu wire.MarkingMenuView) markingRingLayout {
	m := native.Metrics()
	iconPx := scaledIconPx(smallIconPx)
	var maxWidth, maxHeight float32
	for _, item := range menu.Quadrants {
		cmd, ok := s.Commands().ByID(item.CommandID)
		if !ok {
			continue
		}
		hasIcon := false
		if icons != nil {
			_, hasIcon = icons.texture(cmd.Icon(), cmd.InlineIconSVG(), iconPx)
		}
		w, h := markingSlotExtent(cmd.DisplayName(), hasIcon, iconPx, m)
		if w > maxWidth {
			maxWidth = w
		}
		if h > maxHeight {
			maxHeight = h
		}
	}
	return newMarkingRingLayout(maxWidth, maxHeight)
}

// drawClassicContextMenu draws the same commands as a plain vertical list — the classic
// (non-radial) right-click menu (#915 C8).
func drawClassicContextMenu(s *app.Session) {
	drawRepeatEntry(s)
	drawActiveToolMenuOptions(s)
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

// drawActiveToolMenuOptions renders the active tool's in-command toggle options (Inventor's
// right-click options, e.g. the Offset tool's Loop Select / Constrain Offset) as checkable rows at
// the top of the context menu. A checkmark prefix shows an on option; clicking one flips it and
// closes the menu, as Inventor does. Nothing renders when no tool offers options.
func drawActiveToolMenuOptions(s *app.Session) {
	opts := s.ActiveToolMenuOptions()
	if len(opts) == 0 {
		return
	}
	for _, opt := range opts {
		label := opt.Label
		if opt.Checked {
			label = "✓ " + opt.Label
		}
		if native.MenuItem(label + "##tool-opt-" + opt.Label) {
			s.ToggleActiveToolMenuOption(opt.Label)
			native.CloseCurrentPopup()
		}
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

// drawMarkingSlot places one quadrant's command button on the ring. When the command has an
// icon asset, it draws icon + label side by side (the same composition as the ribbon's small
// labelled icon buttons, drawLabeledIconButton, chrome_ribbon.go:613-624 — the icon is the click
// target there too); a command with no icon falls back to the original text-only button.
func drawMarkingSlot(s *app.Session, ringX, ringY float32, layout markingRingLayout, item wire.MarkingMenuItem) {
	cmd, ok := s.Commands().ByID(item.CommandID)
	if !ok {
		return
	}
	label := cmd.DisplayName()
	iconPx := scaledIconPx(smallIconPx)
	tex, hasIcon := icons.texture(cmd.Icon(), cmd.InlineIconSVG(), iconPx)
	m := native.Metrics()
	w, h := markingSlotExtent(label, hasIcon, iconPx, m)
	x, y := markingSlotPosition(layout, item.Quadrant, w, h)
	native.SetCursorPos(ringX+x, ringY+y)
	native.BeginDisabled(!cmd.IsEnabled(s))
	if drawMarkingSlotButton(item, label, tex, hasIcon, iconPx, m) {
		_ = s.Execute(item.CommandID)
		native.CloseCurrentPopup()
	}
	native.EndDisabled()
}

// drawMarkingSlotButton draws a marking-slot's clickable control — an icon beside its label when the
// command has an icon asset, else a plain text button — and reports whether it was clicked.
func drawMarkingSlotButton(item wire.MarkingMenuItem, label string, tex uint64, hasIcon bool, iconPx int, m native.StyleMetrics) bool {
	if !hasIcon {
		return native.Button(label + "##mm-" + item.CommandID)
	}
	native.BeginGroup()
	clicked := native.ImageButton("##mm-"+item.CommandID, tex, float32(iconPx), float32(iconPx), identityTint)
	native.SameLine()
	cx, cy := native.GetCursorScreenPos()
	native.SetCursorScreenPos(cx, cy+(float32(iconPx)+2*m.FramePadY-native.TextLineHeight())/2)
	native.Text(label)
	native.EndGroup()
	return clicked
}

// markingSlotExtent mirrors the ImGui primitives used by drawMarkingSlot, so the ring can
// reserve enough room before it places any command. ImageButton includes frame padding; the
// label is a plain text item separated by the live item spacing.
func markingSlotExtent(label string, hasIcon bool, iconPx int, m native.StyleMetrics) (float32, float32) {
	textWidth := native.CalcTextWidth(label)
	textHeight := native.FrameHeight()
	if !hasIcon {
		return textWidth + 2*m.FramePadX, textHeight
	}
	iconWidth := float32(iconPx) + 2*m.FramePadX
	iconHeight := float32(iconPx) + 2*m.FramePadY
	width := iconWidth + m.ItemSpacingX + textWidth
	if textHeight > iconHeight {
		iconHeight = textHeight
	}
	return width, iconHeight
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
