// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/feature"
)

// ReplaceFaceTool is the interactive Replace Face command: click the faces to replace, then
// switch to target mode and click the face — or the WORK PLANE — whose plane they should take,
// and OK. The picked faces move onto the target's plane and the neighbours retrim.
//
// AddReplaceFacePlanes (targets frozen as planes rather than as a face reference, #1886) was
// implemented and routed over the API but the tool only ever called AddReplaceFace, so replacing
// onto a work plane was API-only (#2050).
type ReplaceFaceTool struct {
	faces         []FaceHandle
	target        *FaceHandle
	targetPlane   *feature.WorkPlane
	pickingTarget bool
	added         *feature.PartFeature
}

// NewReplaceFaceTool returns a replace-face tool (starting in replace-face selection).
func NewReplaceFaceTool() *ReplaceFaceTool { return &ReplaceFaceTool{} }

// Name implements [Tool].
func (t *ReplaceFaceTool) Name() string { return "Replace Face" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ReplaceFaceTool) Start(*Session) {}

// AcceptedKinds declares replace-face picks faces (the faces to replace) and, for the target,
// a face or a work plane.
func (t *ReplaceFaceTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectFace, SelectWorkPlane}
}

// Picks reports the faces to replace plus the target face for the unified highlight.
func (t *ReplaceFaceTool) Picks() []Selectable {
	return appendPick(selectables(t.faces), t.target)
}

// Pick routes a click to the target slot when in target mode, else appends a face to replace.
func (t *ReplaceFaceTool) Pick(_ *Session, sel Selectable) {
	if wp, ok := sel.(WorkPlaneHandle); ok {
		if t.pickingTarget { // a work plane is only ever a target, never a face to replace
			t.targetPlane, t.target = wp.Plane, nil
		}
		return
	}
	f, ok := sel.(FaceHandle)
	if !ok {
		return
	}
	if t.pickingTarget {
		fc := f
		t.target, t.targetPlane = &fc, nil
		return
	}
	if !t.hasFace(f) {
		t.faces = append(t.faces, f)
	}
}

func (t *ReplaceFaceTool) hasFace(f FaceHandle) bool {
	for _, h := range t.faces {
		if h == f {
			return true
		}
	}
	return false
}

// SetPickingTarget switches between picking the faces to replace and the target face.
func (t *ReplaceFaceTool) SetPickingTarget(b bool) { t.pickingTarget = b }

// PickingTarget reports whether the next click sets the target face.
func (t *ReplaceFaceTool) PickingTarget() bool { return t.pickingTarget }

// Faces returns the faces to replace (for the UI to list/highlight).
func (t *ReplaceFaceTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// PickedTarget reports the target face if one has been chosen.
func (t *ReplaceFaceTool) PickedTarget() (FaceHandle, bool) {
	if t.target == nil {
		return FaceHandle{}, false
	}
	return *t.target, true
}

// TargetPlane reports the target work plane if one has been chosen instead of a face.
func (t *ReplaceFaceTool) TargetPlane() (*feature.WorkPlane, bool) {
	return t.targetPlane, t.targetPlane != nil
}

// TargetPicked reports whether a target — a face or a work plane — has been chosen.
func (t *ReplaceFaceTool) TargetPicked() bool { return t.target != nil || t.targetPlane != nil }

// CanCommit reports whether at least one face and a target are picked.
func (t *ReplaceFaceTool) CanCommit() bool { return len(t.faces) > 0 && t.TargetPicked() }

// Commit replaces the picked faces with the target's plane on the active part and
// recomputes; a sick feature keeps the tool open by returning an error.
func (t *ReplaceFaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addReplaceFace(part.Features())
	part.Recompute()
	s.recordEdit(part, "Replace Face")
	if !t.added.Health().OK() {
		return errors.New("replace face: " + t.added.Health().Reason)
	}
	return nil
}

// addReplaceFace builds the replace-face feature into engine fs — shared by Commit and preview.
func (t *ReplaceFaceTool) addReplaceFace(fs *feature.PartFeatures) *feature.PartFeature {
	mods := feature.NewModifyFeatures(fs)
	if pl, ok := t.targetGeomPlane(); ok {
		// A work plane's geometry is frozen into the definition: unlike a face it carries no
		// lineage key the recompute could rebind, which is what AddReplaceFacePlanes is for.
		return mods.AddReplaceFacePlanes(faceKeys(t.faces), []geom.Plane{pl})
	}
	return mods.AddReplaceFace(faceKeys(t.faces), t.target.Face.ReferenceKey())
}

// targetGeomPlane converts the picked work plane to the kernel plane the definition freezes.
// A degenerate plane reports false, so the commit falls back to the face path and errors there
// rather than freezing a garbage target.
func (t *ReplaceFaceTool) targetGeomPlane() (geom.Plane, bool) {
	if t.targetPlane == nil {
		return geom.Plane{}, false
	}
	p := t.targetPlane.Plane()
	pl, err := geom.NewPlane(p.Origin(), p.Normal().AsVector())
	return pl, err == nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ReplaceFaceTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached replace-face feature the viewport previews before commit.
func (t *ReplaceFaceTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addReplaceFace(fs), nil
	})
}

// Prompt guides the user through the replace-face steps.
func (t *ReplaceFaceTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to replace"
	}
	if t.target == nil {
		return "Switch to target, then click the face whose plane to use"
	}
	return "Click OK to replace"
}

// Cancel restores the default selection filter.
func (t *ReplaceFaceTool) Cancel(*Session) {}

// ClearFaces / ClearTarget empty one pick set each — the property panel's selector
// clear (⊗) affordances on the replace-faces and target chips.
func (t *ReplaceFaceTool) ClearFaces() { t.faces = nil }

// ClearTarget drops the picked target face.
func (t *ReplaceFaceTool) ClearTarget() { t.target = nil }
