// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"slices"

	"oblikovati.org/model/feature"
)

// FaceFilletTool is the interactive Face Fillet command (#694): pick two sets of faces, set the
// blend radius, and OK to round every edge shared between the two sets. Selecting by face — rather
// than edge — is the natural choice when a long chain of edges separates two regions. Picks land in
// the ACTIVE set (Face Set 1 by default); arm the other set from the panel to extend it.
type FaceFilletTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed face fillet (see editFaceFilletTool)
	setA, setB      []FaceHandle
	seededKeysA     [][]byte // edit mode: the retained face keys per set (their faces are consumed, so no live handles exist)
	seededKeysB     [][]byte
	activeSet       int // 0 = set A, 1 = set B — which set a pick extends
	radius          float64
	added           *feature.PartFeature
}

// NewFaceFilletTool returns a face-fillet tool with a default 1-unit radius and Face Set 1 active.
func NewFaceFilletTool() *FaceFilletTool { return &FaceFilletTool{radius: 1} }

// Name implements [Tool].
func (t *FaceFilletTool) Name() string { return "Face Fillet" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *FaceFilletTool) Start(*Session) {}

// AcceptedKinds declares face-fillet picks faces (the two face sets to blend between).
func (t *FaceFilletTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports every picked face for the unified highlight.
func (t *FaceFilletTool) Picks() []Selectable { return selectables(t.Faces()) }

// Pick appends the clicked face to the active set, ignoring a face already in either set.
func (t *FaceFilletTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	if t.activeSet == 1 {
		t.setB = append(t.setB, f)
		return
	}
	t.setA = append(t.setA, f)
}

// hasFace reports whether the face is already picked into either set.
func (t *FaceFilletTool) hasFace(f FaceHandle) bool {
	return faceSetHas(t.setA, f) || faceSetHas(t.setB, f)
}

// faceSetHas reports whether set already contains f.
func faceSetHas(set []FaceHandle, f FaceHandle) bool {
	return slices.Contains(set, f)
}

// ArmSetA / ArmSetB choose which set the next picks extend (the panel's two selector chips).
func (t *FaceFilletTool) ArmSetA() { t.activeSet = 0 }
func (t *FaceFilletTool) ArmSetB() { t.activeSet = 1 }

// ActiveSet reports which face set picks currently extend (0 = set A, 1 = set B).
func (t *FaceFilletTool) ActiveSet() int { return t.activeSet }

// CountA / CountB count each set's faces: the retained keys (edit mode) plus this session's picks.
func (t *FaceFilletTool) CountA() int { return len(t.seededKeysA) + len(t.setA) }
func (t *FaceFilletTool) CountB() int { return len(t.seededKeysB) + len(t.setB) }

// Faces returns every picked face (both sets) so the viewport highlight (toolPicks) lights them.
func (t *FaceFilletTool) Faces() []FaceHandle {
	return append(append([]FaceHandle(nil), t.setA...), t.setB...)
}

// SetRadius / Radius set the blend radius (database units).
func (t *FaceFilletTool) SetRadius(r float64) { t.radius = r }
func (t *FaceFilletTool) Radius() float64     { return t.radius }

// ClearSetA / ClearSetB empty a set — the panel's selector clear (×) — including any retained keys.
func (t *FaceFilletTool) ClearSetA() { t.setA, t.seededKeysA = nil, nil }
func (t *FaceFilletTool) ClearSetB() { t.setB, t.seededKeysB = nil, nil }

// keysA / keysB are the reference-key set a commit writes per set: retained keys plus this
// session's picks.
func (t *FaceFilletTool) keysA() [][]byte { return appendFaceKeys(cloneKeys(t.seededKeysA), t.setA) }
func (t *FaceFilletTool) keysB() [][]byte { return appendFaceKeys(cloneKeys(t.seededKeysB), t.setB) }

// appendFaceKeys appends the reference keys of picked faces onto keys.
func appendFaceKeys(keys [][]byte, faces []FaceHandle) [][]byte {
	for _, f := range faces {
		keys = append(keys, f.Face.ReferenceKey())
	}
	return keys
}

// CanCommit reports whether both sets have at least one face and the radius is positive.
func (t *FaceFilletTool) CanCommit() bool {
	return t.CountA() > 0 && t.CountB() > 0 && t.radius > 0
}

// Commit rounds the edges shared between the two face sets and recomputes; a sick feature (the
// sets share no edge, or a radius the geometry can't take) keeps the tool open via an error.
func (t *FaceFilletTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addFaceFillet(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Face Fillet")
	if !t.added.Health().OK() {
		return errors.New("face fillet: " + t.added.Health().Reason)
	}
	return nil
}

// addFaceFillet appends the face fillet between the two sets — shared by Commit and the preview.
func (t *FaceFilletTool) addFaceFillet(dress *feature.DressUpFeatures) *feature.PartFeature {
	r := t.radius
	return dress.AddFaceFillet(t.keysA(), t.keysB(), func() float64 { return r })
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FaceFilletTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached face fillet the viewport previews before commit (the same
// addFaceFillet the commit uses, so the translucent ghost is exactly what OK creates).
func (t *FaceFilletTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFaceFillet(feature.NewDressUpFeatures(fs)), nil
	})
}

// commitEdit writes the panel state back into the committed face fillet's definition.
func (t *FaceFilletTool) commitEdit(s *Session) error {
	r := t.radius
	def := t.target.Definition().(*feature.FaceFilletFeature).Definition()
	def.FaceKeysA, def.FaceKeysB, def.Radius = t.keysA(), t.keysB(), func() float64 { return r }
	return commitFeatureEdit(s, t.target)
}

// Prompt guides the user through the face-fillet steps.
func (t *FaceFilletTool) Prompt(*Session) string {
	switch {
	case t.CountA() == 0:
		return "Click the first set of faces"
	case t.CountB() == 0:
		return "Arm Face Set 2, then click the second set of faces"
	default:
		return "Set the radius, then click OK"
	}
}

// Cancel restores the default selection filter, or aborts the edit when re-editing.
func (t *FaceFilletTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}
