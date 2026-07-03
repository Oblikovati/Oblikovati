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

// commitCancelOutcome reports which button of a commit row fired this frame: OKCommitted is true
// only when OK was clicked AND the commit succeeded (a sick/failed commit keeps the panel open),
// Cancelled when Cancel was clicked. A dialog with per-panel teardown (clearing its edit state)
// reacts to it; creation dialogs that carry no such state ignore the return value.
type commitCancelOutcome struct {
	OKCommitted bool
	Cancelled   bool
}

// drawCommitCancelButtons draws a feature panel's OK/Cancel row — the SINGLE owner of the tool-commit
// row so the sick-config gate stays in one place (M40 audit S7, #1642). OK is disabled both when the
// tool lacks its basic inputs (canCommit) AND when the pending configuration would recompute SICK
// (CommitBlockedReason) — the rule that no sick feature may be committed to the design. When blocked
// for the latter reason, a short amber line explains why, since a disabled button shows no tooltip.
func drawCommitCancelButtons(s *app.Session, canCommit bool) commitCancelOutcome {
	blocked := ""
	if canCommit {
		blocked = s.CommitBlockedReason() // only meaningful once the required inputs are present
	}
	var out commitCancelOutcome
	native.BeginDisabled(!canCommit || blocked != "")
	if native.Button("OK") {
		out.OKCommitted = s.OK() == nil
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
		out.Cancelled = true
	}
	if blocked != "" {
		native.PushStyleColor("Text", commitBlockedColor)
		native.TextWrapped("! " + shortCommitReason(blocked)) // "!" over "⚠": the ImGui font lacks ⚠
		native.PopStyleColor(1)
	}
	return out
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
