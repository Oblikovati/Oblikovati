//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/app/cmdline"
	"oblikovati.org/head/internal/native"
)

// The status bar (Inventor-parity 2026-08-17). Inventor keeps a prompt line at the foot of
// the window that "indicates the next action the active command requires", toggled from
// View ▸ Windows ▸ User Interface ▸ Status Bar (C-inventor-ui-reference §7). Oblikovati's
// own bar was retired in M26 when every message moved into the docked Command Window
// (head/ui/command_window.go:13-16) — but the Command Window is a scrolling log, not a
// glanceable prompt line, and it can be closed. This band restores the glanceable line
// WITHOUT taking anything back from the Command Window: it is a read-only mirror of the
// same app.StatusBar model (app/statusbar.go:35 BuildStatus), so the two never disagree.
//
// The band takes app.StatusBar rather than the session — it needs no session verbs, which
// also keeps it off the head→*app.Session coupling ratchet (archguard/head_session_ratchet_test.go).

// statusBarPadX is the status line's inset from the band's right edge.
const statusBarPadX = 8

// statusDimColor is the muted grey of the right-hand summary, so the prompt (default text
// colour) stays the line's focus.
var statusDimColor = [4]float32{0.62, 0.62, 0.62, 1}

// statusBarHeight is the band's fixed height: one text line plus the window padding.
// Computed from the live style so a font-scale change can never clip the line.
func statusBarHeight() float32 {
	m := native.Metrics()
	return native.TextLineHeight() + 2*m.WindowPadY
}

// drawStatusBar renders the bottom band: the active command's prompt on the left, any
// failed-commit notice beside it in the warning colour, and the environment/selection
// summary pinned right. Call it BEFORE the dockspace is created — the band claims its
// slice of the viewport work area, so the panels lay out above it.
func drawStatusBar(sb app.StatusBar) {
	if native.BeginStatusBar("##statusbar", statusBarHeight()) {
		native.Text(statusPromptText(sb))
		if sb.Notice != "" {
			native.SameLine()
			native.PushStyleColor("Text", severityColor(cmdline.Warning))
			native.Text(sb.Notice)
			native.PopStyleColor(1)
		}
		drawStatusBarRight(statusRightText(sb))
	}
	native.End()
}

// drawStatusBarRight pins the summary to the band's right edge on the same line. It stands
// down when the prompt already reaches that far, so a long prompt is never overwritten.
func drawStatusBarRight(text string) {
	if text == "" {
		return
	}
	w, _ := native.MainViewportSize()
	native.SameLine()
	x, y := native.GetCursorPos()
	right := w - native.CalcTextWidth(text) - statusBarPadX
	if right <= x {
		return
	}
	native.SetCursorPos(right, y)
	native.PushStyleColor("Text", statusDimColor)
	native.Text(text)
	native.PopStyleColor(1)
}
