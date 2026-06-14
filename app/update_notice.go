// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/app/options"
	"oblikovati.org/update"
)

// Software-update surface (Help ▸ Check for Updates). The session owns the user-visible
// state — the "check on startup" preference and the result to display — but never does
// the network I/O itself: the head runs the check ([oblikovati.org/update]) off the UI
// thread and hands the outcome back through ShowUpdateResult, so the whole flow stays
// unit-testable without a live GitHub (ADR-0014).

// UpdateChecksEnabled reports whether the startup auto-check is on (the persisted
// Updates.CheckOnStartup preference). A manual Help ▸ Check for Updates ignores it.
func (s *Session) UpdateChecksEnabled() bool { return s.appOptions.Updates.CheckOnStartup }

// SetUpdateChecksEnabled stores the startup-check preference (the update notification's
// "don't check on startup" toggle), persisting it to the user's options file.
func (s *Session) SetUpdateChecksEnabled(on bool) error {
	s.appOptions.Updates = options.Updates{CheckOnStartup: on}
	return s.saveOptions()
}

// RequestUpdateCheck flags a manual check; the head consumes it with
// TakeUpdateCheckRequest, runs the network check, and reports back via ShowUpdateResult.
func (s *Session) RequestUpdateCheck() { s.updateCheckRequested = true }

// TakeUpdateCheckRequest returns and clears the pending manual-check request (one-shot,
// so the head launches exactly one check per click).
func (s *Session) TakeUpdateCheckRequest() bool {
	req := s.updateCheckRequested
	s.updateCheckRequested = false
	return req
}

// ShowUpdateResult records a check's outcome for the update window to display, and when a
// newer release exists also drops a status-bar notice so the user sees it even if the
// window is dismissed. Called by the head after the check completes.
func (s *Session) ShowUpdateResult(res update.Result) {
	s.pendingUpdate = &res
	if res.UpdateAvailable {
		s.notice = fmt.Sprintf("Update available: %s — see Help ▸ Check for Updates", res.Latest.Version)
	}
}

// PendingUpdate returns the update outcome to display, or nil when the window is closed.
func (s *Session) PendingUpdate() *update.Result { return s.pendingUpdate }

// DismissUpdate closes the update window.
func (s *Session) DismissUpdate() { s.pendingUpdate = nil }

// OpenLatestReleasePage opens the pending release's GitHub page through the platform URL
// opener (the notification's download link). It errors if there is no release to open.
func (s *Session) OpenLatestReleasePage() error {
	if s.pendingUpdate == nil || s.pendingUpdate.Latest.HTMLURL == "" {
		return fmt.Errorf("app: no release page to open (pending update: %v)", s.pendingUpdate)
	}
	return s.OpenURL(s.pendingUpdate.Latest.HTMLURL)
}
