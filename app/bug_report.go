// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"oblikovati.org/report"
)

// Help ▸ Report Bug surface. The session owns the report's lifecycle but never blocks the
// frame loop or touches itself off-thread: BeginBugReport snapshots the diagnostics and
// asks the head to capture the two screenshots; ServiceBugReport (called each frame) waits
// for both PNGs, then submits on a goroutine and drains the result through an atomic
// pointer — the same off-thread handoff the update check uses (see update_notice.go).

// bugSubmitTimeout bounds the upload so a slow network never strands the submit goroutine.
// Generous because the payload carries two base64 PNGs.
const bugSubmitTimeout = 30 * time.Second

// bugSubmitter is the seam the session POSTs reports through; report.Submitter is the real
// implementation and tests inject a fake (CLAUDE.md: external I/O behind a thin interface).
type bugSubmitter interface {
	Submit(ctx context.Context, p report.Payload) error
}

// bugPhase tracks where a report is in the capture→submit pipeline.
type bugPhase int

const (
	bugIdle       bugPhase = iota // no report in flight
	bugCapturing                  // waiting for the head to write both screenshot PNGs
	bugSubmitting                 // upload goroutine running
)

// bugReportState is the in-flight report: its phase, the diagnostics gathered at Begin
// (screenshots filled in once captured), and the temp paths the head writes the PNGs to.
type bugReportState struct {
	phase    bugPhase
	payload  report.Payload
	winPath  string
	viewPath string
}

// bugResult is the submit goroutine's outcome handed back to the frame loop.
type bugResult struct{ err error }

// BeginBugReport starts a report: it snapshots the diagnostics, attaches the user's
// comment, and requests a whole-window and a viewport screenshot to temp files. It is a
// no-op while a report is already in flight, so a double-click can't start two.
func (s *Session) BeginBugReport(comment string) {
	if s.bugReport.phase != bugIdle {
		return
	}
	p := s.CollectDiagnostics()
	p.Comment = comment
	id := fmt.Sprintf("oblikovati-bugreport-%d", time.Now().UnixNano())
	s.bugReport = bugReportState{
		phase:    bugCapturing,
		payload:  p,
		winPath:  filepath.Join(os.TempDir(), id+"-window.png"),
		viewPath: filepath.Join(os.TempDir(), id+"-viewport.png"),
	}
	s.RequestWindowCapture(s.bugReport.winPath)
	s.RequestViewportCapture(s.bugReport.viewPath)
	s.feedNotice("Capturing bug report…")
}

// BugReportInProgress reports whether a report is being captured or submitted (the dialog
// uses it to show progress / disable Send).
func (s *Session) BugReportInProgress() bool { return s.bugReport.phase != bugIdle }

// ServiceBugReport advances the report state machine. It must run once per frame AFTER the
// head's window/viewport captures for the frame (so the PNGs exist to read). It is cheap
// and a no-op when idle.
func (s *Session) ServiceBugReport() {
	switch s.bugReport.phase {
	case bugCapturing:
		s.serviceBugCapture()
	case bugSubmitting:
		s.serviceBugSubmit()
	}
}

// serviceBugCapture waits until both screenshot PNGs have been written (atomically, by the
// head), folds them into the payload, and launches the upload goroutine.
func (s *Session) serviceBugCapture() {
	win, okWin := readCapture(s.bugReport.winPath)
	view, okView := readCapture(s.bugReport.viewPath)
	if !okWin || !okView {
		return // the head has not finished writing both captures yet
	}
	s.bugReport.payload.WindowPNG = win
	s.bugReport.payload.ViewportPNG = view
	_ = os.Remove(s.bugReport.winPath)
	_ = os.Remove(s.bugReport.viewPath)
	s.bugReport.phase = bugSubmitting

	payload := s.bugReport.payload
	submitter := s.submitter()
	out := &s.bugOutcome
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), bugSubmitTimeout)
		defer cancel()
		out.Store(&bugResult{err: submitter.Submit(ctx, payload)})
	}()
}

// serviceBugSubmit drains a finished upload and reports the outcome to the user, returning
// the machine to idle.
func (s *Session) serviceBugSubmit() {
	r := s.bugOutcome.Swap(nil)
	if r == nil {
		return
	}
	s.bugReport = bugReportState{} // back to idle, releasing the captured payload
	switch {
	case r.err == nil:
		s.feedNotice("Bug report sent — thank you! We'll follow up on GitHub.")
	case errors.Is(r.err, report.ErrOffline):
		s.feedNotice("Bug report not sent: no connection to the reporting service.")
	default:
		s.feedNotice(fmt.Sprintf("Bug report failed: %v", r.err))
	}
}

// submitter returns the injected reporting submitter, lazily creating the real HTTP one so
// headless/test sessions that never file a report pay nothing and can inject a fake first.
// OBLIKOVATI_REPORTING_ENDPOINT overrides the target so a dev/staging build can point at a
// local reporting service (empty ⇒ report.DefaultEndpoint).
func (s *Session) submitter() bugSubmitter {
	if s.bugSubmitter == nil {
		endpoint := os.Getenv("OBLIKOVATI_REPORTING_ENDPOINT")
		s.bugSubmitter = report.NewSubmitter(endpoint, &http.Client{Timeout: bugSubmitTimeout})
	}
	return s.bugSubmitter
}

// SetBugSubmitter injects the reporting submitter (the head can point it at a configured
// endpoint; tests pass a fake). Pass nil to fall back to the default.
func (s *Session) SetBugSubmitter(sub bugSubmitter) { s.bugSubmitter = sub }

// readCapture reads a screenshot PNG, returning ok=false until it exists and is non-empty.
// The head writes captures via temp-file rename, so a present file is always complete.
func readCapture(path string) ([]byte, bool) {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}
