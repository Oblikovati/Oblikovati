// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// #1425: a discarded MarshalSnapshot error used to leave the undo baseline empty, so a later edit could
// record before=empty and wipe the whole model on undo. These tests pin that the error is surfaced, the
// baseline is never poisoned, and an empty snapshot is never recorded as a revert target.

// errMarshalBoom is the injected marshal failure.
var errMarshalBoom = errors.New("snapshot marshal boom")

// fakeRecipeStore is a recipeStore whose MarshalSnapshot can be made to fail on demand — the named fake
// the marshal-error tests inject instead of a real component definition (no inline stub, per house rules).
type fakeRecipeStore struct {
	marshalErr error
	restored   []byte
}

func (f *fakeRecipeStore) MarshalSnapshot() ([]byte, error) {
	if f.marshalErr != nil {
		return nil, f.marshalErr
	}
	return []byte(`{"recipe":"ok"}`), nil
}

func (f *fakeRecipeStore) RestoreSnapshot(b []byte) error {
	f.restored = b
	return nil
}

// fakeFailingContent is document content (a recipe store) whose snapshot marshal fails — injected via
// doc.SetContent to exercise the baseline-capture and session-log error paths that resolve content from
// the document rather than taking it as a parameter.
type fakeFailingContent struct {
	fakeRecipeStore
}

func (fakeFailingContent) DocumentType() doc.DocumentType { return doc.Part }

// TestResyncContentKeepsBaselineOnMarshalFailure: a failed resync must leave the existing snapshot intact
// (never replace a good baseline with an empty one), and a successful one updates it.
func TestResyncContentKeepsBaselineOnMarshalFailure(t *testing.T) {
	dh := &docHistory{snapshot: []byte("GOOD-BASELINE")}

	if err := dh.resyncContent(&fakeRecipeStore{marshalErr: errMarshalBoom}); err == nil {
		t.Fatal("resyncContent swallowed the marshal error")
	}
	if string(dh.snapshot) != "GOOD-BASELINE" {
		t.Errorf("a failed resync poisoned the baseline: snapshot = %q, want it untouched", dh.snapshot)
	}

	if err := dh.resyncContent(&fakeRecipeStore{}); err != nil {
		t.Fatalf("resyncContent on a healthy store: %v", err)
	}
	if string(dh.snapshot) != `{"recipe":"ok"}` {
		t.Errorf("a successful resync did not update the baseline: snapshot = %q", dh.snapshot)
	}
}

// TestCommitRecipeDeltaSurfacesMarshalFailure: when capturing the after-snapshot fails, no undo step is
// recorded and the failure is surfaced (structured log + Messages panel), never silently dropped.
func TestCommitRecipeDeltaSurfacesMarshalFailure(t *testing.T) {
	s := NewSession()
	d, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	dh := s.documentHistory(d)
	before := dh.hist.Len()

	s.commitRecipeDelta(d, dh, &fakeRecipeStore{marshalErr: errMarshalBoom}, "Edit")

	if dh.hist.Len() != before {
		t.Errorf("an edit whose snapshot failed to marshal was still recorded (Len %d → %d)", before, dh.hist.Len())
	}
	if !s.messageCenter.hasErrors {
		t.Error("a snapshot marshal failure was not surfaced as an error (silent discard, #1425)")
	}
}

// TestCommitRecipeDeltaRefusesEmptyBaseline: even when the after-snapshot marshals fine, an edit recorded
// against an EMPTY baseline (a prior failure left it empty) would revert to an empty model on undo — so it
// must be refused and surfaced, never recorded.
func TestCommitRecipeDeltaRefusesEmptyBaseline(t *testing.T) {
	s := NewSession()
	d, err := s.NewPart()
	if err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	dh := s.documentHistory(d)
	dh.snapshot = nil // simulate a prior resync marshal failure that left no baseline
	before := dh.hist.Len()

	s.commitRecipeDelta(d, dh, &fakeRecipeStore{}, "Edit") // after marshals fine, baseline is empty

	if dh.hist.Len() != before {
		t.Errorf("an edit was recorded against an empty baseline — a Revert would wipe the model (#1425)")
	}
	if !s.messageCenter.hasErrors {
		t.Error("recording against an empty baseline was not surfaced as an error")
	}
}

// TestPoisonedBaselineDoesNotEmptyModelOnSubsequentEdit is the end-to-end guard: a real part with a body,
// whose undo baseline is then poisoned (as a resync marshal failure would), must not record a wipe-risk
// step on the next edit, must keep its geometry, and must surface the failure.
func TestPoisonedBaselineDoesNotEmptyModelOnSubsequentEdit(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)

	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after extrude: %d bodies, want 1", def.SurfaceBodies().Count())
	}

	d := s.ActiveDocument()
	dh := s.histories[d.ID()]
	dh.snapshot = nil // a resync marshal failure poisoned the baseline
	steps := dh.hist.Len()

	s.RecordActiveEdit("PostPoisonEdit") // routes through commitRecipeDelta with the real part

	if dh.hist.Len() != steps {
		t.Errorf("recorded an edit against the poisoned (empty) baseline: Len %d → %d (#1425)", steps, dh.hist.Len())
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Errorf("the model lost its body after a poisoned-baseline edit: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	if !s.messageCenter.hasErrors {
		t.Error("the poisoned-baseline edit was not surfaced as an error")
	}
}

// TestBaselineAndSessionLogSurfaceContentMarshalFailure covers the two sites that resolve content from the
// document (not as a parameter): documentHistory capturing the open-state baseline, and watchTransactions
// capturing the session-log recipe. Both must surface a marshal failure rather than drop it — the baseline
// stays empty (guarded at commit) and the audit event records with a nil recipe, never silently (#1425).
func TestBaselineAndSessionLogSurfaceContentMarshalFailure(t *testing.T) {
	s := NewSession()
	d, err := s.Workspace().Add(doc.Part, "marshal-fail.obk", true)
	if err != nil {
		t.Fatalf("Workspace().Add: %v", err)
	}
	d.SetContent(&fakeFailingContent{fakeRecipeStore{marshalErr: errMarshalBoom}})

	dh := s.documentHistory(d) // captures the open-state baseline — marshal fails here
	if !s.messageCenter.hasErrors {
		t.Error("open-state baseline marshal failure was not surfaced (silent discard, #1425)")
	}
	if len(dh.snapshot) != 0 {
		t.Errorf("a failed baseline capture left snapshot = %q, want empty (guarded at commit)", dh.snapshot)
	}

	s.messageCenter.hasErrors = false // observe the watchTransactions path's own report
	beforeEvents := len(s.txEvents)
	event.Emit(s.bus, event.After, TransactionCommitted{Document: d.ID(), Label: "X"})

	if len(s.txEvents) != beforeEvents+1 {
		t.Errorf("the audit event was dropped on a marshal failure: %d → %d", beforeEvents, len(s.txEvents))
	}
	if r := s.txEvents[len(s.txEvents)-1].recipe; r != nil {
		t.Errorf("a failed-marshal audit recipe should be nil, got %d bytes", len(r))
	}
	if !s.messageCenter.hasErrors {
		t.Error("session-log marshal failure was not surfaced (silent discard, #1425)")
	}
}
