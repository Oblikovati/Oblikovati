// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"bytes"
	"errors"

	"oblikovati.org/command"
	"oblikovati.org/event"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// The part definition is the concrete RecipeStore a snapshot event navigates.
var _ command.RecipeStore = (*compdef.PartComponentDefinition)(nil)

// docHistory is one document's transaction-event stream: the [command.History] (the
// cursor over the events) plus snapshot — the recipe the model holds at the current
// cursor position. Every interaction appends a [command.RecipeEvent] whose before is
// this snapshot and whose after is the recipe the interaction produced; undo/redo
// move the cursor and restore the matching snapshot, then resync this field. The
// stream begins when the document is opened (snapshot captures the open state), per
// the event-sourcing model: current state is the fold of all events since open.
type docHistory struct {
	hist     *command.History
	snapshot []byte

	// groupDepth > 0 means a bounded transaction is open: per-edit recording is
	// suppressed and the whole delta is recorded as one event when the outermost
	// EndTransaction runs. groupLabel names that single coalesced step (the outermost
	// Begin's label wins). While open, snapshot stays at the pre-group state, so it is
	// exactly the "before" for the group. See BeginTransaction/EndTransaction.
	groupDepth int
	groupLabel string
}

// resync sets snapshot to the recipe the document now holds — called after undo/redo
// so the next new edit captures the correct before-snapshot for its event.
func (dh *docHistory) resync(d *doc.Document) {
	if part, ok := d.Content().(*compdef.PartComponentDefinition); ok {
		dh.snapshot, _ = part.MarshalRecipe()
	}
}

// documentHistory returns d's event stream, creating it (and capturing the open-state
// snapshot as the stream's baseline) on first use. It is called eagerly when a
// document is created or opened so the first edit's before-snapshot is the open state,
// not the post-edit state. OnChange marks the document dirty and notifies observers
// once per committed step, undo, or redo (the coalesced recompute/notify seam).
func (s *Session) documentHistory(d *doc.Document) *docHistory {
	if dh, ok := s.histories[d.ID()]; ok {
		return dh
	}
	dh := &docHistory{hist: command.NewHistory()}
	dh.resync(d)
	dh.hist.OnChange(func() {
		d.MarkDirty()
		event.Emit(s.bus, event.After, TransactionChanged{Document: d.ID()})
	})
	s.histories[d.ID()] = dh
	return dh
}

// recordEdit appends one transaction event for the interaction that just mutated the
// active part's recipe. It is the single finalize chokepoint every edit routes
// through, called right after the edit's existing part.Recompute(): it captures the
// new recipe (the after-snapshot) and records it against the stream's current
// snapshot, then advances the cursor. The recipe is the parametric input, independent
// of the geometry recompute the caller already ran, so recordEdit does not recompute.
// An edit that changed nothing (e.g. exiting a sketch untouched) leaves before==after
// and records no event, so the stream holds only real changes.
func (s *Session) recordEdit(part *compdef.PartComponentDefinition, label string) {
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	dh := s.documentHistory(d)
	if dh.groupDepth > 0 {
		// Inside a bounded transaction: defer recording so the whole batch becomes one
		// undo step at EndTransaction. snapshot is intentionally left untouched.
		return
	}
	after, err := part.MarshalRecipe()
	if err != nil || bytes.Equal(after, dh.snapshot) {
		return
	}
	dh.hist.Record(command.NewRecipeEvent(label, dh.snapshot, after, part))
	dh.snapshot = after
}

// ErrNoOpenTransaction is returned by EndTransaction when no bounded transaction is open.
var ErrNoOpenTransaction = errors.New("app: no open transaction to end")

// BeginTransaction opens a bounded transaction on the active document: every edit
// recorded until the matching EndTransaction is coalesced into a single undo step named
// label. Begin/End nest; the outermost Begin's label names the step and only the
// outermost End commits it. A collaboration add-in wraps a drained batch of remote
// operations in one Begin/End so the batch is one team-shared undo step (ADR-0005).
func (s *Session) BeginTransaction(label string) error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	dh := s.documentHistory(d)
	if dh.groupDepth == 0 {
		dh.groupLabel = label
	}
	dh.groupDepth++
	return nil
}

