// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/sketch"
)

// CreateSketchTool is Inventor's "Create 2D Sketch" interaction: after the command
// starts it, the user picks a sketch host — a work plane OR a planar face, in the 3D
// view or the browser — and the sketch is created and opened on that plane the instant
// it is clicked (auto-commit). It restricts the selection filter to the valid hosts
// (work planes and faces) while active, so only those highlight/pick — and a click on a
// face starts the sketch on that face rather than on a plane hidden behind it.
type CreateSketchTool struct {
	plane      sketch.Plane
	picked     bool
	prevFilter *SelectionFilter
}

// NewCreateSketchTool returns a sketch-host selection tool.
func NewCreateSketchTool() *CreateSketchTool { return &CreateSketchTool{} }

// Name implements [Tool].
func (t *CreateSketchTool) Name() string { return "Create 2D Sketch" }

// Start filters selection to the valid sketch hosts — work planes and planar faces — so
// both highlight and pick while the tool is active (a face in front wins over the origin
// plane behind it, which is why the face must be accepted, not just the plane).
func (t *CreateSketchTool) Start(s *Session) {
	t.prevFilter = s.Selection().Filter()
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane, SelectFace))
}

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
	s.Selection().SetFilter(t.prevFilter)
	if !t.picked {
		return errors.New("create sketch: no work plane or planar face selected")
	}
	_, err := s.CreateSketch(t.plane)
	return err
}

// Cancel restores the previous selection filter with no change.
func (t *CreateSketchTool) Cancel(s *Session) { s.Selection().SetFilter(t.prevFilter) }

// Prompt guides the user to pick a sketch plane (Inventor's status-bar prompt).
func (t *CreateSketchTool) Prompt(*Session) string {
	return "Select a plane or planar face to create the sketch on"
}
