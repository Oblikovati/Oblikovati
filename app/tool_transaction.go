// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/doc"

// Tool-session transactions (#1750 GUI trigger).
//
// The in-transaction Undo/Redo guard — dispatchUndo/dispatchRedo (bindings.go:89,96) and
// the QAT buttons (head/ui/qat.go:69,74) — is driven entirely by [Session.InTransaction].
// Until now the ONLY caller of BeginTransaction/EndTransaction was the add-in wire router
// (addin/router/transactions.go:38,47), so no interactive gesture could ever make the guard
// fire: a smoke test found Undo mid-sketch-line undid normally instead of being blocked.
//
// A multi-click tool is exactly the interactive analogue of an add-in batch — it is midway
// through composing ONE unit of work across many events — so an opted-in tool now holds a
// bounded transaction open for its whole session, from activation to finish or cancel.
//
// The invariant this file exists to hold: a tool transaction NEVER outlives its tool. A
// leaked open group would pin InTransaction() true forever and kill Undo/Redo for the rest
// of the document session — strictly worse than the missing guard it replaces. Every tool
// teardown path therefore routes through endToolTransaction:
//
//	OK          → finishToolCommit → endToolTransaction   (commit)
//	CancelTool  → abandonTool      → endToolTransaction   (Escape / cancel)
//	StartTool   → endToolTransaction                      (tool switch: the outgoing tool)
//	CloseDocument → releaseEditStateFor → abandonTool      (document close)
//
// session_input.go:56,124,166 are the only three sites that create or drop s.tool, so those
// four paths are the complete set (verified by grep for `s.tool = `).

// transactionalTool is an interactive tool whose ENTIRE session is one bounded transaction:
// the guard is live from activation, before the first click, not merely once geometry
// exists. Opt-in per tool (the capability-interface idiom this package already uses for
// chainingTool, autoCommitter and modelReferencePicker) — a tool that does not implement it
// behaves exactly as it did before.
//
// The multi-click line chain is the case (#2024): it accumulates points in the tool and
// creates every segment at once in Commit, so mid-chain the document holds no geometry yet
// and an Undo aimed at "the line I am drawing" would silently rewind some UNRELATED earlier
// step instead — with the placed points still live in the tool.
type transactionalTool interface {
	// RunsInTransaction reports whether this tool's session should be bounded.
	RunsInTransaction() bool
}

// toolTransaction remembers the bounded transaction the active tool holds open, and — this
// is the point of the type — WHICH document it was opened on. BeginTransaction/
// EndTransaction/AbortTransaction all act on whatever document is active when they are
// called (undo.go:439,455,483), so closing by "the active document" would, after a document
// switch mid-tool, leave the real group open on the original document and (worse) decrement
// an add-in's unrelated group on the new one. Closing by id cannot make that mistake.
type toolTransaction struct {
	open bool
	doc  doc.ID
}

// beginToolTransaction opens the bounded transaction spanning an opted-in tool's session.
// It is deliberately quiet: a tool armed with no active document, or on a document with no
// undo stream (a non-recipe document), simply runs unbounded exactly as before — arming a
// tool must never surface an error the user did not cause.
//
// The group is labelled with the tool's Name(), which is the SAME label OK() would have
// passed to RecordActiveEdit (session_input.go:101,115), so the coalesced step the outermost
// EndTransaction records is named exactly as it is today ("Line") — the wrap changes when
// the step is recorded, never what it is called.
func (s *Session) beginToolTransaction(t Tool) {
	tt, ok := t.(transactionalTool)
	if !ok || !tt.RunsInTransaction() {
		return
	}
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	if _, ok := metaStoreFor(d); !ok {
		return // no recipe content ⇒ no undo stream to bound
	}
	if err := s.BeginTransaction(t.Name()); err != nil {
		return
	}
	s.toolTxn = toolTransaction{open: true, doc: d.ID()}
}

// endToolTransaction closes the tool's bounded transaction, on the document it was opened
// on. It is idempotent and safe to call on every teardown path, including paths where no
// transaction was ever opened.
//
// It ENDS rather than aborts, and that choice is load-bearing for the cancel path.
// AbortTransaction (undo.go:483) unconditionally RestoreSnapshot()s the pre-group recipe and
// — unlike undo/redo (undo.go:544,555) — does NOT call reattachActiveSketchAfterRestore, so
// aborting while a sketch is open would rebuild the sketch objects and leave s.activeSketch
// dangling into the discarded ones (#1270 is that exact bug). EndTransaction needs no
// restore: a cancelled tool session leaves the recipe byte-identical to the pre-group
// snapshot, and commitRecipeDelta's no-op guard (undo.go:282-284) then records nothing. So
// cancel closes the group and adds no undo entry — the pre-change behaviour, reached without
// touching the model.
//
// The state is cleared BEFORE the close so an error path cannot leave the flag set and pin
// InTransaction() true; a document that has since been closed took its history with it
// (watchDocumentCloses, undo.go:403-416), so there is nothing left to close.
func (s *Session) endToolTransaction() {
	if !s.toolTxn.open {
		return
	}
	id := s.toolTxn.doc
	s.toolTxn = toolTransaction{}
	d, ok := s.workspace.ByID(id)
	if !ok {
		return
	}
	_ = s.endTransactionOn(d) // ErrNoOpenTransaction here means already closed: nothing to do
}

// ToolHoldsTransaction reports whether the active interactive tool is holding a bounded
// transaction open. It exists for tests and diagnostics — the guard itself reads
// InTransaction(), which is true for an add-in group too.
func (s *Session) ToolHoldsTransaction() bool { return s.toolTxn.open }

// RunsInTransaction bounds the line tool's whole multi-click chain in one transaction, so
// Undo/Redo are inert from the moment the tool is armed until the chain is finished or
// cancelled (#1750). The chain already committed as a single undo step
// (TestChainedLineIsOneUndoStep), and this does not change that: the step is recorded by the
// closing EndTransaction under the same "Line" label instead of by OK's RecordActiveEdit.
func (t *LineTool) RunsInTransaction() bool { return true }

// The line tool is the transactional tool; the assertion keeps the capability wired.
var _ transactionalTool = (*LineTool)(nil)
