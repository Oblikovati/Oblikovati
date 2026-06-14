// SPDX-License-Identifier: GPL-2.0-only

package app

// The model browser's right-click menu. Inventor shows a different set of commands per
// node type; BrowserMenu returns that set as plain data so the head (head/ui) renders it
// without knowing the model — each item carries the closure that performs the action.

// BrowserMenuItem is one entry in a node's context menu. A disabled item is shown greyed.
// Invoke performs the action against the session; a nil Invoke is a non-actionable label.
type BrowserMenuItem struct {
	Label   string
	Enabled bool
	Invoke  func(*Session) error
}

// BrowserMenu returns the context-menu entries for a node, keyed on its Kind. Folder and
// other non-actionable nodes return nil (no menu). The entries close over the node's
// concrete handle, so a wrong handle/kind pairing safely yields no menu.
func BrowserMenu(n BrowserNode) []BrowserMenuItem {
	switch n.Kind {
	case "sketch":
		return sketchMenu(n.Select)
	case "feature":
		return featureMenu(n.Select)
	case "workplane":
		return workPlaneMenu(n.Select)
	case "workaxis":
		return workAxisMenu(n.Select)
	case "body":
		return bodyMenu(n.Select)
	case "occurrence":
		return occurrenceMenu(n.Select)
	default:
		return nil
	}
}

// occurrenceMenu offers a Ground/Unground toggle, a Suppress/Unsuppress toggle, and Delete for a
// placed component occurrence (#764). Each label reflects the occurrence's current state.
func occurrenceMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(OccurrenceHandle)
	if !ok {
		return nil
	}
	groundLabel := "Ground"
	if h.Occurrence.Grounded() {
		groundLabel = "Unground"
	}
	suppressLabel := "Suppress"
	if h.Occurrence.Suppressed() {
		suppressLabel = "Unsuppress"
	}
	return []BrowserMenuItem{
		{Label: groundLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleOccurrenceGrounded(h.Occurrence) }},
		{Label: suppressLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleOccurrenceSuppressed(h.Occurrence) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteOccurrence(h.Occurrence) }},
	}
}

// sketchMenu offers Edit Sketch, a Share/Unshare toggle, a Visibility toggle, and Delete.
// Share Sketch (Inventor's command) keeps the sketch at the browser's top level even after a
// feature consumes it, and lets several features consume it (issue #132); the label reflects
// the current state.
func sketchMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(SketchHandle)
	if !ok {
		return nil
	}
	shareLabel := "Share Sketch"
	if h.Sketch.Shared() {
		shareLabel = "Unshare Sketch"
	}
	return []BrowserMenuItem{
		{Label: "Edit Sketch", Enabled: true, Invoke: func(s *Session) error { return s.EditSketch(h.Sketch) }},
		{Label: shareLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleSketchShared(h.Sketch) }},
		{Label: "Visibility", Enabled: true, Invoke: func(*Session) error {
			h.Sketch.SetVisible(!h.Sketch.Visible())
			return nil
		}},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteSketch(h.Sketch) }},
	}
}

// featureMenu offers Edit (opens the parameter editor, like double-click), a
// Suppress/Unsuppress toggle (label reflects current state), and Delete. Edit is greyed for a
// feature with no editable parameters (e.g. a mirror, which has only geometric references).
func featureMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(FeatureHandle)
	if !ok {
		return nil
	}
	suppressLabel := "Suppress"
	if h.Feature.Suppressed() {
		suppressLabel = "Unsuppress"
	}
	return []BrowserMenuItem{
		{Label: "Edit", Enabled: FeatureIsEditable(h.Feature), Invoke: func(s *Session) error { s.BeginEditFeature(h); return nil }},
		{Label: suppressLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleFeatureSuppressed(h.Feature) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteFeature(h.Feature) }},
	}
}

// workPlaneMenu offers New Sketch on the plane and a Visibility toggle.
func workPlaneMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(WorkPlaneHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "New Sketch", Enabled: true, Invoke: func(s *Session) error {
			_, err := s.CreateSketch(h.Plane.Plane())
			return err
		}},
		{Label: "Visibility", Enabled: true, Invoke: func(*Session) error {
			h.Plane.SetVisible(!h.Plane.Visible())
			return nil
		}},
	}
}

func workAxisMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(WorkAxisHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{{Label: "Visibility", Enabled: true, Invoke: func(*Session) error {
		h.Axis.SetVisible(!h.Axis.Visible())
		return nil
	}}}
}

func bodyMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(BodyHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{{Label: "Visibility", Enabled: true, Invoke: func(s *Session) error {
		s.ToggleBodyVisibility(h.Body)
		return nil
	}}}
}
