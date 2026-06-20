// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/feature"

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
	if items := partNodeMenu(n); items != nil {
		return items
	}
	if items := assemblyNodeMenu(n); items != nil {
		return items
	}
	return drawingNodeMenu(n)
}

// drawingNodeMenu returns the menu for a drawing-document node kind (a view), or nil.
func drawingNodeMenu(n BrowserNode) []BrowserMenuItem {
	if n.Kind == "drawingView" {
		return drawingViewMenu(n.Select)
	}
	return nil
}

// drawingViewMenu is the Edit/Delete menu for a drawing view (browser or canvas right-click).
func drawingViewMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(DrawingViewHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Edit", Enabled: true, Invoke: func(s *Session) error { s.BeginEditDrawingView(h); return nil }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteDrawingView(h) }},
	}
}

// partNodeMenu returns the menu for a part-document node kind, or nil.
func partNodeMenu(n BrowserNode) []BrowserMenuItem {
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
	case "pointCloud":
		return pointCloudMenu(n.Select)
	case "pointCloudCrop":
		return pointCloudCropMenu(n.Select)
	default:
		return nil
	}
}

// pointCloudCropMenu offers an active toggle and Delete for a crop volume (#645).
func pointCloudCropMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(PointCloudCropHandle)
	if !ok {
		return nil
	}
	toggle := "Deactivate"
	if !h.Crop.Active() {
		toggle = "Activate"
	}
	return []BrowserMenuItem{
		{Label: toggle, Enabled: true, Invoke: func(*Session) error {
			h.Crop.SetActive(!h.Crop.Active())
			return nil
		}},
		{Label: "Delete", Enabled: true, Invoke: func(*Session) error {
			h.Cloud.Crops().Remove(h.Crop.Name())
			return nil
		}},
	}
}

// pointCloudMenu offers a Visibility toggle and Delete for an attached scan (M17-F06, #645).
func pointCloudMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(PointCloudHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Visibility", Enabled: true, Invoke: func(*Session) error {
			h.Cloud.SetVisible(!h.Cloud.Visible())
			return nil
		}},
		{Label: "Delete", Enabled: true, Invoke: func(*Session) error {
			h.Clouds.Remove(h.Cloud.Name())
			return nil
		}},
	}
}

// assemblyNodeMenu returns the menu for an assembly node kind (occurrence/representation/
// model state/assembly feature), or nil.
func assemblyNodeMenu(n BrowserNode) []BrowserMenuItem {
	switch n.Kind {
	case "occurrence":
		return occurrenceMenu(n.Select)
	case "representation":
		return representationMenu(n.Select)
	case "modelState":
		return modelStateMenu(n.Select)
	case "assemblyFeature":
		return assemblyFeatureNodeMenu(n.Select)
	default:
		return nil
	}
}

// assemblyFeatureNodeMenu offers Edit (its scalar parameters), a Suppress/Unsuppress toggle, and
// Delete for a committed assembly machining feature (#766). Edit is greyed for a feature with no
// editable parameters.
func assemblyFeatureNodeMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(AssemblyFeatureHandle)
	if !ok {
		return nil
	}
	suppressLabel := "Suppress"
	if h.Feature.Suppressed() {
		suppressLabel = "Unsuppress"
	}
	return []BrowserMenuItem{
		{Label: "Edit", Enabled: assemblyFeatureEditable(h.Feature), Invoke: func(s *Session) error { s.BeginEditAssemblyFeature(h); return nil }},
		{Label: suppressLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleAssemblyFeatureSuppressed(h.Feature) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteAssemblyFeature(h.Feature) }},
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
	items := []BrowserMenuItem{
		{Label: "Edit", Enabled: h.Occurrence.ComponentName() != "", Invoke: func(s *Session) error { return s.OpenOccurrenceDocument(h.Occurrence) }},
		{Label: groundLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleOccurrenceGrounded(h.Occurrence) }},
		{Label: suppressLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleOccurrenceSuppressed(h.Occurrence) }},
	}
	// Only a sub-assembly occurrence can be flexible (solve its components independently per
	// placement, M12-F06); the label reflects the current state.
	if h.Occurrence.SubOccurrences() != nil {
		flexLabel := "Flexible"
		if h.Occurrence.Flexible() {
			flexLabel = "Rigid"
		}
		items = append(items, BrowserMenuItem{Label: flexLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleOccurrenceFlexible(h.Occurrence) }})
	}
	return append(items, BrowserMenuItem{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteOccurrence(h.Occurrence) }})
}

// representationMenu offers Activate and Delete for a representation selected in the
// Representations folders (M12-F04).
func representationMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(RepresentationHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Activate", Enabled: true, Invoke: func(s *Session) error { return s.ActivateRepresentation(h) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteRepresentation(h) }},
	}
}

// modelStateMenu offers Activate and Delete for a model state (M12-F04).
func modelStateMenu(sel Selectable) []BrowserMenuItem {
	h, ok := sel.(ModelStateHandle)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Activate", Enabled: true, Invoke: func(s *Session) error { return s.ActivateModelState(h) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteModelState(h) }},
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
	items := []BrowserMenuItem{
		{Label: "Edit", Enabled: FeatureIsEditable(h.Feature), Invoke: func(s *Session) error { s.BeginEditFeature(h); return nil }},
		{Label: suppressLabel, Enabled: true, Invoke: func(s *Session) error { return s.ToggleFeatureSuppressed(h.Feature) }},
		{Label: "Delete", Enabled: true, Invoke: func(s *Session) error { return s.DeleteFeature(h.Feature) }},
	}
	return append(items, deriveMenuItems(h.Feature)...)
}

// deriveMenuItems adds the derive-family actions to a feature menu: Update (re-sync to the source,
// enabled only when out of date) and Break Link (freeze, enabled while still linked). A non-derive
// feature adds nothing (#767).
func deriveMenuItems(f *feature.PartFeature) []BrowserMenuItem {
	ds, ok := f.Definition().(feature.DeriveStatus)
	if !ok {
		return nil
	}
	return []BrowserMenuItem{
		{Label: "Update", Enabled: ds.OutOfDate(), Invoke: func(s *Session) error { return s.UpdateDerivedFeature(f) }},
		{Label: "Break Link", Enabled: ds.Linked(), Invoke: func(s *Session) error { return s.BreakDerivedLink(f) }},
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
