//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The 2D-sketch dynamic-input HUD (#790): a small heads-up panel that follows the cursor while
// a geometry tool draws, showing the next point's X/Y (or Length/Angle relative to the last
// point) and accepting typed precise input. The editing logic lives in app.Session
// (SketchHUDView + HUDInputRune/HUDTab/HUDCommit/…); this file only forwards keystrokes and
// paints the panel, per the head's thin-view rule. handleKeyboard yields plain keyboard to the
// HUD while it is engaged (s.HUDEngaged), so Enter commits the point and Esc clears the entry.

// hud panel geometry and colours.
const (
	hudCursorPadX   = 18 // panel offset right of the cursor (px)
	hudCursorPadY   = 8  // panel offset below the cursor (px)
	hudRowPadX      = 6  // horizontal text padding inside the panel
	hudRowPadY      = 3  // vertical text padding around the rows
	hudLabelGap     = 8  // gap between a field's label and its value
	hudFieldGap     = 16 // gap between the two fields on a row
	hudValueMinPx   = 36 // minimum value-column width so a 1-char value still reads as a field
	hudActiveAlpha  = 0.9
	hudPanelOpacity = 0.85
)

var (
	hudBgColor       = [4]float32{0.10, 0.11, 0.13, hudPanelOpacity}
	hudLabelColor    = [4]float32{0.62, 0.66, 0.72, 1}
	hudValueColor    = [4]float32{0.92, 0.94, 0.97, 1}
	hudActiveBgColor = [4]float32{0.20, 0.42, 0.78, hudActiveAlpha}
	hudActiveValueFg = [4]float32{1, 1, 1, 1}
)

// handleSketchHUD forwards keys to and paints the dynamic-input HUD for the viewport whose
// image origin is (bx,by). It is a no-op when the HUD is not applicable (disabled, no sketch,
// no coordinate tool, the cursor is off the plane) or the cursor is not over the viewport, so
// the panel never floats over the ribbon or the command window.
func handleSketchHUD(s *app.Session, bx, by float32, viewportHovered bool) {
	if !viewportHovered {
		return
	}
	mx, my := native.MousePos()
	cx, cy := float64(mx-bx), float64(my-by)
	view := s.SketchHUDView(cx, cy)
	if !view.Visible {
		return
	}
	if !native.WantTextInput() { // a focused text widget (the command line) keeps its own typing
		routeSketchHUDKeys(s, cx, cy)
		view = s.SketchHUDView(cx, cy) // re-read so the paint reflects this frame's keystrokes
		if !view.Visible {
			return
		}
	}
	drawSketchHUDPanel(view, mx, my)
}

// sketchEntrySession and placementKeySession are the session surfaces the two entry surfaces
// need (audit I5, the arrowSession pattern), so each router is testable with a small fake.
type sketchEntrySession interface {
	PlacementFieldEngaged() bool
	HUDEngaged() bool
}

// pointerInputSession is the coordinate panel's surface — the six methods that edit and
// commit a first-point entry.
type pointerInputSession interface {
	HUDInputRune(r rune)
	HUDTab()
	HUDBackspace()
	HUDEngaged() bool
	HUDCommit(px, py float64) error
	HUDCancel()
}

type placementKeySession interface {
	PlacementFieldInput(r rune)
	PlacementFieldTab()
	PlacementFieldBackspace()
	PlacementFieldEngaged() bool
	PlacementFieldCommit(px, py float64) error
	PlacementFieldCancel()
}

var (
	_ sketchEntrySession  = (*app.Session)(nil)
	_ placementKeySession = (*app.Session)(nil)
	_ pointerInputSession = (*app.Session)(nil)
)

// sketchEntryEngaged reports whether either entry surface has typed text, so the head keeps the
// keystrokes rather than letting them fall through to the viewport's shortcuts.
func sketchEntryEngaged(s sketchEntrySession) bool {
	return s.PlacementFieldEngaged() || s.HUDEngaged()
}

// routeSketchHUDKeys applies this frame's typed characters and editing keys. While a shape is
// being placed the in-place dimension fields take the keys (Tab locks a value, Enter finishes
// the shape); otherwise they go to the pointer-input panel, which places a shape's first point
// by coordinate. That split mirrors the reference application's separate pointer input and
// dimension input (#2014, #790).
func routeSketchHUDKeys(s *app.Session, cx, cy float64) {
	if len(s.PlacementFields()) > 0 {
		routePlacementFieldKeys(s, cx, cy)
		return
	}
	routePointerInputKeys(s, cx, cy)
}

