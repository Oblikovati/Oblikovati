// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/command"
	"oblikovati.org/event"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
)

// maxSessionTxEvents bounds the append-only transaction log so a very long session cannot
// grow it without limit; once full the oldest events are dropped (the report shows the most
// recent activity, which is what matters for a repro).
const maxSessionTxEvents = 2000

// sessionTxEvent is one committed transaction recorded for the life of the session — an
// append-only audit of what the user did since the app opened, independent of the
// per-document undo stacks (which shrink on undo). recipe is the document's full parametric
// recipe after the step (the complete command payload), so the sequence replays precisely.
type sessionTxEvent struct {
	when   time.Time
	doc    string
	label  string
	recipe []byte
}

// watchTransactions appends every committed transaction (on any document) to the session's
// append-only event log, capturing the document's resulting recipe as the replayable
// command payload. Undo/redo do not emit TransactionCommitted, so the log is a forward
// audit that survives undo — unlike a document's undo stack. The event fires right after
// commitRecipeDelta advances the snapshot, so the content's recipe is the after-state.
func (s *Session) watchTransactions() {
	event.Subscribe(s.bus, event.After, func(_ event.Context, e TransactionCommitted) event.Outcome {
		name, recipe := "untitled", []byte(nil)
		if d, ok := s.workspace.ByID(e.Document); ok {
			name = d.DisplayName()
			if rc, ok := d.Content().(recipeStore); ok {
				if r, err := rc.MarshalSnapshot(); err != nil {
					s.reportSnapshotFailure("capturing the session-log recipe", d, err) // #1425: not silent
				} else {
					recipe = r
				}
			}
		}
		s.txEvents = append(s.txEvents, sessionTxEvent{when: time.Now().UTC(), doc: name, label: e.Label, recipe: recipe})
		if len(s.txEvents) > maxSessionTxEvents {
			s.txEvents = s.txEvents[len(s.txEvents)-maxSessionTxEvents:]
		}
		return event.Continue()
	})
}

// recipeStore is the document content the app's undo stream records: content whose entire
// recipe can be captured (MarshalSnapshot) and restored (command.RecipeStore.RestoreSnapshot).
// Part and assembly definitions both satisfy it, so an assembly edit — placing a component
// (#763) — is one undo step exactly like a part edit, recorded through the same chokepoint.
//
// Snapshots use a fast internal codec (JSON), NOT the human-readable YAML of the on-disk recipe:
// yaml.v3-encoding a large part's recipe on every edit was ~7 s / 150 MB (#1147). The codec is
// internal to the undo stream + the session transaction log, so it never affects file format.
type recipeStore interface {
	command.RecipeStore
	MarshalSnapshot() ([]byte, error)
}

// Part and assembly definitions and drawing content are the concrete recipe stores a snapshot event
// navigates. Drawing content joined them in #1448 so a wire drawing edit (views/annotations/
// dimensions/sketches/sheets), classified mutating in #1426/#1447 for replication, also records an
// undo step — its undo labels were dead until DrawingContent gained snapshot support.
var (
	_ recipeStore = (*compdef.PartComponentDefinition)(nil)
	_ recipeStore = (*compdef.AssemblyComponentDefinition)(nil)
	_ recipeStore = (*drawing.Content)(nil)
)

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

	// savedDepths are the cursor positions (history depths) at which the document was
	// written to disk — the save checkpoints a history browser flags so the user can tell
	// persisted edits from in-memory ones. Saving appends here; it never clears the stream,
	// so the full history since the document opened stays navigable across saves.
	savedDepths []int
}

// markSaved records the current cursor depth as a save checkpoint, ignoring a repeated save
// at the same depth (saving twice with no edit between).
func (dh *docHistory) markSaved() {
	pos := dh.hist.Len()
	if n := len(dh.savedDepths); n > 0 && dh.savedDepths[n-1] == pos {
		return
	}
	dh.savedDepths = append(dh.savedDepths, pos)
}