// EndTransaction closes the innermost open bounded transaction. At depth 0 (the
// outermost End) it records the whole accumulated delta as one event and advances the
// cursor; a no-op group (before == after) records nothing.
func (s *Session) EndTransaction() error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	dh := s.documentHistory(d)
	if dh.groupDepth == 0 {
		return ErrNoOpenTransaction
	}
	dh.groupDepth--
	if dh.groupDepth > 0 {
		return nil // inner end: keep accumulating until the outermost End
	}
	label := dh.groupLabel
	dh.groupLabel = ""
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil
	}
	after, err := part.MarshalRecipe()
	if err != nil || bytes.Equal(after, dh.snapshot) {
		return nil
	}
	dh.hist.Record(command.NewRecipeEvent(label, dh.snapshot, after, part))
	dh.snapshot = after
	return nil
}

// InTransaction reports whether a bounded transaction is open on the active document.
func (s *Session) InTransaction() bool {
	st := s.activeStream()
	return st != nil && st.groupDepth > 0
}

// Undo moves the active document's cursor back one transaction event, restoring the
// prior recipe and recomputing. Redo moves it forward. Both are navigators over the
// event stream — non-destructive: undo leaves the event available to redo until a new
// edit truncates the forward branch.
func (s *Session) Undo() error {
	d := s.ActiveDocument()
	if d == nil {
		s.notice = ErrNoActiveDoc.Error()
		return ErrNoActiveDoc
	}
	dh := s.documentHistory(d)
	if err := dh.hist.Undo(); err != nil {
		s.notice = err.Error()
		return err
	}
	dh.resync(d)
	s.notice = ""
	return nil
}

// Redo re-applies the next event ahead of the active document's cursor.
func (s *Session) Redo() error {
	d := s.ActiveDocument()
	if d == nil {
		s.notice = ErrNoActiveDoc.Error()
		return ErrNoActiveDoc
	}
	dh := s.documentHistory(d)
	if err := dh.hist.Redo(); err != nil {
		s.notice = err.Error()
		return err
	}
	dh.resync(d)
	s.notice = ""
	return nil
}

// CanUndo / CanRedo report whether the active document has an event behind / ahead of
// its cursor. They drive the ribbon Undo/Redo enable state. No active document ⇒ false.
func (s *Session) CanUndo() bool { return s.activeStream() != nil && s.activeStream().hist.CanUndo() }

func (s *Session) CanRedo() bool { return s.activeStream() != nil && s.activeStream().hist.CanRedo() }

// UndoLabel / RedoLabel return the name of the step undo/redo would act on next, for
// the ribbon tooltip ("Undo Extrude"); "" when there is nothing to act on.
func (s *Session) UndoLabel() string { return lastLabel(s.undoLabels()) }
func (s *Session) RedoLabel() string { return firstLabel(s.redoLabels()) }

func (s *Session) undoLabels() []string {
	if st := s.activeStream(); st != nil {
		return st.hist.UndoLabels()
	}
	return nil
}

func (s *Session) redoLabels() []string {
	if st := s.activeStream(); st != nil {
		return st.hist.RedoLabels()
	}
	return nil
}

// activeStream returns the active document's stream without creating one — a read-only
// query for the CanUndo/label accessors. nil when there is no active part document or
// no edit has happened yet.
func (s *Session) activeStream() *docHistory {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	return s.histories[d.ID()]
}

// lastLabel / firstLabel pick the next-acted step from an ordered label slice:
// UndoLabels is oldest-first (undo acts on the last), RedoLabels is redo-order (redo
// acts on the first).
func lastLabel(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	return labels[len(labels)-1]
}

func firstLabel(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}
