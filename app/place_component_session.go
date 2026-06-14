// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/doc"

// Session bridge for the Place Component tool's UI (#763): the head's file dialog feeds the
// chosen component document to the running tool, and the viewport reads whether a component is
// still awaited (the cue to open that dialog) — without touching the tool's internals, the same
// shape as the Extrude and Offset Plane bridges.

// ActivePlaceComponent returns the running Place Component tool, or nil when the active tool is
// not a place-component (or there is none).
func (s *Session) ActivePlaceComponent() *PlaceComponentTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*PlaceComponentTool)
	return t
}

// SetPlaceComponentDocument hands the running Place tool the component the user chose in the
// file dialog. A no-op when no Place tool is active, so a late dialog result is harmless.
func (s *Session) SetPlaceComponentDocument(d *doc.Document) {
	if t := s.ActivePlaceComponent(); t != nil {
		t.SetComponentDocument(d)
	}
}

// PlaceComponentAwaitingFile reports that a Place tool is active but has no component yet — the
// cue for the head to open its file dialog.
func (s *Session) PlaceComponentAwaitingFile() bool {
	t := s.ActivePlaceComponent()
	return t != nil && t.component == nil
}
