// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/model/feature"
)

// CreateSketchTool is Inventor's "Create 2D Sketch" interaction: after the command
// starts it, the user picks the work plane (or, later, a planar face) to sketch on —
// in the 3D view or the browser — and the sketch is created and opened on that plane
// the instant it is clicked (auto-commit). It restricts the selection filter to work
// planes while active, so only valid sketch hosts highlight/pick.
type CreateSketchTool struct {
	plane      *feature.WorkPlane
	prevFilter *SelectionFilter
}

// NewCreateSketchTool returns a sketch-plane selection tool.
func NewCreateSketchTool() *CreateSketchTool { return &CreateSketchTool{} }

// Name implements [Tool].
func (t *CreateSketchTool) Name() string { return "Create 2D Sketch" }

// Start filters selection to work planes (the valid sketch hosts) for the pick.
func (t *CreateSketchTool) Start(s *Session) {
	t.prevFilter = s.Selection().Filter()
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane))
}

// Pick records the chosen work plane.
func (t *CreateSketchTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(WorkPlaneHandle); ok {
		t.plane = h.Plane
	}
}

// CanCommit is true once a plane has been picked.
func (t *CreateSketchTool) CanCommit() bool { return t.plane != nil }

// AutoCommitOnPick makes the tool enter the sketch as soon as a plane is clicked.
func (t *CreateSketchTool) AutoCommitOnPick() bool { return true }

// Commit creates the sketch on the picked plane and opens the sketch environment.
func (t *CreateSketchTool) Commit(s *Session) error {
	s.Selection().SetFilter(t.prevFilter)
	if t.plane == nil {
		return errors.New("create sketch: no plane selected")
	}
	_, err := s.CreateSketch(t.plane.Plane())
	return err
}

// Cancel restores the previous selection filter with no change.
func (t *CreateSketchTool) Cancel(s *Session) { s.Selection().SetFilter(t.prevFilter) }

// Prompt guides the user to pick a sketch plane (Inventor's status-bar prompt).
func (t *CreateSketchTool) Prompt(*Session) string {
	return "Select a plane or planar face to create the sketch on"
}
