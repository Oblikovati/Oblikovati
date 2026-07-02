// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The commit gate enforces one rule for every feature tool: a configuration that would recompute
// SICK must not be committable. The tool already builds a DRAFT of the exact feature OK would
// create (DraftPreviewable), and PartFeatures.PreviewResult evaluates it non-destructively — the
// same recompute a commit runs — so its error is precisely "this would be sick". The gate reads
// that: s.OK() refuses (nothing enters the design) and the head disables the OK button while it
// holds, so a sick node can never persist in the tree.

// commitBlockedReason reports why the active tool's pending feature cannot be committed: a
// non-empty reason when evaluating its draft against the current model yields a sick recompute.
// It is empty when there is no tool, the tool builds no draft (non-DraftPreviewable, e.g. sketch
// tools — unconstrained here), the draft is not ready (CanCommit already gates that), the context
// is not a part, or the draft previews healthy. A DEFERRED result is a warning, not sick, so it
// does not block.
func (s *Session) commitBlockedReason() string {
	if s.tool == nil {
		return ""
	}
	dp, ok := s.tool.tool.(DraftPreviewable)
	if !ok {
		return ""
	}
	// Only NEW-feature creation is gated. An in-place EDIT re-applies the feature at its own
	// position in the history, but PreviewResult appends the draft at end-of-part, where an
	// edited feature's references resolve against a different (later) topology — so its result
	// says nothing about the edit's real health. Editing keeps its existing commit-time check.
	if ed, editing := s.tool.tool.(interface{ IsEditing() bool }); editing && ed.IsEditing() {
		return ""
	}
	draft, ready := dp.DraftFeature(s)
	if !ready {
		return ""
	}
	part, err := activePart(s)
	if err != nil {
		return ""
	}
	if _, err := part.Features().PreviewResult(draft); err != nil && !errors.Is(err, feature.ErrDeferred) {
		return err.Error()
	}
	return ""
}

// CommitBlockedReason is [commitBlockedReason] for the head: "" when the active tool's pending
// feature can be committed, otherwise the reason it is sick — shown beside the disabled OK button
// so the user knows why the commit is unavailable.
func (s *Session) CommitBlockedReason() string { return s.commitBlockedReason() }