// savedDepthsWithin returns the save checkpoints that still fall within a timeline of max
// steps, as a fresh slice. A new edit truncates the redo branch, so a checkpoint recorded on
// a since-discarded branch (depth > max) is dropped rather than shown against an unrelated
// state.
func (dh *docHistory) savedDepthsWithin(max int) []int {
	out := make([]int, 0, len(dh.savedDepths))
	for _, d := range dh.savedDepths {
		if d <= max {
			out = append(out, d)
		}
	}
	return out
}

// resync sets snapshot to the recipe the document now holds — called after undo/redo
// so the next new edit captures the correct before-snapshot for its event. It returns the
// marshal error (content that is not a recipe store is a silent no-op) so the caller can
// surface it; on failure the existing snapshot is left untouched (see resyncContent, #1425).
func (dh *docHistory) resync(d *doc.Document) error {
	c, ok := d.Content().(recipeStore)
	if !ok {
		return nil
	}
	return dh.resyncContent(c)
}

// resyncContent sets snapshot to content's current recipe. On a marshal failure it returns the error
// and leaves the EXISTING snapshot untouched: a transient failure must never replace a good baseline
// with an empty snapshot, which a later edit's before-revert would then restore as an empty model —
// silent data loss on undo (#1425). An empty snapshot must never become a revert baseline.
func (dh *docHistory) resyncContent(content recipeStore) error {
	snap, err := content.MarshalSnapshot()
	if err != nil {
		return err
	}
	dh.snapshot = snap
	return nil
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
	if err := dh.resync(d); err != nil {
		s.reportSnapshotFailure("capturing the open-state baseline", d, err) // #1425: empty baseline guarded at commit
	}
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
func (s *Session) recordEdit(content recipeStore, label string) {
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
	s.commitRecipeDelta(d, dh, content, label)
}

// commitRecipeDelta records the part's recipe delta as one undo event and
// pushes values other documents derive from through the reference graph
// (M02-F06). A no-op delta (before == after) records nothing. A Before
// TransactionCommitted handler may veto, which reverts the part to the
// pre-edit snapshot and reports the commit as an abort (M04-F05).
func (s *Session) commitRecipeDelta(d *doc.Document, dh *docHistory, content recipeStore, label string) {
	after, err := content.MarshalSnapshot()
	if err != nil {
		s.reportSnapshotFailure("recording an edit", d, err) // never silently drop it (#1425)
		return
	}
	if bytes.Equal(after, dh.snapshot) {
		return
	}
	// Refuse to record against an empty baseline: a missing snapshot (a prior marshal failure left it
	// empty) would make this event's Revert restore an empty recipe and wipe the model on undo. An empty
	// snapshot must never become a revert baseline (#1425); surface it instead of poisoning the stream.
	if len(dh.snapshot) == 0 {
		s.reportSnapshotFailure("recording an edit (no undo baseline)", d, errEmptyBaseline)
		return
	}
	if out := event.Emit(s.bus, event.Before, TransactionCommitted{Document: d.ID(), Label: label}); out.Vetoed() {
		s.revertVetoedCommit(d, dh, content, label, out.Reason)
		return
	}
	dh.hist.Record(command.NewRecipeEvent(label, dh.snapshot, after, content))
	dh.snapshot = after
	s.resyncDerivedTables(d, map[string]bool{})
	event.Emit(s.bus, event.After, TransactionCommitted{Document: d.ID(), Label: label})
}

// errEmptyBaseline marks a refused undo record whose before-snapshot was empty — a prior marshal failure
// left no baseline, so recording would risk an empty-recipe revert (#1425).
var errEmptyBaseline = errors.New("app: empty undo baseline (a prior snapshot marshal failed)")

// reportSnapshotFailure surfaces a recipe-snapshot marshal failure — a serious, data-loss-adjacent event
// — to the structured log and the reviewable Messages panel instead of silently discarding the error and
// risking a poisoned undo baseline. The edit is left unrecorded so the model is never put at risk (#1425).
func (s *Session) reportSnapshotFailure(where string, d *doc.Document, err error) {
	s.messageCenter.AddMessage(
		fmt.Sprintf("undo: %s failed for %q: %v — edit not recorded to protect the model", where, d.DisplayName(), err),
		types.SeverityError)
}

// revertVetoedCommit rolls the part back to the stream's snapshot after a Before
// handler vetoed the commit, surfacing the veto reason in the status bar.
func (s *Session) revertVetoedCommit(d *doc.Document, dh *docHistory, content recipeStore, label, reason string) {
	if err := content.RestoreSnapshot(dh.snapshot); err != nil {
		s.notice = err.Error()
		return
	}
	s.rebindReferences(d)
	s.notice = reason
	s.resyncDerivedTables(d, map[string]bool{})
	event.Emit(s.bus, event.After, TransactionAborted{Document: d.ID(), Label: label})
}

// rebindReferences re-binds cross-document references after a recipe restore (#763). An
// assembly's RestoreSnapshot leaves its occurrences pending — binding each to its component
// document needs the workspace to open the component, which only an owner-aware caller has — so
// every app-layer restore (undo, redo, abort, veto-revert) pairs with this to re-bind. It also
// re-binds a derived part's source after a part restore. Content that restores fully in place
// implements no resolver and is a silent no-op.
func (s *Session) rebindReferences(d *doc.Document) {
	if r, ok := d.Content().(doc.ReferenceResolver); ok {
		if err := r.ResolveReferences(d); err != nil {
			s.notice = err.Error()
		}
	}
}

// RecordAddInEdit finalizes a router-applied mutation as one undo step: the
// add-in wire path has no interactive tool to call recordEdit, so its handlers
// call this exported seam after their recompute (M02-F08, Oblikovati#607).
func (s *Session) RecordAddInEdit(part *compdef.PartComponentDefinition, label string) {
	s.recordEdit(part, label)
}

// EnsureActiveEditBaseline opens the active document's undo stream now, if it has not been
// opened already, so its baseline snapshot is the pre-edit state. The router calls it before
// dispatching a mutating method, so a document whose stream was never opened (it did not come
// through NewPart/NewAssembly/open) still records its first wire edit against the state before
// the handler runs — a lazily-created stream would capture the post-edit recipe as its
// baseline and silently drop that first step. Idempotent: an existing stream keeps its
// snapshot, so calling it before every mutating method is cheap.
func (s *Session) EnsureActiveEditBaseline() {
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	if _, ok := d.Content().(recipeStore); !ok {
		return
	}
	s.documentHistory(d)
}

// RecordActiveEdit registers the active document's recipe delta as one undo step labelled
// label, resolving the document's recipe-store content (part or assembly) itself. It is the
// central seam the method router calls after any mutating wire method succeeds, so every
// API / MCP / Lua mutation lands on the same undo stream interactive tools use — no
// per-handler wiring (the gap that left feature/sketch/work-plane/assembly edits made over
// the wire un-undoable). Resolving content from the active document means the router needs
// no per-method knowledge of part vs assembly.
//
// It is safe to call unconditionally: a no-op delta records nothing, so a method whose
// handler already recorded its own step (parameters) and a metadata-only method whose change
// the parametric recipe does not capture both leave the stream untouched. Inside a bounded
// transaction the delta defers to EndTransaction.
func (s *Session) RecordActiveEdit(label string) {
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	content, ok := d.Content().(recipeStore)
	if !ok {
		return
	}
	s.recordEdit(content, label)
}

// ErrNoOpenTransaction is returned by EndTransaction when no bounded transaction is open.
var ErrNoOpenTransaction = errors.New("app: no open transaction to end")

// watchDocumentCloses discards a closed document's transaction stream and
// announces the deletion (M04-F05): the undo steps die with the document, and
// dropping the map entry keeps closed documents from leaking histories.
func (s *Session) watchDocumentCloses() {
	event.Subscribe(s.workspace.Events(), event.After, func(_ event.Context, e doc.DocumentClose) event.Outcome {
		if _, ok := s.histories[e.Document.ID()]; ok {
			delete(s.histories, e.Document.ID())
			event.Emit(s.bus, event.After, TransactionDeleted{Document: e.Document.ID()})
		}
		// The document's add-in client graphics die with it, so its overlays cannot resurface
		// on a later document that happens to reuse no id (M05-F05 doc-scoped graphics).
		delete(s.graphicsByDoc, e.Document.ID())
		// Its hidden-body view state dies with it too (#1105 doc-scoped visibility).
		delete(s.hiddenBodyKeysByDoc, e.Document.ID())
		return event.Continue()
	})
}

// watchDocumentSwitches clears the selection when a DIFFERENT document is activated (#1105). The
// selection set holds Selectable refs into the previously active document; without this they leaked
// into the newly activated document's viewport (stale highlights, wrong-document operations). It
// fires only on an actual switch, so re-activating the same document leaves its selection intact.
func (s *Session) watchDocumentSwitches() {
	var prev doc.ID
	event.Subscribe(s.workspace.Events(), event.After, func(_ event.Context, e doc.DocumentActivate) event.Outcome {
		id := e.Document.ID()
		if id != prev {
			prev = id
			s.selection.Clear()
		}
		return event.Continue()
	})
}

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
	content, ok := d.Content().(recipeStore)
	if !ok {
		return nil
	}
	s.commitRecipeDelta(d, dh, content, label)
	return nil
}

