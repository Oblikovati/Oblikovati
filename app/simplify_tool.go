// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"slices"

	"oblikovati.org/model/feature"
)

// SimplifyTool is the interactive Simplify command: pick the faces to remove — a boss, a fillet
// run, a hole's walls — and/or ask for internal voids to be filled, and OK to reduce the body to
// a lighter one for display, drawings or analysis.
//
// SimplifyDefinition and ModifyFeatures.AddSimplify were implemented and routed over the API,
// but the ribbon's Simplify panel held only Derive and Shrinkwrap — the reduction itself had no
// tool at all (#2050).
type SimplifyTool struct {
	faces     []FaceHandle
	fillVoids bool
	added     *feature.PartFeature
}

// NewSimplifyTool returns a simplify tool with void filling off.
func NewSimplifyTool() *SimplifyTool { return &SimplifyTool{} }

// Name implements [Tool].
func (t *SimplifyTool) Name() string { return "Simplify" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *SimplifyTool) Start(*Session) {}

// AcceptedKinds declares simplify picks the faces to remove.
func (t *SimplifyTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked faces for the unified highlight.
func (t *SimplifyTool) Picks() []Selectable { return selectables(t.faces) }

// Pick appends the clicked face (ignoring a duplicate).
func (t *SimplifyTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	t.faces = append(t.faces, f)
}

func (t *SimplifyTool) hasFace(f FaceHandle) bool {
	return slices.Contains(t.faces, f)
}

// Faces returns the picked faces; FaceCount is what the property panel's chip shows.
func (t *SimplifyTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }
func (t *SimplifyTool) FaceCount() int      { return len(t.faces) }

// FillVoids / SetFillVoids choose whether internal voids are filled.
func (t *SimplifyTool) FillVoids() bool      { return t.fillVoids }
func (t *SimplifyTool) SetFillVoids(on bool) { t.fillVoids = on }

// CanCommit requires something to do: faces to remove, voids to fill, or both. The feature
// itself errors on neither, so the OK button refuses first.
func (t *SimplifyTool) CanCommit() bool { return len(t.faces) > 0 || t.fillVoids }

// Commit reduces the running body and recomputes; a sick feature keeps the tool open.
func (t *SimplifyTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addSimplify(part.Features())
	part.Recompute()
	s.recordEdit(part, "Simplify")
	if !t.added.Health().OK() {
		return errors.New("simplify: " + t.added.Health().Reason)
	}
	return nil
}

// addSimplify builds the simplify feature into engine fs — shared by Commit and the preview.
func (t *SimplifyTool) addSimplify(fs *feature.PartFeatures) *feature.PartFeature {
	return feature.NewModifyFeatures(fs).AddSimplify(faceKeys(t.faces), t.fillVoids)
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SimplifyTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature satisfies DraftPreviewable so the commit gate has a draft to inspect (#1626).
func (t *SimplifyTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addSimplify(fs), nil
	})
}

// Prompt guides the user through the simplify steps.
func (t *SimplifyTool) Prompt(*Session) string {
	if !t.CanCommit() {
		return "Click the faces to remove, or turn on Fill voids"
	}
	return "Click OK to reduce the body"
}

// Cancel is a no-op; the engine restores the ambient filter.
func (t *SimplifyTool) Cancel(*Session) {}

// ClearFaces empties the picked faces — the property panel's selector clear (⊗).
func (t *SimplifyTool) ClearFaces() { t.faces = nil }
