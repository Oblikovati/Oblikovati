// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"oblikovati.org/build"
	"oblikovati.org/report"
)

// fakeBugSubmitter records the payload it is handed instead of POSTing it, so the
// capture→submit flow can be driven without a network.
type fakeBugSubmitter struct {
	mu   sync.Mutex
	got  report.Payload
	err  error
	hits int
}

func (f *fakeBugSubmitter) Submit(_ context.Context, p report.Payload) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = p
	f.hits++
	return f.err
}

func (f *fakeBugSubmitter) payload() (report.Payload, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.got, f.hits
}

func TestCollectDiagnosticsPopulatesPlatformAndSettings(t *testing.T) {
	s := NewSession()
	d := s.CollectDiagnostics()
	if d.OS != runtime.GOOS || d.Arch != runtime.GOARCH {
		t.Errorf("platform = %s/%s, want %s/%s", d.OS, d.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if d.AppVersion != build.Version {
		t.Errorf("AppVersion = %q, want %q", d.AppVersion, build.Version)
	}
	if d.UserSettings == "" {
		t.Error("UserSettings should render the options YAML, got empty")
	}
}

func TestBugReportCaptureThenSubmit(t *testing.T) {
	s := NewSession()
	fake := &fakeBugSubmitter{}
	s.SetBugSubmitter(fake)

	s.BeginBugReport("crash when I extrude")
	if !s.BugReportInProgress() {
		t.Fatal("report should be in progress after Begin")
	}
	// The head would write both PNGs to the requested paths; simulate that.
	mustWrite(t, s.bugReport.winPath, "WINDOWPNG")
	mustWrite(t, s.bugReport.viewPath, "VIEWPORTPNG")

	s.ServiceBugReport() // captures present → launches the submit goroutine
	waitUntil(t, func() bool { return s.bugOutcome.Load() != nil })
	s.ServiceBugReport() // drains the outcome → idle

	if s.BugReportInProgress() {
		t.Fatal("report should be idle after submit drains")
	}
	got, hits := fake.payload()
	if hits != 1 {
		t.Fatalf("submit hits = %d, want 1", hits)
	}
	if got.Comment != "crash when I extrude" {
		t.Errorf("comment = %q", got.Comment)
	}
	if string(got.WindowPNG) != "WINDOWPNG" || string(got.ViewportPNG) != "VIEWPORTPNG" {
		t.Errorf("screenshots not folded into payload: win=%q view=%q", got.WindowPNG, got.ViewportPNG)
	}
	// Temp captures are cleaned up after they are read.
	if _, err := os.Stat(s.bugReport.winPath); err == nil {
		t.Error("temp window capture not removed")
	}
}

func TestBugReportWaitsForBothCaptures(t *testing.T) {
	s := NewSession()
	fake := &fakeBugSubmitter{}
	s.SetBugSubmitter(fake)
	s.BeginBugReport("note")
	mustWrite(t, s.bugReport.winPath, "WINDOWPNG") // only one capture ready

	s.ServiceBugReport() // must NOT submit yet
	if _, hits := fake.payload(); hits != 0 {
		t.Fatalf("submitted before both captures present (hits=%d)", hits)
	}
	if !s.BugReportInProgress() {
		t.Error("should still be capturing")
	}
}

func TestBeginBugReportIgnoredWhileInProgress(t *testing.T) {
	s := NewSession()
	s.SetBugSubmitter(&fakeBugSubmitter{})
	s.BeginBugReport("first")
	first := s.bugReport.winPath
	s.BeginBugReport("second") // no-op
	if s.bugReport.winPath != first || s.bugReport.payload.Comment != "first" {
		t.Error("second Begin should be ignored while one is in flight")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
