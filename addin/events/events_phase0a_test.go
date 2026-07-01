// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestPhase0aRelaysPanelReferencesChanged checks that emitting app.PanelReferencesChanged
// on the session bus delivers a wire.PanelReferencesChangedEvent JSON to the sink with the
// correct type, windowId, controlId, refs, and action fields.
func TestPhase0aRelaysPanelReferencesChanged(t *testing.T) {
	s := app.NewSession()
	if err := s.SetDockableWindow(wire.DockableWindowSpec{
		ID: "w", Title: "W", Visible: true,
		Controls: []wire.PanelControlSpec{{Kind: types.PanelReferenceList, ID: "faces"}},
	}); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}

	var rec rawRecorder
	subs := Subscribe(s, rec.sink)
	defer func() {
		for _, sub := range subs {
			sub.Cancel()
		}
	}()

	s.SetDockableWindowReferences("w", "faces", []string{"face/a"})

	rec.mu.Lock()
	defer rec.mu.Unlock()
	var ev wire.PanelReferencesChangedEvent
	found := false
	for _, raw := range rec.got {
		if json.Unmarshal([]byte(raw), &ev) == nil && ev.Type == wire.EventPanelReferencesChanged {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("panel.referencesChanged not found in sink output: %v", rec.got)
	}
	if ev.WindowId != "w" || ev.ControlId != "faces" || len(ev.Refs) != 1 || ev.Refs[0] != "face/a" || ev.Action != "set" {
		t.Errorf("event = %+v, want windowId=w controlId=faces refs=[face/a] action=set", ev)
	}
}

// TestPhase0aRelaysTaskPanelClosed checks that emitting app.TaskPanelClosed (via
// ResolveTaskPanel) delivers a wire.TaskPanelClosedEvent JSON to the sink with the
// correct type, id, and accepted fields.
func TestPhase0aRelaysTaskPanelClosed(t *testing.T) {
	s := app.NewSession()
	if err := s.ShowTaskPanel(wire.TaskPanelSpec{ID: "fix", Title: "Fixed"}); err != nil {
		t.Fatalf("ShowTaskPanel: %v", err)
	}

	var rec rawRecorder
	subs := Subscribe(s, rec.sink)
	defer func() {
		for _, sub := range subs {
			sub.Cancel()
		}
	}()

	if err := s.ResolveTaskPanel("fix", true); err != nil {
		t.Fatalf("ResolveTaskPanel: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	var ev wire.TaskPanelClosedEvent
	found := false
	for _, raw := range rec.got {
		if json.Unmarshal([]byte(raw), &ev) == nil && ev.Type == wire.EventTaskPanelClosed {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("taskPanel.closed not found in sink output: %v", rec.got)
	}
	if ev.ID != "fix" || !ev.Accepted {
		t.Errorf("event = %+v, want id=fix accepted=true", ev)
	}
}
