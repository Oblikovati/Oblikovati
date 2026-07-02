//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// commitBlockedColor tints the "why OK is unavailable" line amber — the config is not an error to
// fix in the field, just not yet buildable.
var commitBlockedColor = [4]float32{0.95, 0.55, 0.25, 1}

// drawCommitCancelButtons draws a feature panel's OK/Cancel row. OK is disabled both when the tool
// lacks its basic inputs (canCommit) AND when the pending configuration would recompute SICK
// (CommitBlockedReason) — the rule that no sick feature may be committed to the design. When
// blocked for the latter reason, a short amber line explains why, since a disabled button shows no
// tooltip.
func drawCommitCancelButtons(s *app.Session, canCommit bool) {
	blocked := ""
	if canCommit {
		blocked = s.CommitBlockedReason() // only meaningful once the required inputs are present
	}
	native.BeginDisabled(!canCommit || blocked != "")
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
	if blocked != "" {
		native.PushStyleColor("Text", commitBlockedColor)
		native.TextWrapped("! " + shortCommitReason(blocked)) // "!" over "⚠": the ImGui font lacks ⚠
		native.PopStyleColor(1)
	}
}

// shortCommitReason trims a kernel health reason to one readable line for the panel: it drops a
// bracketed detail list (e.g. an inconsistent-orientation dump enumerating dozens of edges) and
// caps the length, so the warning stays a hint rather than flooding the dialog.
func shortCommitReason(r string) string {
	if i := strings.IndexByte(r, '['); i > 0 {
		r = strings.TrimSpace(r[:i])
	}
	const max = 140
	if len(r) > max {
		r = r[:max] + "…"
	}
	return r
}