// AbortTransaction discards the open bounded transaction instead of committing
// it: the part reverts to the group's pre-Begin snapshot (whatever the nesting
// depth — an abort cancels the whole group, there is no partial abort) and no
// undo step is recorded. The seam behind wire transaction.abort, so an add-in
// whose batch fails partway does not leave the document half-edited (M04-F05).
func (s *Session) AbortTransaction() error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	dh := s.documentHistory(d)
	if dh.groupDepth == 0 {
		return ErrNoOpenTransaction
	}
	label := dh.groupLabel
	dh.groupDepth, dh.groupLabel = 0, ""
	if content, ok := d.Content().(recipeStore); ok {
		if err := content.RestoreSnapshot(dh.snapshot); err != nil {
			return err
		}
		s.rebindReferences(d)
		s.resyncDerivedTables(d, map[string]bool{})
	}
	event.Emit(s.bus, event.After, TransactionAborted{Document: d.ID(), Label: label})
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
	return s.undoDocument(d)
}

// Redo re-applies the next event ahead of the active document's cursor.
func (s *Session) Redo() error {
	d := s.ActiveDocument()
	if d == nil {
		s.notice = ErrNoActiveDoc.Error()
		return ErrNoActiveDoc
	}
	return s.redoDocument(d)
}

// undoDocument / redoDocument move one document's cursor by a single step, restoring the
// matching recipe and emitting the lifecycle event for that document. They take the document
// explicitly (not the active one) so a history browser can navigate a background document's
// timeline without first activating it — the per-step seam JumpDocumentTo loops over.
func (s *Session) undoDocument(d *doc.Document) error {
	dh := s.documentHistory(d)
	ev := TransactionUndone{Document: d.ID(), Label: lastLabel(dh.hist.UndoLabels())}
	if err := cursorMove(s, d, dh, ev, dh.hist.Undo); err != nil {
		return err
	}
	s.reattachActiveSketchAfterRestore(d) // restore rebuilt the sketch objects; re-bind the edit (#1270)
	event.Emit(s.bus, event.After, ev)
	return nil
}

