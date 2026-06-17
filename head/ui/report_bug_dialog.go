//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// reportBugUI holds the Help ▸ Report Bug modal's UI-only state: whether it is open and the
// comment buffer. The actual capture + submit lifecycle is session state (see app.BeginBugReport),
// so this stays a thin view, like the other head-only dialog toggles (showPreferences et al.).
var reportBugUI struct {
	open bool
	buf  [4096]byte
}

// drawReportBugDialog renders the Report Bug window when open: a free-text box plus Send /
// Cancel. Send hands the comment to the session, which snapshots diagnostics and requests
// the two screenshots; the frame loop's ServiceBugReport finishes capture and submits. The
// window closes on Send so the captures grab the app WITHOUT this dialog covering it.
func drawReportBugDialog(s *app.Session) {
	if !reportBugUI.open {
		return
	}
	native.SetNextWindowSizeOnce(520, 360)
	if native.Begin("Report Bug") {
		native.Text("Describe what you were doing and what went wrong.")
		native.Text("Your settings, open files, platform, recent actions, and two")
		native.Text("screenshots (whole window + viewport) are attached automatically.")
		native.Separator()
		native.InputTextMultiline("##bug-comment", reportBugUI.buf[:], 0, 200)
		native.Separator()
		drawReportBugActions(s)
	}
	native.End()
}

// drawReportBugActions draws the Send / Cancel row, or a progress line while a previous
// report is still being captured or submitted (so a second can't pile on).
func drawReportBugActions(s *app.Session) {
	if s.BugReportInProgress() {
		native.Text("Sending the previous report…")
		return
	}
	if native.Button("Send Report") {
		s.BeginBugReport(bufString(reportBugUI.buf[:]))
		clearBuf(reportBugUI.buf[:])
		reportBugUI.open = false
	}
	native.SameLine()
	if native.Button("Cancel") {
		clearBuf(reportBugUI.buf[:])
		reportBugUI.open = false
	}
}
