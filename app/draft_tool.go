// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/model/feature"
)

// degToRad converts the draft angle the UI takes in degrees to the radians the feature uses.
const degToRad = stdmath.Pi / 180

// DraftTool is the interactive Draft command: activate it, click one or more faces, set the
// draft angle (degrees) in the property window, and OK to taper them about the +Z pull
// direction (the mould-pull default). Negative angle leans the face in, positive out.
type DraftTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed draft (see editDraftTool)
	faces           []FaceHandle
	seededFaceKeys  [][]byte // edit mode: the feature's existing face keys
	angleDeg        float64
	added           *feature.PartFeature
}

// NewDraftTool returns a draft tool with a default 3° angle.
func NewDraftTool() *DraftTool { return &DraftTool{angleDeg: 3} }

// Name implements [Tool].
func (t *DraftTool) Name() string { return "Draft" }

// Start sets the selection filter to faces.
func (t *DraftTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick appends the clicked face (ignoring a duplicate).
func (t *DraftTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	t.faces = append(t.faces, f)
}

func (t *DraftTool) hasFace(f FaceHandle) bool {
	for _, h := range t.faces {
		if h == f {
			return true
		}
	}
	return false
}

// SetAngleDegrees/AngleDegrees set the draft angle in degrees (signed).
func (t *DraftTool) SetAngleDegrees(a float64) { t.angleDeg = a }
func (t *DraftTool) AngleDegrees() float64     { return t.angleDeg }

// Faces returns the picked faces (for the UI to list/highlight).
func (t *DraftTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// FaceCount counts the selection the panel shows: faces picked this session plus, in
// edit mode, the feature's retained faces.
func (t *DraftTool) FaceCount() int { return len(t.seededFaceKeys) + len(t.faces) }

// selectedFaceKeys is the reference-key set a commit writes: the retained keys plus
// this session's picks.
func (t *DraftTool) selectedFaceKeys() [][]byte {
	keys := cloneKeys(t.seededFaceKeys)
	for _, f := range t.faces {
		keys = append(keys, f.Face.ReferenceKey())
	}
	return keys
}

// CanCommit reports whether at least one face is selected and the angle is non-zero.
func (t *DraftTool) CanCommit() bool { return t.FaceCount() > 0 && t.angleDeg != 0 }

// Commit tapers the picked faces on the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *DraftTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	rad := t.angleDeg * degToRad
	t.added = feature.NewDressUpFeatures(part.Features()).AddDraft(t.selectedFaceKeys(), func() float64 { return rad })
	part.Recompute()
	s.recordEdit(part, "Draft")
	if !t.added.Health().OK() {
		return errors.New("draft: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *DraftTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the draft steps.
func (t *DraftTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click one or more faces to draft"
	}
	return "Set the angle, then click OK"
}

// Cancel restores the default selection filter.
func (t *DraftTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
	s.Selection().SetFilter(NewSelectionFilter())
}

// commitEdit writes the panel state back into the committed draft's definition.
func (t *DraftTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.FaceDraftFeature).Definition()
	def.FaceKeys = t.selectedFaceKeys()
	def.Angle = konst(t.angleDeg * degToRad)
	return commitFeatureEdit(s, t.target)
}

// ClearFaces empties the face selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-faces step.
func (t *DraftTool) ClearFaces() {
	t.faces = nil
	t.seededFaceKeys = nil
}
