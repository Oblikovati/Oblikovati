// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/sketch"
)

// CreateSketchTool is Inventor's "Create 2D Sketch" interaction: after the command
// starts it, the user picks a sketch host — a work plane OR a planar face, in the 3D
// view or the browser — and the sketch is created and opened on that plane the instant
// it is clicked (auto-commit). It is the canonical selection-engine consumer: it DECLARES
// that it picks work planes and planar faces (AcceptedKinds), so the host installs that
// filter and highlights both — a click on a face starts the sketch on that face rather
// than on a plane hidden behind it (the bug that motivated the engine). See ADR-0041.
type CreateSketchTool struct {
	plane  sketch.Plane
	picked bool
}

// NewCreateSketchTool returns a sketch-host selection tool.
func NewCreateSketchTool() *CreateSketchTool { return &CreateSketchTool{} }

// Name implements [Tool].
func (t *CreateSketchTool) Name() string { return "Create 2D Sketch" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *CreateSketchTool) Start(*Session) {}

// RevealsDatumHosts marks this as a datum-host pick so the host reveals the normally-hidden
// origin planes (XY/XZ/YZ) — otherwise a brand-new part offers nothing to click in the viewport
// and the only route to the first sketch is the browser Origin folder (#1752).
func (t *CreateSketchTool) RevealsDatumHosts() bool { return true }

// AcceptedKinds declares the valid sketch hosts: work planes and planar faces. Declaring the
// face here (not just the plane) is what lets a face in front win over the origin plane behind it.
func (t *CreateSketchTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectWorkPlane, SelectFace}
}

// Picks reports nothing: the tool auto-commits on the pick, so there is never a lingering
// selection to highlight (the hover preselect already lights the host under the cursor).
func (t *CreateSketchTool) Picks() []Selectable { return nil }

// Pick records the chosen sketch host: a work plane directly, or the plane of a picked
// planar face. A non-planar face is ignored (it cannot host a sketch).
func (t *CreateSketchTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case WorkPlaneHandle:
		t.plane, t.picked = h.Plane.Plane(), true
	case FaceHandle:
		if pl, ok := sketchPlaneFromFace(h); ok {
			t.plane, t.picked = pl, true
		}
	}
}

// CanCommit is true once a host plane has been picked.
func (t *CreateSketchTool) CanCommit() bool { return t.picked }

// AutoCommitOnPick makes the tool enter the sketch as soon as a host is clicked.
func (t *CreateSketchTool) AutoCommitOnPick() bool { return true }

// Commit creates the sketch on the picked plane and opens the sketch environment.
func (t *CreateSketchTool) Commit(s *Session) error {
	if !t.picked {
		return errors.New("create sketch: no work plane or planar face selected")
	}
	_, err := s.CreateSketch(t.plane)
	return err
}

// Cancel abandons the tool; the engine restores the ambient filter.
func (t *CreateSketchTool) Cancel(*Session) {}

// Prompt guides the user to pick a sketch plane (Inventor's status-bar prompt).
func (t *CreateSketchTool) Prompt(*Session) string {
	return "Select a plane or planar face to create the sketch on"
}
