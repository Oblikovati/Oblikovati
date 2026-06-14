//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/update"
)

// drawUpdateWindow renders the software-update notification when a check result is
// pending ([app.Session.PendingUpdate]). It shows whether a newer release exists, offers
// a direct link to the GitHub release page, and lets the user turn off the startup check
// — the toggle persists to the user's options (Updates.CheckOnStartup).
func drawUpdateWindow(s *app.Session) {
	res := s.PendingUpdate()
	if res == nil {
		return
	}
	native.SetNextWindowSizeOnce(420, 200)
	if native.Begin("Software Update") {
		drawUpdateBody(s, res)
	}
	native.End()
}

// drawUpdateBody renders the window contents for a pending result.
func drawUpdateBody(s *app.Session, res *update.Result) {
	native.Text(updateHeadline(res))
	native.Separator()
	if res.UpdateAvailable {
		drawReleaseLink(s, res)
	}
	drawStartupToggle(s)
	native.Separator()
	if native.Button("Close") {
		s.DismissUpdate()
	}
}

// drawReleaseLink renders the button that opens the release's GitHub page via the
// platform URL opener.
func drawReleaseLink(s *app.Session, res *update.Result) {
	native.Text("Current: " + res.Current)
	if native.Button("Open Release Page") {
		_ = s.OpenLatestReleasePage()
	}
	native.Separator()
}

// drawStartupToggle renders the "check on startup" checkbox, persisting changes.
func drawStartupToggle(s *app.Session) {
	check := s.UpdateChecksEnabled()
	if native.Checkbox("Check for updates on startup", &check) {
		_ = s.SetUpdateChecksEnabled(check)
	}
}
