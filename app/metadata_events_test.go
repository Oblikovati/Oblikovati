// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
)

// TestRenameFeatureEmitsObjectRenamed checks a session-level feature rename fires exactly one
// object.renamed carrying the feature kind, its key, and the old and new names (#1644) — the seam
// both the UI and the wire router go through, so an add-in observes UI renames too.
func TestRenameFeatureEmitsObjectRenamed(t *testing.T) {
	s, def := extrudedBoxPart(t)
	f := def.Features().Item(0)
	var got ObjectRenamed
	fired := 0
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e ObjectRenamed) event.Outcome {
		got, fired = e, fired+1
		return event.Continue()
	})
	old := f.Name()
	if err := s.RenameFeature(f, "Base Boss"); err != nil {
		t.Fatalf("RenameFeature: %v", err)
	}
	if fired != 1 {
		t.Fatalf("object.renamed fired %d times, want exactly 1", fired)
	}
	if got.Kind != types.ObjectKindFeature || got.OldName != old || got.NewName != "Base Boss" {
		t.Errorf("event = %+v, want feature %q->%q", got, old, "Base Boss")
	}
	if got.Key != featureMetaKey(f) {
		t.Errorf("event key = %q, want the feature key %q", got.Key, featureMetaKey(f))
	}
}

// TestSetFeatureSuppressedEmitsPropertyChanged checks toggling suppression fires one property.changed
// carrying the old and new boolean state, and that an idempotent set fires nothing (#1644).
func TestSetFeatureSuppressedEmitsPropertyChanged(t *testing.T) {
	s, def := extrudedBoxPart(t)
	f := def.Features().Item(0)
	var got PropertyChanged
	fired := 0
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e PropertyChanged) event.Outcome {
		got, fired = e, fired+1
		return event.Continue()
	})
	if err := s.SetFeatureSuppressed(f, true); err != nil {
		t.Fatalf("SetFeatureSuppressed: %v", err)
	}
	if fired != 1 {
		t.Fatalf("property.changed fired %d times, want exactly 1", fired)
	}
	if got.Property != "suppressed" || got.OldValue != "false" || got.NewValue != "true" {
		t.Errorf("event = %+v, want suppressed false->true", got)
	}
	if err := s.SetFeatureSuppressed(f, true); err != nil { // idempotent: already suppressed
		t.Fatalf("repeated SetFeatureSuppressed: %v", err)
	}
	if fired != 1 {
		t.Errorf("an idempotent suppress fired a spurious property.changed (fired=%d)", fired)
	}
}

// TestMetadataEventIDsAreStable pins the wire-facing event type ids: an add-in relay routes on them,
// so a silent change would misroute every metadata event. They live in the 0x08xx modeling block.
func TestMetadataEventIDsAreStable(t *testing.T) {
	if got := (ObjectRenamed{}).EventID(); got != tidObjectRenamed {
		t.Errorf("ObjectRenamed.EventID() = %#x, want %#x", got, tidObjectRenamed)
	}
	if got := (PropertyChanged{}).EventID(); got != tidPropertyChanged {
		t.Errorf("PropertyChanged.EventID() = %#x, want %#x", got, tidPropertyChanged)
	}
	if tidObjectRenamed == tidPropertyChanged {
		t.Errorf("the two metadata events share type id %#x; they must be distinct to route", tidObjectRenamed)
	}
}

// TestEmitObjectRenamedKeysToZeroWithoutActiveDocument checks the no-document branch of activeDocID:
// a session with nothing open still emits (keyed to the zero doc id) rather than panicking, so an
// early rename before any document is open is observable, not a crash.
func TestEmitObjectRenamedKeysToZeroWithoutActiveDocument(t *testing.T) {
	s := NewSession()
	if s.ActiveDocument() != nil {
		t.Fatalf("a fresh session must have no active document")
	}
	var got ObjectRenamed
	fired := 0
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e ObjectRenamed) event.Outcome {
		got, fired = e, fired+1
		return event.Continue()
	})
	s.emitObjectRenamed(types.ObjectKindBody, "b1", "Old", "New")
	if fired != 1 {
		t.Fatalf("object.renamed fired %d times, want exactly 1", fired)
	}
	if got.Document != 0 {
		t.Errorf("event document id = %d, want 0 (no active document)", got.Document)
	}
}
