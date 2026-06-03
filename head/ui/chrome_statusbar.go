//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
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
		if sb.Notice != "" {
			native.SameLine()
			native.Text("— " + sb.Notice)
		}
	}
	native.End()
}

// selectionText renders the selection count as a short status (e.g. "1 selected").
func selectionText(n int) string {
	return strconv.Itoa(n) + " selected"
}
