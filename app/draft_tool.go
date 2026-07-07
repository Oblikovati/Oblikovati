// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// degToRad converts the draft angle the UI takes in degrees to the radians the feature uses.
const degToRad = stdmath.Pi / 180

// draftPickMode selects which slot the next viewport click fills: the faces to taper, the pull-
// direction face (its normal), or the neutral (parting) plane face. Mirrors ReplaceFaceTool's mode.
type draftPickMode int

const (
	pickDraftFaces draftPickMode = iota
	pickDraftPull
	pickDraftNeutral
)

// DraftTool is the interactive Draft command: activate it, click one or more faces, optionally set a
// pull-direction face and a neutral (parting) plane face, set the draft angle (degrees), and OK to
// taper. Pull defaults to +Z (the mould-pull default); with no neutral plane each face pivots on the
// implicit lowest-vertex hinge. Negative angle leans the face in, positive out (#1801).
type DraftTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed draft (see editDraftTool)
	faces           []FaceHandle
	seededFaceKeys  [][]byte // edit mode: the feature's existing face keys
	angleDeg        float64
	mode            draftPickMode
	pull            *FaceHandle   // pull-direction face (its normal); nil ⇒ default/seeded
	neutral         *FaceHandle   // neutral parting-plane face; nil ⇒ default/seeded
	seededPull      *math.Vector3 // edit mode: the committed pull direction (kept if not re-picked)
	seededNeutral   *geom.Plane   // edit mode: the committed neutral plane (kept if not re-picked)
	added           *feature.PartFeature
}

// NewDraftTool returns a draft tool with a default 3° angle.
func NewDraftTool() *DraftTool { return &DraftTool{angleDeg: 3} }

// Name implements [Tool].
func (t *DraftTool) Name() string { return "Draft" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *DraftTool) Start(*Session) {}

// AcceptedKinds declares draft picks faces (the faces to taper).
func (t *DraftTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Pick routes the clicked face to the active slot: the pull-direction face, the neutral-plane face,
// or (default) the faces to taper.
func (t *DraftTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok {
		return
	}
	switch t.mode {
	case pickDraftPull:
		fc := f
		t.pull = &fc
	case pickDraftNeutral:
		fc := f
		t.neutral = &fc
	default:
		if !t.hasFace(f) {
			t.faces = append(t.faces, f)
		}
	}
}

// Picks reports the tapered faces plus the pull and neutral faces for the unified highlight.
func (t *DraftTool) Picks() []Selectable {
	return appendPick(appendPick(selectables(t.faces), t.pull), t.neutral)
}

// SetPickMode / PickMode switch which slot the next viewport click fills.
func (t *DraftTool) SetPickMode(m draftPickMode) { t.mode = m }
func (t *DraftTool) PickMode() draftPickMode     { return t.mode }

// PickingPull / PickingNeutral report the active pick slot (for the dialog's row highlight).
func (t *DraftTool) PickingPull() bool    { return t.mode == pickDraftPull }
func (t *DraftTool) PickingNeutral() bool { return t.mode == pickDraftNeutral }

// SetPickingPull / SetPickingNeutral route the next viewport click to the pull-direction or
// neutral-plane slot when enabled, or back to picking faces when disabled — the exported toggles
// the head's dialog drives (mirrors ReplaceFaceTool.SetPickingTarget). Enabling one clears the
// other so only one slot is armed at a time.
func (t *DraftTool) SetPickingPull(on bool) {
	if on {
		t.mode = pickDraftPull
		return
	}
	t.mode = pickDraftFaces
}
func (t *DraftTool) SetPickingNeutral(on bool) {
	if on {
		t.mode = pickDraftNeutral
		return
	}
	t.mode = pickDraftFaces
}

// PullSet / NeutralSet report whether a pull face / neutral plane is in effect (picked or seeded).
func (t *DraftTool) PullSet() bool    { return t.pull != nil || t.seededPull != nil }
func (t *DraftTool) NeutralSet() bool { return t.neutral != nil || t.seededNeutral != nil }

// ClearPull / ClearNeutral drop the pull / neutral input, reverting to the default (+Z / implicit
// hinge), and return to picking faces.
func (t *DraftTool) ClearPull()    { t.pull, t.seededPull, t.mode = nil, nil, pickDraftFaces }
func (t *DraftTool) ClearNeutral() { t.neutral, t.seededNeutral, t.mode = nil, nil, pickDraftFaces }

// pullVector resolves the pull direction: the picked face's normal, else the seeded (edit) direction,
// else the +Z mould-pull default.
func (t *DraftTool) pullVector() math.Vector3 {
	if t.pull != nil {
		if pl, ok := t.pull.Face.Geometry().(geom.Plane); ok {
			return pl.Normal()
		}
	}
	if t.seededPull != nil {
		return *t.seededPull
	}
	return math.V3(0, 0, 1)
}

// neutralPlane resolves the neutral parting plane: the picked planar face, else the seeded (edit)
// plane, else nil (the implicit lowest-vertex hinge).
func (t *DraftTool) neutralPlane() *geom.Plane {
	if t.neutral != nil {
		if pl, ok := t.neutral.Face.Geometry().(geom.Plane); ok {
			return &pl
		}
	}
	return t.seededNeutral
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
	t.added = t.addDraft(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Draft")
	if !t.added.Health().OK() {
		return errors.New("draft: " + t.added.Health().Reason)
	}
	return nil
}

// addDraft builds the draft feature into collection dress — the shared constructor used by
// both Commit (the part's engine) and DraftFeature (a scratch engine).
func (t *DraftTool) addDraft(dress *feature.DressUpFeatures) *feature.PartFeature {
	rad := t.angleDeg * degToRad
	return dress.AddDraftPullNeutral(t.selectedFaceKeys(), t.pullVector(), t.neutralPlane(), func() float64 { return rad })
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *DraftTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached draft feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addDraft the commit uses. Empty until a
// face is selected.
func (t *DraftTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addDraft(feature.NewDressUpFeatures(fs)), nil
	})
}

// Prompt guides the user through the draft steps.
func (t *DraftTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click one or more faces to draft"
	}
	return "Set the angle, then click OK"
}

// Cancel abandons the tool; the engine restores the ambient filter.
func (t *DraftTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}

// commitEdit writes the panel state back into the committed draft's definition.
func (t *DraftTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.FaceDraftFeature).Definition()
	def.FaceKeys = t.selectedFaceKeys()
	def.Angle = konst(t.angleDeg * degToRad)
	def.PullDir = t.pullVector()
	def.Neutral = t.neutralPlane()
	return commitFeatureEdit(s, t.target)
}

// ClearFaces empties the face selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-faces step.
func (t *DraftTool) ClearFaces() {
	t.faces = nil
	t.seededFaceKeys = nil
}
