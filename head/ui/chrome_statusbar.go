//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawStatusBar renders Inventor's status bar: the active command's step prompt and the
// selection count, with OK/Cancel for the running tool. OK/Cancel act on the session
// directly — a tool's lifecycle is a UI concern — mirroring the Edit-menu Cancel above.
func drawStatusBar(s *app.Session) {
	if native.Begin("Status") {
		sb := app.BuildStatus(s)
		native.Text(sb.Prompt)
		if sb.ToolActive {
			native.SameLine()
			native.BeginDisabled(!sb.CanCommit)
			if native.Button("OK") {
				_ = s.OK() // a failed commit keeps the tool open (Inventor behavior)
			}
			native.EndDisabled()
			native.SameLine()
			if native.Button("Cancel") {
				s.CancelTool()
			}
		}
		native.SameLine()
		native.Text(selectionText(sb.SelectionCount))
		if sb.StatusText != "" {
			native.SameLine()
			native.Text("· " + sb.StatusText)
		}
		drawStatusProgress(s, sb)
		// M26 F03: the notice and the message-center badge are retired here — notices and
		// messages now funnel into the docked Command Window.
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