func (s *Session) redoDocument(d *doc.Document) error {
	dh := s.documentHistory(d)
	ev := TransactionRedone{Document: d.ID(), Label: firstLabel(dh.hist.RedoLabels())}
	if err := cursorMove(s, d, dh, ev, dh.hist.Redo); err != nil {
		return err
	}
	s.reattachActiveSketchAfterRestore(d) // restore rebuilt the sketch objects; re-bind the edit (#1270)
	event.Emit(s.bus, event.After, ev)
	return nil
}

// ErrHistoryTransactionOpen is returned by JumpDocumentTo when a bounded transaction is open
// on the target document — a bare cursor move would corrupt the unit being recorded.
var ErrHistoryTransactionOpen = errors.New("app: cannot jump history while a transaction is open")

// JumpDocumentTo moves the open document id's undo cursor to an absolute position (0 = the
// open/baseline state, len(entries) = the latest state), undoing or redoing as many steps as
// needed in one call — the click-to-jump a history browser needs over a long stream. Each step
// runs through the same per-step seam as Undo/Redo, so references re-bind and derived values
// resync along the way (essential for an assembly whose parts the jump moves). It errors if the
// document is not open, a transaction is open, or position is out of range.
func (s *Session) JumpDocumentTo(id doc.ID, position int) error {
	d, ok := s.workspace.ByID(id)
	if !ok {
		return fmt.Errorf("app: history jump: document %d is not open", id)
	}
	dh := s.documentHistory(d)
	if dh.groupDepth > 0 {
		return ErrHistoryTransactionOpen
	}
	if total := dh.hist.Len() + len(dh.hist.RedoLabels()); position < 0 || position > total {
		return fmt.Errorf("app: history jump: position %d out of range [0,%d]", position, total)
	}
	for dh.hist.Len() > position {
		if err := s.undoDocument(d); err != nil {
			return err
		}
	}
	for dh.hist.Len() < position {
		if err := s.redoDocument(d); err != nil {
			return err
		}
	}
	return nil
}

