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
	default:
		return nil
	}
}

// sketchMenu offers Edit Sketch, a Visibility toggle, and Delete.
func sketchMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(SketchHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Edit Sketch", Enabled: true, Invoke: func(s *Session) error { return s.EditSketch(h.Sketch) }},
		{Label: "Visibility", Enabled: true, Invoke: func(*Session) error {
			h.Sketch.SetVisible(!h.Sketch.Visible())
			return nil
		}},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteSketch(h.Sketch) }},
	}
}

// featureMenu offers a Suppress/Unsuppress toggle (label reflects current state) and Delete.
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
