//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawStatusBar renders the status bar's interactive controls and live indicators: OK/Cancel
// for the running tool, the selection count, and the progress bar. All TEXT MESSAGES — the
// step prompt, add-in status text, and notices — are consolidated into the Command Window
// (M26), so the status bar carries no message text, only state the user acts on or watches.
func drawStatusBar(s *app.Session) {
	if native.Begin("Status") {
		sb := app.BuildStatus(s)
		if sb.ToolActive {
			native.BeginDisabled(!sb.CanCommit)
			if native.Button("OK") {
				_ = s.OK() // a failed commit keeps the tool open (Inventor behavior)
			}
			native.EndDisabled()
			native.SameLine()
			if native.Button("Cancel") {
				s.CancelTool()
			}
			native.SameLine()
		}
		native.Text(selectionText(sb.SelectionCount))
		drawStatusProgress(s, sb)
	}
	native.End()
}

// drawStatusProgress renders the innermost live progress bar with its cancel
// control (M05-F09). Cancel marks the bar; the owner observes it in its next
// update reply and as a progress.cancelled event.
func drawStatusProgress(s *app.Session, sb app.StatusBar) {
	if !sb.HasProgress {
		return
	}
	native.SameLine()
	fraction := float32(0)
	if sb.Progress.Steps > 0 {
		fraction = float32(sb.Progress.Step) / float32(sb.Progress.Steps)
	}
	native.ProgressBar(fraction, 160, sb.Progress.Message)
	if !sb.Progress.Cancelled {
		native.SameLine()
		if native.Button("Cancel##progress") {
			_ = s.CancelProgress(sb.Progress.ID)
		}
	}
}

// selectionText renders the selection count as a short status (e.g. "1 selected").
func selectionText(n int) string {
	return strconv.Itoa(n) + " selected"
}