// DocumentTimeline is one open document's undo stream for a history browser: every committed
// step oldest-first (Labels), the cursor Position (how many steps are applied, so
// Labels[:Position] are past and Labels[Position:] are future), the save checkpoints
// (SavedDepths, cursor positions at which the document was written to disk), and the document
// Name. It is a read-only snapshot; JumpDocumentTo moves the cursor.
type DocumentTimeline struct {
	Name        string
	Position    int
	Labels      []string
	SavedDepths []int
}

// DocumentHistoryView returns the open document id's full timeline since it was opened, or
// false when id is not an open recipe document. It does not activate the document, so a browser
// can read several documents' timelines side by side. UndoLabels is the applied past
// (oldest-first) and RedoLabels the redoable future, so their concatenation is the whole stream.
func (s *Session) DocumentHistoryView(id doc.ID) (DocumentTimeline, bool) {
	d, ok := s.workspace.ByID(id)
	if !ok {
		return DocumentTimeline{}, false
	}
	if _, ok := d.Content().(recipeStore); !ok {
		return DocumentTimeline{}, false
	}
	dh := s.documentHistory(d)
	labels := append(dh.hist.UndoLabels(), dh.hist.RedoLabels()...)
	return DocumentTimeline{
		Name:        d.DisplayName(),
		Position:    dh.hist.Len(),
		Labels:      labels,
		SavedDepths: dh.savedDepthsWithin(len(labels)),
	}, true
}

// cursorMove runs one undo/redo navigation behind its Before event: a handler may
// veto the move (e.g. external state cannot roll back, M04-F05), and a failed
// move surfaces in the status bar. The caller emits the matching After event.
// Generic (and free, since methods cannot be) because the bus dispatches on the
// event's static type — an interface-typed emit would reach no subscriber.
func cursorMove[E event.Event](s *Session, d *doc.Document, dh *docHistory, ev E, move func() error) error {
	if out := event.Emit(s.bus, event.Before, ev); out.Vetoed() {
		s.notice = out.Reason
		return &doc.VetoError{Operation: "transaction", Reason: out.Reason}
	}
	if err := move(); err != nil {
		s.notice = err.Error()
		return err
	}
	s.rebindReferences(d) // an assembly restore leaves occurrences pending; re-bind before resync (#763)
	if err := dh.resync(d); err != nil {
		s.reportSnapshotFailure("re-capturing the baseline after a cursor move", d, err) // #1425
	}
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
