// SPDX-License-Identifier: GPL-2.0-only

package app

// Edit-scope visibility (issue #132): while a feature, sketch, or work plane is being edited,
// everything created AFTER it is hidden. The model is uncertain what a mid-history edit will
// invalidate, and trailing geometry can obstruct the view, so Inventor rolls the part back to
// the edited node for the duration of the edit. We engage the feature engine's end-of-part
// marker to drop trailing feature geometry, and the head's overlays consult [Session.EditScopeSeq]
// to drop trailing work planes/axes/sketches. Both use the shared creation stamp (model/seq).

// editScope records the in-progress edit's cutoff: the edited node's creation stamp, and the
// feature engine's end-of-part marker to restore when the edit closes.
type editScope struct {
	active   bool
	seq      uint64 // hide every node whose Seq() is strictly greater than this
	savedEOP int    // feature engine EOP index to restore on endEditScope (eopAll ⇒ -1)
}

// EditScopeSeq returns the active edit's cutoff stamp and true while an edit is open; callers
// (the head's plane/axis/sketch overlays) hide any node whose Seq() exceeds it. ok is false
// when no edit is in progress, so everything is shown normally.
func (s *Session) EditScopeSeq() (uint64, bool) {
	return s.editScope.seq, s.editScope.active
}

// EditScopeHides reports whether a node with the given creation stamp is hidden by the active
// edit (created strictly after the edited node). False when no edit is open.
func (s *Session) EditScopeHides(nodeSeq uint64) bool {
	return s.editScope.active && nodeSeq > s.editScope.seq
}

// beginEditScope opens an edit centered on the node with stamp scopeSeq: it rolls the feature
// engine back to just after that node (so trailing feature geometry vanishes) and records the
// prior end-of-part marker for restoration. Recomputes so the rolled-back model shows at once.
func (s *Session) beginEditScope(scopeSeq uint64) {
	part, err := activePart(s)
	if err != nil {
		return
	}
	feats := part.Features()
	s.editScope = editScope{active: true, seq: scopeSeq, savedEOP: feats.EndOfPartIndex()}
	for i := 0; i < feats.Count(); i++ {
		if feats.Item(i).Seq() > scopeSeq {
			_ = feats.SetEndOfPart(feats.Item(i)) // exclude this feature and everything after it
			part.Recompute()
			return
		}
	}
	feats.RollToEnd() // nothing was created after the edited node
	part.Recompute()
}

// endEditScope closes the active edit, restoring the end-of-part marker captured at open. The
// caller recomputes afterward (Commit/Cancel/FinishSketch already do). A no-op when no edit is
// open, so it is safe to call unconditionally on every edit exit.
func (s *Session) endEditScope() {
	if !s.editScope.active {
		return
	}
	saved := s.editScope.savedEOP
	s.editScope = editScope{}
	part, err := activePart(s)
	if err != nil {
		return
	}
	feats := part.Features()
	if saved < 0 || saved >= feats.Count() {
		feats.RollToEnd()
		return
	}
	_ = feats.SetEndOfPart(feats.Item(saved))
}
