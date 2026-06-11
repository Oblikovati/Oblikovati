// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// recorder collects forwarded events (thread-safe; events may fire from any goroutine).
type recorder struct {
	mu  sync.Mutex
	got []wireEvent
}

func (r *recorder) sink(b []byte) {
	var ev wireEvent
	if err := json.Unmarshal(b, &ev); err != nil {
		return
	}
	r.mu.Lock()
	r.got = append(r.got, ev)
	r.mu.Unlock()
}

func (r *recorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.got))
	for i, e := range r.got {
		out[i] = e.Type
	}
	return out
}

func TestForwardsDocumentAndCommandEvents(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	if err := s.Commands().Add(app.NewCommand("test.noop", "Noop", "Test", func(*app.Session) error { return nil })); err != nil {
		t.Fatalf("add command: %v", err)
	}
	var rec recorder
	subs := Subscribe(s, rec.sink)
	if len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}

	if _, err := s.Workspace().Add(doc.Part, "ev.obk", true); err != nil {
		t.Fatalf("add document: %v", err)
	}
	if err := s.Execute("test.noop"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := rec.types()
	if !has(got, "document.created") {
		t.Errorf("missing document.created in %v", got)
	}
	if !has(got, "command.ended") {
		t.Errorf("missing command.ended in %v", got)
	}
}

func TestDocumentCreatedCarriesName(t *testing.T) {
	s := app.NewSession()
	var rec recorder
	Subscribe(s, rec.sink)
	if _, err := s.Workspace().Add(doc.Part, "bracket.obk", true); err != nil {
		t.Fatalf("add document: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.got) != 1 || rec.got[0].Document != "bracket" || rec.got[0].ID == 0 {
		t.Fatalf("event = %+v, want document=bracket with nonzero id", rec.got)
	}
}

func has(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// rawRecorder collects the raw JSON the sink receives, for events whose shapes are
// wire DTOs rather than the package's own wireEvent (M05-F03 push events).
type rawRecorder struct {
	mu  sync.Mutex
	got []string
}

func (r *rawRecorder) sink(b []byte) {
	r.mu.Lock()
	r.got = append(r.got, string(b))
	r.mu.Unlock()
}

// TestForwardsBrowserPaneAndDockableWindowEvents checks the M05-F03 UI events reach
// the sink as their wire DTOs: a pane-node gesture as browser.node and a window
// visibility change as dockableWindow.changed.
func TestForwardsBrowserPaneAndDockableWindowEvents(t *testing.T) {
	s := app.NewSession()
	var rec rawRecorder
	subs := Subscribe(s, rec.sink)
	defer func() {
		for _, sub := range subs {
			sub.Cancel()
		}
	}()

	if err := s.BrowserPanes().Set(wire.BrowserPaneSpec{
		ID: "sim", Title: "Simulation",
		Nodes: []wire.BrowserNodeSpec{{ID: "f1", Label: "Force"}},
	}); err != nil {
		t.Fatalf("Set pane: %v", err)
	}
	if err := s.ActivateBrowserPaneNode("sim", "f1", app.BrowserGestureSelect); err != nil {
		t.Fatalf("ActivateBrowserPaneNode: %v", err)
	}
	if err := s.SetDockableWindow(wire.DockableWindowSpec{ID: "w", Title: "W", Visible: true}); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	var node wire.BrowserNodeEvent
	var win wire.DockableWindowChangedEvent
	for _, raw := range rec.got {
		_ = json.Unmarshal([]byte(raw), &node)
		if node.Type == wire.EventBrowserNode {
			break
		}
	}
	if node.Pane != "sim" || node.Node != "f1" || node.Gesture != "select" {
		t.Errorf("browser.node = %+v, want sim/f1/select", node)
	}
	found := false
	for _, raw := range rec.got {
		if json.Unmarshal([]byte(raw), &win) == nil && win.Type == wire.EventDockableWindowChanged {
			found = true
			break
		}
	}
	if !found || win.ID != "w" || !win.Visible {
		t.Errorf("dockableWindow.changed = %+v (found=%v), want visible w", win, found)
	}
}

// TestClientOperationServicesFlavoredDocuments checks a subtyped document's
// lifecycle additionally reaches its owner as client.operation (M05-F15).
func TestClientOperationServicesFlavoredDocuments(t *testing.T) {
	s := app.NewSession()
	if err := s.RegisterDocumentSubType(app.DocumentSubType{ID: "com.x.study", BaseType: doc.Part}); err != nil {
		t.Fatalf("RegisterDocumentSubType: %v", err)
	}
	var rec rawRecorder
	subs := Subscribe(s, rec.sink)
	defer func() {
		for _, sub := range subs {
			sub.Cancel()
		}
	}()

	d, err := s.Workspace().Add(doc.Part, "study.obk", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.StampDocumentSubType(d, "com.x.study"); err != nil {
		t.Fatalf("Stamp: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	var op wire.ClientOperationEvent
	found := false
	for _, raw := range rec.got {
		if json.Unmarshal([]byte(raw), &op) == nil && op.Type == wire.EventClientOperation {
			found = true
			break
		}
	}
	if !found || op.SubType != "com.x.study" {
		t.Fatalf("client.operation = (%+v, found=%v), want the study serviced", op, found)
	}
}
