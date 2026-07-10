// SPDX-License-Identifier: GPL-2.0-only

package app

// Construction (hidden, consumer-tied) work datums auto-delete when their last consumer is removed
// (#1849). This is the app-session seam for that lifecycle: a delete handler snapshots the
// construction datums that have a consumer (read-only) before the delete, then calls
// PruneOrphanedConstructionDatums after — so the recompute/undo-record stays behind a Session verb,
// not orchestrated in the wire handler (audit B1, #1612).

// ConstructionConsumerSnapshot returns the refs of the active part's construction datums that
// currently have a consumer — the candidates a following delete could orphan. It is read-only; a
// non-part active document yields nil.
func (s *Session) ConstructionConsumerSnapshot() []string {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	return part.ConstructionConsumerSnapshot()
}

// PruneOrphanedConstructionDatums auto-deletes each snapshot datum whose last consumer the
// just-applied delete removed, then recomputes and records the edit so the auto-delete is a single
// undoable step (#1849). A no-op when nothing was orphaned.
func (s *Session) PruneOrphanedConstructionDatums(snapshot []string) {
	if len(snapshot) == 0 {
		return
	}
	part, err := activePart(s)
	if err != nil {
		return
	}
	if part.PruneConstructionOrphans(snapshot) > 0 {
		part.Recompute()
		s.recordEdit(part, labelDeleteConstructionDatum)
	}
}
