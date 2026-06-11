// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// DeleteFaceTool is the interactive Delete Face command: activate it, click the faces to
// remove, and OK to delete them and heal the openings (the neighbours extend to meet) — e.g.
// deleting a chamfer or fillet face restores the sharp edge.
type DeleteFaceTool struct {
	faces []FaceHandle
	added *feature.PartFeature
}

// NewDeleteFaceTool returns a delete-face tool.
func NewDeleteFaceTool() *DeleteFaceTool { return &DeleteFaceTool{} }

// Name implements [Tool].
func (t *DeleteFaceTool) Name() string { return "Delete Face" }

// Start sets the selection filter to faces.
func (t *DeleteFaceTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick appends the clicked face (ignoring a duplicate).
func (t *DeleteFaceTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	t.faces = append(t.faces, f)
}

func (t *DeleteFaceTool) hasFace(f FaceHandle) bool {
	for _, h := range t.faces {
		if h == f {
			return true
		}
	}
	return false
}

// Faces returns the picked faces (for the UI to list/highlight).
func (t *DeleteFaceTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// CanCommit reports whether at least one face is picked.
func (t *DeleteFaceTool) CanCommit() bool { return len(t.faces) > 0 }

// Commit deletes the picked faces on the active part and heals; a sick feature (the heal did
// not close the body) keeps the tool open by returning an error.
func (t *DeleteFaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddDeleteFace(keys)
	part.Recompute()
	s.recordEdit(part, "Delete Face")
	if !t.added.Health().OK() {
		return errors.New("delete face: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *DeleteFaceTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the delete-face steps.
func (t *DeleteFaceTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to delete and heal"
	}
	return "Click OK to delete and heal"
}

// Cancel restores the default selection filter.
func (t *DeleteFaceTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// ClearFaces empties the picked faces — the property panel's selector clear (⊗) —
// returning the tool to its pick-faces step.
func (t *DeleteFaceTool) ClearFaces() { t.faces = nil }
