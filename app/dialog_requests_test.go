// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// FakeURLOpener is a named app.URLOpener recording what it was asked to open.
type FakeURLOpener struct{ opened []string }

func (f *FakeURLOpener) OpenURL(url string) error {
	f.opened = append(f.opened, url)
	return nil
}

func TestFileDialogRequestResolveRoundTrip(t *testing.T) {
	s := NewSession()
	var chosen []FileDialogChosen
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e FileDialogChosen) event.Outcome {
		chosen = append(chosen, e)
		return event.Continue()
	})

	req := FileDialogRequest{ID: "sim.report", Title: "Save report", Save: true}
	if err := s.RequestFileDialog(req); err != nil {
		t.Fatalf("RequestFileDialog: %v", err)
	}
	if err := s.RequestFileDialog(req); err != nil { // duplicate id: one answer serves both
		t.Fatalf("RequestFileDialog(dup): %v", err)
	}
	pending, ok := s.PendingFileDialog()
	if !ok || pending.ID != "sim.report" || !pending.Save {
		t.Fatalf("Pending = (%+v, %v), want the save request", pending, ok)
	}

	if err := s.ResolveFileDialog("sim.report", []string{"/tmp/report.html"}, false); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(chosen) != 1 || chosen[0].Paths[0] != "/tmp/report.html" || chosen[0].Cancelled {
		t.Fatalf("events = %+v, want the chosen path", chosen)
	}
	if _, ok := s.PendingFileDialog(); ok {
		t.Error("queue should be empty after resolve")
	}
	if err := s.ResolveFileDialog("sim.report", nil, true); err == nil {
		t.Error("resolving twice should fail")
	}
}

func TestWebDialogLifecycleEmitsVisibility(t *testing.T) {
	s := NewSession()
	var changes []WebDialogChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e WebDialogChanged) event.Outcome {
		changes = append(changes, e)
		return event.Continue()
	})

	view := wire.WebDialogSpec{ID: "docs", Title: "Help", URL: "https://example.org", Visible: true}
	if err := s.ShowWebDialog(view); err != nil {
		t.Fatalf("ShowWebDialog: %v", err)
	}
	if err := s.ShowWebDialog(view); err != nil { // same visibility ⇒ silent
		t.Fatalf("ShowWebDialog(re-show): %v", err)
	}
	if err := s.CloseWebDialog("docs"); err != nil {
		t.Fatalf("CloseWebDialog: %v", err)
	}
	if len(s.WebViews()) != 0 {
		t.Error("view survived Close")
	}
	want := []WebDialogChanged{{ID: "docs", Visible: true}, {ID: "docs", Visible: false}}
	if len(changes) != 2 || changes[0] != want[0] || changes[1] != want[1] {
		t.Fatalf("events = %+v, want shown then hidden", changes)
	}
	if err := s.ShowWebDialog(wire.WebDialogSpec{ID: "x"}); err == nil {
		t.Error("a web dialog without a url should fail")
	}
}

func TestOpenURLUsesInjectedOpener(t *testing.T) {
	s := NewSession()
	if err := s.OpenURL("https://example.org"); err == nil {
		t.Fatal("OpenURL without an opener should fail")
	}
	opener := &FakeURLOpener{}
	s.SetURLOpener(opener)
	if err := s.OpenURL("https://example.org"); err != nil {
		t.Fatalf("OpenURL: %v", err)
	}
	if len(opener.opened) != 1 || opener.opened[0] != "https://example.org" {
		t.Errorf("opened = %v, want the url routed through the seam", opener.opened)
	}
}
