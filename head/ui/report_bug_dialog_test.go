//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"context"
	"testing"

	"oblikovati.org/report"
)

// stubBugSubmitter satisfies the session's bug submitter without any network — the render
// test never completes a capture, so Submit is never actually called, but injecting it
// keeps the default HTTP client out of the test.
type stubBugSubmitter struct{}

func (stubBugSubmitter) Submit(context.Context, report.Payload) error { return nil }

// TestInWindowReportBugDialogRenders opens the real window with the Report Bug dialog
// visible and renders frames in both the idle (Send/Cancel) and in-progress (progress
// line) branches — so a mismatched ImGui Begin/End in either branch trips Dear ImGui's
// assertions here rather than in front of a user.
func TestInWindowReportBugDialogRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	s.SetBugSubmitter(stubBugSubmitter{})

	reportBugUI.open = true
	t.Cleanup(func() { reportBugUI.open = false })

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	// Idle branch: the Send Report / Cancel row.
	frame()
	frame()
	if !reportBugUI.open {
		t.Fatal("dialog should stay open while idle")
	}

	// In-progress branch: requesting a report flips the dialog to its progress line.
	s.BeginBugReport("render-test note")
	if !s.BugReportInProgress() {
		t.Fatal("BeginBugReport should mark the report in progress")
	}
	frame()
	frame()
}
