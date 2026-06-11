// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// FaceOffsetTool is the interactive Offset Face command: activate it, click one or more
// planar faces, set the offset distance (positive grows the solid along each face's
// outward normal, negative shaves it), and OK to retopologize the active part.
type FaceOffsetTool struct {
	faces    []FaceHandle
	distance float64
	added    *feature.PartFeature
}

// NewFaceOffsetTool returns a face-offset tool with a default 1-unit offset.
func NewFaceOffsetTool() *FaceOffsetTool { return &FaceOffsetTool{distance: 1} }

// Name implements [Tool].
func (t *FaceOffsetTool) Name() string { return "Offset Face" }

// Start sets the selection filter to faces.
func (t *FaceOffsetTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick appends the clicked face (ignoring a duplicate).
func (t *FaceOffsetTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	t.faces = append(t.faces, f)
}

func (t *FaceOffsetTool) hasFace(f FaceHandle) bool {
	for _, h := range t.faces {
		if h == f {
			return true
		}
	}
	return false
}

// SetDistance/Distance set the offset distance (database units, signed).
func (t *FaceOffsetTool) SetDistance(d float64) { t.distance = d }
func (t *FaceOffsetTool) Distance() float64     { return t.distance }

// Faces returns the picked faces (for the UI to list/highlight).
func (t *FaceOffsetTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// CanCommit reports whether at least one face is picked and the distance is non-zero.
func (t *FaceOffsetTool) CanCommit() bool { return len(t.faces) > 0 && t.distance != 0 }

// Commit offsets the picked faces on the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *FaceOffsetTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddFaceOffset(keys, t.distance)
	part.Recompute()
	s.recordEdit(part, "Offset Face")
	if !t.added.Health().OK() {
		return errors.New("offset face: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FaceOffsetTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the offset steps.
func (t *FaceOffsetTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click one or more faces to offset"
	}
	return "Set the distance, then click OK"
}

// Cancel restores the default selection filter.
func (t *FaceOffsetTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// ClearFaces empties the picked faces — the property panel's selector clear (⊗) —
// returning the tool to its pick-faces step.
func (t *FaceOffsetTool) ClearFaces() { t.faces = nil }
