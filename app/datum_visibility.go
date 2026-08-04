// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/feature"

// Toggling a datum's visibility (#2016). A work plane, axis and point each carry their own
// Visible flag, and the browser's Visibility menu item and the V shortcut are the two ways to
// flip it. Both used to be written per datum kind, which is how the origin Center Point ended up
// with no menu at all and the shortcut ended up reaching only planes: nothing tied the three
// kinds together, so each new one had to remember to opt in.

// VisibleDatum is a datum whose visibility the user can toggle — a work plane, axis or point.
// Naming the shared capability is what lets one menu item and one shortcut serve all three.
type VisibleDatum interface {
	Visible() bool
	SetVisible(bool)
}

// Every togglable datum satisfies it; a new datum kind that forgets to breaks the build here
// rather than silently shipping without a Visibility toggle.
var (
	_ VisibleDatum = (*feature.WorkPlane)(nil)
	_ VisibleDatum = (*feature.WorkAxis)(nil)
	_ VisibleDatum = (*feature.WorkPoint)(nil)
)

// selectedDatum returns the datum a selection handle refers to, if it is one whose visibility
// can be toggled.
func selectedDatum(sel Selectable) (VisibleDatum, bool) {
	switch h := sel.(type) {
	case WorkPlaneHandle:
		return h.Plane, true
	case WorkAxisHandle:
		return h.Axis, true
	case WorkPointHandle:
		return h.Point, true
	}
	return nil, false
}

// SelectedDatums returns every selected work plane, axis and point, in selection order — the
// targets of the visibility shortcut.
func (s *Session) SelectedDatums() []VisibleDatum {
	var out []VisibleDatum
	for _, it := range s.selection.Items() {
		if d, ok := selectedDatum(it); ok {
			out = append(out, d)
		}
	}
	return out
}

// ToggleSelectedDatumVisibility flips the Visible flag of every selected datum — the action
// behind the browser's Visibility menu item and the V keyboard shortcut. It returns how many it
// changed, so a caller can tell a no-op selection from a real toggle.
//
//	n := s.ToggleSelectedDatumVisibility()
func (s *Session) ToggleSelectedDatumVisibility() int {
	datums := s.SelectedDatums()
	for _, d := range datums {
		d.SetVisible(!d.Visible())
	}
	return len(datums)
}
