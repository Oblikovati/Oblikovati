// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// typedPayloadRecorder keeps the raw JSON per event type so each payload can be decoded
// with its own wire DTO.
type typedPayloadRecorder struct {
	mu  sync.Mutex
	raw map[string][][]byte
}

func newTypedPayloadRecorder() *typedPayloadRecorder {
	return &typedPayloadRecorder{raw: map[string][][]byte{}}
}

func (r *typedPayloadRecorder) sink(b []byte) {
	var tag struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &tag); err != nil {
		return
	}
	r.mu.Lock()
	r.raw[tag.Type] = append(r.raw[tag.Type], append([]byte(nil), b...))
	r.mu.Unlock()
}

// decodeSingle asserts eventType reached the sink exactly once and decodes it.
// Generic over the wire payloads so each call site stays statically typed.
func decodeSingle[P wire.TransactionEventPayload | wire.FileResolutionEventPayload |
	wire.FileDirtyEventPayload | wire.FileDialogHookPayload](t *testing.T, r *typedPayloadRecorder, eventType string) P {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var payload P
	got := r.raw[eventType]
	if len(got) != 1 {
		t.Fatalf("%s fired %d times, want 1", eventType, len(got))
	}
	if err := json.Unmarshal(got[0], &payload); err != nil {
		t.Fatalf("decode %s: %v", eventType, err)
	}
	return payload
}

// TestForwardsTransactionLifecycle drives a commit, an undo, and a redo through
// the session and checks each reaches the sink as the wire payload with the
// cursor-relative point (M04-F05).
func TestForwardsTransactionLifecycle(t *testing.T) {
	s := app.NewSession()
	rec := newTypedPayloadRecorder()
	subs := Subscribe(s, rec.sink)
	defer cancelAll(subs)

	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}

	committed := decodeSingle[wire.TransactionEventPayload](t, rec, wire.EventTransactionCommitted)
	undone := decodeSingle[wire.TransactionEventPayload](t, rec, wire.EventTransactionUndone)
	redone := decodeSingle[wire.TransactionEventPayload](t, rec, wire.EventTransactionRedone)
	if committed.Label != "Edit Parameters" || committed.Point != types.TransactionPointCurrent {
		t.Errorf("committed = %+v, want Edit Parameters at the current point", committed)
	}
	if undone.Point != types.TransactionPointPrevious || redone.Point != types.TransactionPointNext {
		t.Errorf("undo/redo points = %v/%v, want previous/next", undone.Point, redone.Point)
	}
	if committed.Document == 0 {
		t.Error("committed event must carry the document id")
	}
}

// TestForwardsFileEvents covers the file-access and file-UI relays: a dirty
// transition, a vetoed-free new-part flow, and a dialog hook answer.
func TestForwardsFileEvents(t *testing.T) {
	s := app.NewSession()
	rec := newTypedPayloadRecorder()
	subs := Subscribe(s, rec.sink)
	defer cancelAll(subs)

	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	created := decodeSingle[wire.FileDialogHookPayload](t, rec, wire.EventFileNew)
	if created.DocumentType != types.DocumentPart {
		t.Errorf("file.new payload = %+v, want a part", created)
	}

	// A new part is born dirty (no transition); clean it and dirty it again.
	d := s.ActiveDocument()
	d.ClearDirty()
	d.MarkDirty()
	dirty := decodeSingle[wire.FileDirtyEventPayload](t, rec, wire.EventFileDirty)
	if dirty.Document != uint64(d.ID()) || dirty.FullDocumentName != d.FullDocumentName() {
		t.Errorf("file.dirty payload = %+v, want document %d %q", dirty, d.ID(), d.FullDocumentName())
	}

	// An answered open-dialog hook relays the supplied path to observers.
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e app.FileOpenDialog) event.Outcome {
		e.Supply("/vault/bracket.obk")
		return event.Handle()
	})
	if path, ok := s.HookFileOpenDialog(); !ok || path != "/vault/bracket.obk" {
		t.Fatalf("hook = (%q, %v), want the supplied path", path, ok)
	}
	hook := decodeSingle[wire.FileDialogHookPayload](t, rec, wire.EventFileOpenDialog)
	if hook.FileName != "/vault/bracket.obk" {
		t.Errorf("file.openDialog payload = %+v, want the supplied path", hook)
	}
}
