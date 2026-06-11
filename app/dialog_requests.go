// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// Host-provided dialogs for add-ins (M05-F08, #615): file open/save requests the
// head's explorer answers asynchronously, and web views presented behind a thin
// engine seam (the head shows the URL with an open-in-browser affordance until an
// embedded engine lands).

// FileDialogRequest is one queued file-dialog ask. ID keys the answer event.
type FileDialogRequest struct {
	ID          string
	Title       string
	Save        bool
	Filter      string
	FilterIndex int
	InitialDir  string
	MultiSelect bool
}

// FileDialogChosen fires (After) when the user answers a file dialog; the events
// relay forwards it as a dialog.fileChosen push event.
type FileDialogChosen struct {
	ID        string
	Paths     []string
	Cancelled bool
}

// EventID implements event.Event.
func (FileDialogChosen) EventID() event.TypeID { return tidFileDialogChosen }

// WebDialogChanged fires (After) when a web view is shown or closed.
type WebDialogChanged struct {
	ID      string
	Visible bool
}

// EventID implements event.Event.
func (WebDialogChanged) EventID() event.TypeID { return tidWebDialogChanged }

// URLOpener is the thin seam to the platform's URL opener (xdg-open & friends) —
// injected by the head per the third-party-wrapper rule, faked in tests.
type URLOpener interface {
	OpenURL(url string) error
}

// SetURLOpener installs the platform URL opener.
func (s *Session) SetURLOpener(opener URLOpener) { s.urlOpener = opener }

// OpenURL opens a URL in the platform browser (the web-view fallback affordance).
func (s *Session) OpenURL(url string) error {
	if s.urlOpener == nil {
		return fmt.Errorf("app: no URL opener configured to open %q", url)
	}
	return s.urlOpener.OpenURL(url)
}

// RequestFileDialog queues a file-dialog ask the head's explorer presents; the
// user's answer arrives as a [FileDialogChosen] event keyed by the request's ID.
func (s *Session) RequestFileDialog(req FileDialogRequest) error {
	if req.ID == "" {
		return fmt.Errorf("app: file dialog request needs an id, got %+v", req)
	}
	for _, queued := range s.fileDialogQueue {
		if queued.ID == req.ID {
			return nil // already pending; one answer serves both callers
		}
	}
	s.fileDialogQueue = append(s.fileDialogQueue, req)
	return nil
}

// PendingFileDialog returns the request the head should present (oldest first).
func (s *Session) PendingFileDialog() (FileDialogRequest, bool) {
	if len(s.fileDialogQueue) == 0 {
		return FileDialogRequest{}, false
	}
	return s.fileDialogQueue[0], true
}

// ResolveFileDialog answers a pending request with the chosen paths (or a cancel)
// and emits the event its owner observes.
func (s *Session) ResolveFileDialog(id string, paths []string, cancelled bool) error {
	for i, queued := range s.fileDialogQueue {
		if queued.ID != id {
			continue
		}
		s.fileDialogQueue = append(s.fileDialogQueue[:i], s.fileDialogQueue[i+1:]...)
		event.Emit(s.bus, event.After, FileDialogChosen{ID: id, Paths: paths, Cancelled: cancelled})
		return nil
	}
	return fmt.Errorf("app: no pending file dialog %q", id)
}

// WebViews returns the presented web views in creation order.
func (s *Session) WebViews() []wire.WebDialogSpec {
	out := make([]wire.WebDialogSpec, len(s.webViewOrder))
	for i, id := range s.webViewOrder {
		out[i] = s.webViews[id]
	}
	return out
}

// ShowWebDialog creates the web view or replaces its title/URL, emitting a
// visibility event on transitions (a new visible view is a show).
func (s *Session) ShowWebDialog(spec wire.WebDialogSpec) error {
	if spec.ID == "" || spec.URL == "" {
		return fmt.Errorf("app: web dialog needs id and url, got id=%q url=%q", spec.ID, spec.URL)
	}
	prev, existed := s.webViews[spec.ID]
	if !existed {
		s.webViewOrder = append(s.webViewOrder, spec.ID)
	}
	s.webViews[spec.ID] = spec
	if !existed && spec.Visible || existed && prev.Visible != spec.Visible {
		event.Emit(s.bus, event.After, WebDialogChanged{ID: spec.ID, Visible: spec.Visible})
	}
	return nil
}

// CloseWebDialog removes a web view, emitting a hide first when it was visible —
// also the path the head takes when the user closes the window.
func (s *Session) CloseWebDialog(id string) error {
	spec, ok := s.webViews[id]
	if !ok {
		return fmt.Errorf("app: no web dialog %q", id)
	}
	if spec.Visible {
		event.Emit(s.bus, event.After, WebDialogChanged{ID: id, Visible: false})
	}
	delete(s.webViews, id)
	for i, x := range s.webViewOrder {
		if x == id {
			s.webViewOrder = append(s.webViewOrder[:i], s.webViewOrder[i+1:]...)
			break
		}
	}
	return nil
}
