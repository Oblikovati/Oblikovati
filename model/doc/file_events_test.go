// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/event"
)

// TestFileResolutionSuppliesSubstitutePath: a Before handler answering the
// resolution hook redirects a failed open to the supplied name (the moved-file
// case, M04-F05); the After phase reports the outcome to observers.
func TestFileResolutionSuppliesSubstitutePath(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store, nil)
	moved, _ := ws.Add(Part, "/new/home/bracket.obk", true)
	if err := ws.Save(moved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := ws.Close(moved, true); err != nil {
		t.Fatalf("Close: %v", err)
	}

	event.Subscribe(ws.Events(), event.Before, func(_ event.Context, e FileResolution) event.Outcome {
		if e.RequestedName == "/old/home/bracket.obk" {
			e.Resolve("/new/home/bracket.obk")
		}
		return event.Handle()
	})
	var observed []FileResolution
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e FileResolution) event.Outcome {
		observed = append(observed, e)
		return event.Continue()
	})

	d, err := ws.Open("/old/home/bracket.obk", true)
	if err != nil {
		t.Fatalf("Open with a resolving handler: %v", err)
	}
	if d.FullDocumentName() != "/new/home/bracket.obk" {
		t.Errorf("opened %q, want the substitute /new/home/bracket.obk", d.FullDocumentName())
	}
	if len(observed) != 1 || observed[0].Resolved() != "/new/home/bracket.obk" {
		t.Errorf("After observations = %+v, want one carrying the substitute", observed)
	}
}

// TestFileResolutionUnansweredStillFails: with no handler (or no answer) the
// open fails with the original error and the After event reports no substitute.
func TestFileResolutionUnansweredStillFails(t *testing.T) {
	ws := NewWorkspace(newFakeStore(), nil)
	var observed []FileResolution
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e FileResolution) event.Outcome {
		observed = append(observed, e)
		return event.Continue()
	})

	if _, err := ws.Open("/nowhere/gone.obk", true); err == nil {
		t.Fatal("an unresolved open must fail")
	}
	if len(observed) != 1 || observed[0].Resolved() != "" {
		t.Errorf("After observations = %+v, want one with no substitute", observed)
	}
}

// TestFileResolutionFirstAnswerSticks: a second handler cannot overwrite the
// first handler's substitute.
func TestFileResolutionFirstAnswerSticks(t *testing.T) {
	e := newFileResolution("a.obk")
	e.Resolve("first.obk")
	e.Resolve("second.obk")
	if e.Resolved() != "first.obk" {
		t.Errorf("Resolved() = %q, want the first answer to stick", e.Resolved())
	}
}

// TestFileDirtyFiresOnCleanToDirtyTransition: only the first MarkDirty after
// open/save announces; re-marking an already dirty document is silent.
func TestFileDirtyFiresOnCleanToDirtyTransition(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store, nil)
	d, err := ws.Add(Part, "Part1", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	var dirtied []FileDirty
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e FileDirty) event.Outcome {
		dirtied = append(dirtied, e)
		return event.Continue()
	})

	d.MarkDirty() // already dirty from Add: no transition, no event
	if len(dirtied) != 0 {
		t.Fatalf("marking an already-dirty document fired %d events, want 0", len(dirtied))
	}
	if err := ws.Save(d); err != nil { // save cleans
		t.Fatalf("Save: %v", err)
	}
	d.MarkDirty()
	d.MarkDirty()
	if len(dirtied) != 1 || dirtied[0].Document != d {
		t.Fatalf("dirty events = %d, want exactly 1 for the clean→dirty transition", len(dirtied))
	}
}
