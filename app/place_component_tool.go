// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// PlaceComponentTool instances a component document into the active assembly: each click on the
// assembly ground plane drops one occurrence at that point and records it as its own undo step,
// leaving the tool open to drop more — the multi-place behavior of the reference Place Component
// command (M11, #763). The component document is chosen before placing begins (the head's file
// dialog, or SetComponentDocument in tests), so the tool itself only turns clicks into
// placements and its logic is testable without the UI.
//
// Example:
//
//	tool := NewPlaceComponentTool()
//	tool.SetComponentDocument(widget) // chosen in the file dialog
//	s.StartTool(tool)
//	s.Click(px, py)                   // drops widget:1 on the ground plane
type PlaceComponentTool struct {
	component *doc.Document // the document to instance; nil until chosen
	placed    int           // occurrences dropped so far, for unique instance names
}

var (
	_ Tool           = (*PlaceComponentTool)(nil)
	_ PlaneClickTool = (*PlaceComponentTool)(nil)
)

// NewPlaceComponentTool returns an idle Place tool. Its component must be set
// (SetComponentDocument) before a click places anything.
func NewPlaceComponentTool() *PlaceComponentTool { return &PlaceComponentTool{} }

// Name implements [Tool].
func (t *PlaceComponentTool) Name() string { return "Place Component" }

// SetComponentDocument chooses the document the tool instances. The head calls this once the
// user picks a file; tests inject an already-open component the same way.
func (t *PlaceComponentTool) SetComponentDocument(d *doc.Document) { t.component = d }

// Start implements [Tool]. Placement reads ground-plane clicks rather than entity picks, so the
// tool clears the current selection and waits for ClickAt.
func (t *PlaceComponentTool) Start(s *Session) { s.selection.Clear() }

// Pick implements [Tool]. Placement is driven by ground-plane clicks (ClickAt), not entity
// picks, so a pick is ignored — snapping a placement to existing geometry is a later refinement.
func (t *PlaceComponentTool) Pick(_ *Session, _ Selectable) {}

// ClickAt drops one instance of the component at the clicked point on the assembly ground plane
// (Z = 0) and records it as an undo step, leaving the tool open for the next drop
// ([PlaneClickTool]). A click before a component is chosen, off an active assembly, or on a
// grazing ray that misses the plane is ignored with a notice rather than placing at a bogus
// location.
func (t *PlaceComponentTool) ClickAt(s *Session, px, py float64) {
	if t.component == nil {
		s.notice = "Place Component: choose a component first"
		return
	}
	asm, err := activeAssembly(s)
	if err != nil {
		s.notice = err.Error()
		return
	}
	at, ok := groundPoint(s.Camera(), px, py)
	if !ok {
		s.notice = "Place Component: click within the ground plane"
		return
	}
	t.placed++
	name := fmt.Sprintf("%s:%d", t.component.DisplayName(), t.placed)
	if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), t.component, name, math.Translation4(at.AsVector())); err != nil {
		t.placed-- // unwind the count so the next click reuses the name a failed placement skipped
		s.notice = err.Error()
		return
	}
	s.recordEdit(asm, "Place Component")
}

// CanCommit reports the tool has placed at least one instance, enabling OK to finish.
func (t *PlaceComponentTool) CanCommit() bool { return t.placed > 0 }

// Commit finishes the tool. Each placement was already applied and recorded on its click, so
// finishing makes no further edit — it just closes the tool.
func (t *PlaceComponentTool) Commit(_ *Session) error { return nil }

// Cancel stops the tool. Instances dropped before cancelling remain (each is its own undo step),
// matching the reference command where Esc keeps what was already placed.
func (t *PlaceComponentTool) Cancel(_ *Session) {}

// groundRayEpsilon is the |ray.Z| below which the view is treated as edge-on to the ground
// plane, so no stable intersection exists.
const groundRayEpsilon = 1e-9

// groundPoint intersects the pixel ray with the assembly ground plane (Z = 0), the default
// placement datum. It returns false for a ray parallel to the plane (a grazing, edge-on view)
// or whose hit lies behind the camera, so a click that cannot resolve to a ground point is
// dropped rather than placing at a bogus location.
func groundPoint(cam scene.Camera, px, py float64) (math.Point3, bool) {
	origin, dir := cam.RayThrough(px, py)
	if dir.Z > -groundRayEpsilon && dir.Z < groundRayEpsilon {
		return math.Point3{}, false
	}
	tHit := -origin.Z / dir.Z
	if tHit < 0 {
		return math.Point3{}, false
	}
	return origin.TranslateBy(dir.Scale(tHit)), true
}