// routePlacementFieldKeys drives the in-place dimension boxes: typing fills the active box, Tab
// locks it and moves on, Enter finishes the shape, Esc clears a started entry.
func routePlacementFieldKeys(s placementKeySession, cx, cy float64) {
	for _, r := range native.InputChars() {
		s.PlacementFieldInput(r)
	}
	keys := native.EditorKeysPressed()
	if keys.Tab {
		s.PlacementFieldTab()
	}
	if keys.Backspace {
		s.PlacementFieldBackspace()
	}
	if keys.Enter && s.PlacementFieldEngaged() {
		_ = s.PlacementFieldCommit(cx, cy) // a bad value leaves the strip open for correction
	}
	if keys.Escape && s.PlacementFieldEngaged() {
		s.PlacementFieldCancel()
	}
}

// routePointerInputKeys drives the coordinate panel for a shape's first point. Enter commits the
// resolved point; Esc clears a started entry (an un-started panel lets Esc fall through to
// cancel the tool); Tab cycles fields; Backspace edits the active field.
func routePointerInputKeys(s pointerInputSession, cx, cy float64) {
	for _, r := range native.InputChars() {
		s.HUDInputRune(r)
	}
	keys := native.EditorKeysPressed()
	if keys.Tab {
		s.HUDTab()
	}
	if keys.Backspace {
		s.HUDBackspace()
	}
	if keys.Enter && s.HUDEngaged() {
		_ = s.HUDCommit(cx, cy) // a parse error leaves the HUD open; the field stays for correction
	}
	if keys.Escape && s.HUDEngaged() {
		s.HUDCancel()
	}
}

// drawSketchHUDPanel paints the HUD panel just off the cursor at (mx,my): a row of "label
// value" pairs with the active field's value highlighted.
func drawSketchHUDPanel(view app.SketchHUDView, mx, my float32) {
	x0 := mx + hudCursorPadX
	y0 := my + hudCursorPadY
	h := native.TextLineHeight() + 2*hudRowPadY
	w := hudPanelWidth(view)
	native.DrawRectFilled(x0, y0, x0+w, y0+h, hudBgColor)
	drawHUDFields(view, x0+hudRowPadX, y0+hudRowPadY)
}

// hudPanelWidth measures the panel: both fields' "label value" widths plus the paddings/gaps.
func hudPanelWidth(view app.SketchHUDView) float32 {
	total := float32(2 * hudRowPadX)
	for i := 0; i < len(view.Labels); i++ {
		total += hudFieldWidth(view.Labels[i], hudValueText(view, i))
		if i == 0 {
			total += hudFieldGap
		}
	}
	return total
}

// drawHUDFields paints each "label value(unit)" pair left to right from (x,y), highlighting
// the active field's value cell.
func drawHUDFields(view app.SketchHUDView, x, y float32) {
	for i := 0; i < len(view.Labels); i++ {
		value := hudValueText(view, i)
		native.DrawText(x, y, view.Labels[i], hudLabelColor)
		vx := x + native.CalcTextWidth(view.Labels[i]) + hudLabelGap
		vw := valueCellWidth(value)
		if view.Engaged && i == view.Active {
			native.DrawRectFilled(vx-2, y-1, vx+vw+2, y+native.TextLineHeight()+1, hudActiveBgColor)
			native.DrawText(vx, y, value, hudActiveValueFg)
		} else {
			native.DrawText(vx, y, value, hudValueColor)
		}
		x = vx + vw + hudFieldGap
	}
}

// hudValueText is the displayed value for field i, suffixing the length unit on the coordinate
// and length fields (not the angle field, which is in degrees).
func hudValueText(view app.SketchHUDView, i int) string {
	if view.Mode == app.HUDPolar && i == 1 {
		return view.Values[i] + "°" // angle in degrees
	}
	return view.Values[i] + " " + view.Unit
}

// hudFieldWidth is the rendered width of a "label value" pair.
func hudFieldWidth(label, value string) float32 {
	return native.CalcTextWidth(label) + hudLabelGap + valueCellWidth(value)
}

// valueCellWidth is the value column's width, floored so a short value still reads as a field.
func valueCellWidth(value string) float32 {
	w := native.CalcTextWidth(value)
	if w < hudValueMinPx {
		return hudValueMinPx
	}
	return w
}
