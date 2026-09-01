// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// TestSetDockableWindowReferencesUpdatesAndEmits covers the referenceList control path:
// stored Rows are replaced and a PanelReferencesChanged event is emitted.
func TestSetDockableWindowReferencesUpdatesAndEmits(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.SetDockableWindow(wire.DockableWindowSpec{
		ID: "w", Title: "W", Visible: true,
		Controls: []wire.PanelControlSpec{{Kind: types.PanelReferenceList, ID: "faces"}},
	}); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}

	var captured []PanelReferencesChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e PanelReferencesChanged) event.Outcome {
		captured = append(captured, e)
		return event.Continue()
	})

	s.SetDockableWindowReferences("w", "faces", []string{"face/aaa", "face/bbb"})

	if len(captured) != 1 {
		t.Fatalf("want 1 PanelReferencesChanged event, got %d", len(captured))
	}
	ev := captured[0]
	if ev.WindowID != "w" || ev.ControlID != "faces" || ev.Action != "set" || len(ev.Refs) != 2 || ev.Refs[1] != "face/bbb" {
		t.Fatalf("event = %+v, want windowID=w controlID=faces action=set refs=[face/aaa face/bbb]", ev)
	}
	stored := s.DockableWindows().List()[0].Controls[0]
	if len(stored.Rows) != 2 || stored.Rows[0].Ref != "face/aaa" {
		t.Fatalf("stored rows = %+v, want two rows starting face/aaa", stored.Rows)
	}
}
