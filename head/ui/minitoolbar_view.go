//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	gomath "oblikovati.org/math"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// In-canvas mini-toolbars (M05-F07): each visible declared toolbar renders as a
// small floating window over the viewport — at its projected model anchor, or at
// viewport-relative pixels — with its declared controls and OK/Apply/Cancel row.
// User edits flow back through the session, so the owner receives the
// miniToolbar.changed / .committed events.

// drawMiniToolbars renders the visible toolbars; originX/Y is the viewport image's
// screen origin (anchors project relative to it).
func drawMiniToolbars(s *app.Session, cam scene.Camera, originX, originY float32) {
	for _, tb := range s.MiniToolbars().List() {
		if tb.Visible {
			drawMiniToolbar(s, cam, originX, originY, tb)
		}
	}
}

// drawMiniToolbar renders one toolbar. Closing the window's X is a cancel.
func drawMiniToolbar(s *app.Session, cam scene.Camera, originX, originY float32, tb wire.MiniToolbarSpec) {
	x, y, visible := miniToolbarScreenPos(cam, originX, originY, tb)
	if !visible {
		return // anchored behind the camera: nothing sensible to show
	}
	native.SetNextWindowPos(x, y)
	title := tb.HeadsUpText
	if title == "" {
		title = tb.ID
	}
	shown, open := native.BeginClosable(title + "###minitoolbar-" + tb.ID)
	if shown {
		for i, control := range tb.Controls {
			native.PushIDInt(i)
			drawMiniToolbarControl(s, tb.ID, control)
			native.PopID()
		}
		drawMiniToolbarCommitRow(s, tb)
	}
	native.End()
	if !open {
		_ = s.CommitMiniToolbar(tb.ID, app.MiniToolbarCancel)
	}
}

// miniToolbarScreenPos resolves the toolbar's screen position: the projected model
// anchor when one is declared, else the viewport-relative pixel offset.
func miniToolbarScreenPos(cam scene.Camera, originX, originY float32, tb wire.MiniToolbarSpec) (float32, float32, bool) {
	if tb.Anchor == nil {
		return originX + float32(tb.ScreenX), originY + float32(tb.ScreenY), true
	}
	p := gomath.P3(tb.Anchor[0], tb.Anchor[1], tb.Anchor[2])
	sx, sy, ok := renderer.Project(cam, viewportNear, viewportFar, p)
	if !ok {
		return 0, 0, false
	}
	return originX + float32(sx), originY + float32(sy), true
}

// drawMiniToolbarControl renders one control by kind, reporting edits through the
// session so the owner sees miniToolbar.changed.
func drawMiniToolbarControl(s *app.Session, toolbarID string, control wire.MiniToolbarControlSpec) {
	changed := false
	switch control.Kind {
	case types.MiniToolbarCheckbox:
		changed = native.Checkbox(control.Label, &control.Checked)
	case types.MiniToolbarCombo:
		changed = drawMiniToolbarCombo(&control)
	case types.MiniToolbarSlider:
		v := float32(control.Number)
		if native.SliderFloat(control.Label, &v, float32(control.Min), float32(control.Max)) {
			control.Number = roundSlider(float64(v))
			changed = true
		}
	case types.MiniToolbarTextBox, types.MiniToolbarValueEditor:
		changed = drawMiniToolbarText(&control, false)
	case types.MiniToolbarTextEditor:
		changed = drawMiniToolbarText(&control, true)
	default: // MiniToolbarButton
		changed = native.Button(control.Label)
	}
	if control.Tooltip != "" {
		native.SetItemTooltip(control.Tooltip)
	}
	if changed {
		_ = s.ChangeMiniToolbarControl(toolbarID, control)
	}
}

// drawMiniToolbarCombo renders the pick-one dropdown, writing the selection back.
func drawMiniToolbarCombo(control *wire.MiniToolbarControlSpec) bool {
	preview := ""
	if control.Selected >= 0 && control.Selected < len(control.Options) {
		preview = control.Options[control.Selected]
	}
	changed := false
	if native.BeginCombo(control.Label, preview) {
		for i, option := range control.Options {
			if native.Selectable(option+"##opt-"+strconv.Itoa(i), i == control.Selected) {
				control.Selected = i
				changed = true
			}
		}
		native.EndCombo()
	}
	return changed
}

// drawMiniToolbarText renders a single- or multi-line text input over the control's
// value, reporting the edit when the field is left (deactivated after edit), so a
// keystroke storm does not flood the owner.
func drawMiniToolbarText(control *wire.MiniToolbarControlSpec, multiline bool) bool {
	buf := make([]byte, 256)
	copy(buf, control.Value)
	if multiline {
		native.InputTextMultiline(control.Label, buf, 0, 60)
	} else {
		native.InputText(control.Label, buf)
	}
	if !native.IsItemDeactivatedAfterEdit() {
		return false
	}
	control.Value = cString(buf)
	return true
}

// drawMiniToolbarCommitRow renders the OK/Apply/Cancel row per the spec's flags.
func drawMiniToolbarCommitRow(s *app.Session, tb wire.MiniToolbarSpec) {
	if !tb.ShowOK && !tb.ShowApply && !tb.ShowCancel {
		return
	}
	native.Separator()
	drawCommitButton(s, tb.ID, tb.ShowOK, "OK", app.MiniToolbarOK, false)
	drawCommitButton(s, tb.ID, tb.ShowApply, "Apply", app.MiniToolbarApply, tb.ShowOK)
	drawCommitButton(s, tb.ID, tb.ShowCancel, "Cancel", app.MiniToolbarCancel, tb.ShowOK || tb.ShowApply)
}

// drawCommitButton renders one gesture button when shown.
func drawCommitButton(s *app.Session, toolbarID string, show bool, label, gesture string, sameLine bool) {
	if !show {
		return
	}
	if sameLine {
		native.SameLine()
	}
	if native.Button(label + "##commit-" + toolbarID) {
		_ = s.CommitMiniToolbar(toolbarID, gesture)
	}
}

// roundSlider trims slider noise to 4 decimals so change events carry stable values.
func roundSlider(v float64) float64 { return math.Round(v*10000) / 10000 }

// cString returns the NUL-terminated prefix of buf as a Go string.
func cString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}
