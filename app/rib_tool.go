// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// RibTool is the interactive Rib command: with an open sketch profile (the active sketch, or
// the part's most recent sketch holding an open path), set the wall thickness and depth in the
// property window and OK to thicken the profile into a rib joined to the active part. It exposes
// its parameters through ParameterizedTool, so the head renders the generic dialog.
type RibTool struct {
	profile   *sketch.Sketch
	pathIndex int
	thickness float64
	depth     float64
	added     *feature.PartFeature
}

// NewRibTool returns a rib tool with a default 1×2 wall.
func NewRibTool() *RibTool { return &RibTool{thickness: 1, depth: 2} }

// Name implements [Tool].
func (t *RibTool) Name() string { return "Rib" }

// Start resolves the open profile to rib: the active sketch if one is open, otherwise the
// part's most recent sketch that has an open path.
func (t *RibTool) Start(s *Session) { t.profile, t.pathIndex = ribProfile(s) }

// Pick is unused — the rib operates on the resolved open profile, not a viewport selection.
func (t *RibTool) Pick(*Session, Selectable) {}

// The options the property window drives: thickness and signed depth (database units).
func (t *RibTool) SetThickness(v float64) { t.thickness = v }
func (t *RibTool) Thickness() float64     { return t.thickness }
func (t *RibTool) SetDepth(v float64)     { t.depth = v }
func (t *RibTool) Depth() float64         { return t.depth }

// Params exposes the rib's thickness and depth for the generic property dialog.
func (t *RibTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{Label: "Thickness", Get: t.Thickness, Set: t.SetThickness},
		{Label: "Depth", Get: t.Depth, Set: t.SetDepth},
	}}
}

// CanCommit reports whether an open profile was resolved and the thickness/depth are usable.
func (t *RibTool) CanCommit() bool { return t.profile != nil && t.thickness > 0 && t.depth != 0 }

// Commit thickens the profile into a rib joined to the active part and recomputes; a sick
// feature keeps the tool open by returning an error.
func (t *RibTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addRib(part.Features())
	part.Recompute()
	s.recordEdit(part, "Rib")
	if !t.added.Health().OK() {
		return errors.New("rib: " + t.added.Health().Reason)
	}
	return nil
}

// addRib builds the rib feature into engine fs — shared by Commit and the preview.
func (t *RibTool) addRib(fs *feature.PartFeatures) *feature.PartFeature {
	th, d := t.thickness, t.depth
	return feature.NewRibFeatures(fs).Add(t.profile, t.pathIndex, konst(th), konst(d), ops.Join)
}

// Cancel abandons the tool with no change.
func (t *RibTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RibTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached rib feature the viewport previews before commit.
func (t *RibTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addRib(fs), nil
	})
}

// ribProfile picks the sketch + open-path index the rib operates on: the active sketch if it
// has an open path, else the part's most recent sketch that does. Returns nil when none.
func ribProfile(s *Session) (*sketch.Sketch, int) {
	if sk := s.ActiveSketch(); sk != nil {
		if i, ok := firstOpenPath(sk); ok {
			return sk, i
		}
	}
	part, err := activePart(s)
	if err != nil {
		return nil, 0
	}
	sks := part.Sketches()
	for i := sks.Count() - 1; i >= 0; i-- {
		if j, ok := firstOpenPath(sks.Item(i)); ok {
			return sks.Item(i), j
		}
	}
	return nil, 0
}

// firstOpenPath returns the index of the sketch's first open (non-closed) path.
func firstOpenPath(sk *sketch.Sketch) (int, bool) {
	for i, p := range sk.Paths() {
		if !p.IsClosed() {
			return i, true
		}
	}
	return 0, false
}
