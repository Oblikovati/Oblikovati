// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// FullRoundFilletTool is the interactive Full Round Fillet command (#694): pick a center face and
// the two parallel side faces it sits between, and OK to replace the center with a half-cylinder
// tangent to both sides (the rounded top of a rib/wall). There is no radius — it is derived as half
// the side-to-side distance. Picks land in the ACTIVE set (Side 1 first); arm the others from the
// panel.
type FullRoundFilletTool struct {
	featureEditMode      // set ⇒ this panel re-edits a committed full round (see editFullRoundFilletTool)
	side1, center, side2 []FaceHandle
	seeded1              [][]byte // edit mode: retained keys per set (their faces are consumed, so no live handles exist)
	seededCenter         [][]byte
	seeded2              [][]byte
	activeSet            int // 0 = Side 1, 1 = Center, 2 = Side 2 — which set a pick extends
	added                *feature.PartFeature
}

// NewFullRoundFilletTool returns a full-round tool with Side 1 active.
func NewFullRoundFilletTool() *FullRoundFilletTool { return &FullRoundFilletTool{} }

// Name implements [Tool].
func (t *FullRoundFilletTool) Name() string { return "Full Round Fillet" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *FullRoundFilletTool) Start(*Session) {}

// AcceptedKinds declares full-round-fillet picks faces (the center and side face sets).
func (t *FullRoundFilletTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports every picked face for the unified highlight.
func (t *FullRoundFilletTool) Picks() []Selectable { return faceSelectables(t.Faces()) }

// Pick appends the clicked face to the active set, ignoring a face already in any set.
func (t *FullRoundFilletTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	switch t.activeSet {
	case 1:
		t.center = append(t.center, f)
	case 2:
		t.side2 = append(t.side2, f)
	default:
		t.side1 = append(t.side1, f)
	}
}

// hasFace reports whether the face is already picked into any of the three sets.
func (t *FullRoundFilletTool) hasFace(f FaceHandle) bool {
	return faceSetHas(t.side1, f) || faceSetHas(t.center, f) || faceSetHas(t.side2, f)
}

// ArmSide1 / ArmCenter / ArmSide2 choose which set the next picks extend (the panel's three chips).
func (t *FullRoundFilletTool) ArmSide1()  { t.activeSet = 0 }
func (t *FullRoundFilletTool) ArmCenter() { t.activeSet = 1 }
func (t *FullRoundFilletTool) ArmSide2()  { t.activeSet = 2 }

// ActiveSet reports which set picks currently extend (0 = Side 1, 1 = Center, 2 = Side 2).
func (t *FullRoundFilletTool) ActiveSet() int { return t.activeSet }

// Count1 / CountCenter / Count2 count each set's faces: retained keys (edit mode) plus picks.
func (t *FullRoundFilletTool) Count1() int      { return len(t.seeded1) + len(t.side1) }
func (t *FullRoundFilletTool) CountCenter() int { return len(t.seededCenter) + len(t.center) }
func (t *FullRoundFilletTool) Count2() int      { return len(t.seeded2) + len(t.side2) }

// Faces returns every picked face (all sets) so the viewport highlight (toolPicks) lights them.
func (t *FullRoundFilletTool) Faces() []FaceHandle {
	out := append([]FaceHandle(nil), t.side1...)
	out = append(out, t.center...)
	return append(out, t.side2...)
}

// ClearSide1 / ClearCenter / ClearSide2 empty a set — the panel's selector clear (×).
func (t *FullRoundFilletTool) ClearSide1()  { t.side1, t.seeded1 = nil, nil }
func (t *FullRoundFilletTool) ClearCenter() { t.center, t.seededCenter = nil, nil }
func (t *FullRoundFilletTool) ClearSide2()  { t.side2, t.seeded2 = nil, nil }

// keys1 / keysCenter / keys2 are the reference-key set a commit writes per set: retained keys plus
// this session's picks.
func (t *FullRoundFilletTool) keys1() [][]byte {
	return appendFaceKeys(cloneKeys(t.seeded1), t.side1)
}
func (t *FullRoundFilletTool) keysCenter() [][]byte {
	return appendFaceKeys(cloneKeys(t.seededCenter), t.center)
}
func (t *FullRoundFilletTool) keys2() [][]byte {
	return appendFaceKeys(cloneKeys(t.seeded2), t.side2)
}

// CanCommit reports whether all three sets have at least one face.
func (t *FullRoundFilletTool) CanCommit() bool {
	return t.Count1() > 0 && t.CountCenter() > 0 && t.Count2() > 0
}

// Commit replaces the center face with a full round and recomputes; a sick feature (the sides are
// not parallel, or the center does not sit between them) keeps the tool open via an error.
func (t *FullRoundFilletTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addFullRound(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Full Round Fillet")
	if !t.added.Health().OK() {
		return errors.New("full round fillet: " + t.added.Health().Reason)
	}
	return nil
}

// addFullRound appends the full-round feature — shared by Commit and the preview.
func (t *FullRoundFilletTool) addFullRound(dress *feature.DressUpFeatures) *feature.PartFeature {
	return dress.AddFullRoundFillet(t.keys1(), t.keysCenter(), t.keys2())
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FullRoundFilletTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached full round the viewport previews before commit (the same
// addFullRound the commit uses, so the translucent ghost is exactly what OK creates).
func (t *FullRoundFilletTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFullRound(feature.NewDressUpFeatures(fs)), nil
	})
}

// commitEdit writes the panel state back into the committed full round's definition.
func (t *FullRoundFilletTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.FullRoundFilletFeature).Definition()
	def.Side1Keys, def.CenterKeys, def.Side2Keys = t.keys1(), t.keysCenter(), t.keys2()
	return commitFeatureEdit(s, t.target)
}

// Prompt guides the user through the three picks.
func (t *FullRoundFilletTool) Prompt(*Session) string {
	switch {
	case t.Count1() == 0:
		return "Click the first side face"
	case t.CountCenter() == 0:
		return "Arm Center, then click the face to round"
	case t.Count2() == 0:
		return "Arm Side 2, then click the opposite side face"
	default:
		return "Click OK to round the center face"
	}
}

// Cancel restores the default selection filter, or aborts the edit when re-editing.
func (t *FullRoundFilletTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}
