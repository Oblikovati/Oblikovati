// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/model/feature"
)

// HoleTool is the interactive Hole command: activate it, click a planar face, set the
// diameter and depth in the property window, and OK to drill a hole (a cylinder cut at
// the face centroid along the inward normal) into the active part. Diameter and depth are
// in database units; the property window converts to the document's display unit.
type HoleTool struct {
	face     *FaceHandle
	diameter float64
	depth    float64
	added    *feature.PartFeature
}

// NewHoleTool returns a hole tool with a default Ø1 × 2 drilled hole.
func NewHoleTool() *HoleTool { return &HoleTool{diameter: 1, depth: 2} }

// Name implements [Tool].
func (t *HoleTool) Name() string { return "Hole" }

// Start sets the selection filter to faces so clicks pick a placement face.
func (t *HoleTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick captures the planar face the user clicked.
func (t *HoleTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		fc := f
		t.face = &fc
	}
}

// The options the property window drives: diameter and depth (database units).
func (t *HoleTool) SetDiameter(d float64) { t.diameter = d }
func (t *HoleTool) Diameter() float64     { return t.diameter }
func (t *HoleTool) SetDepth(d float64)    { t.depth = d }
func (t *HoleTool) Depth() float64        { return t.depth }

// PickedFace returns the placement face (and true), or false when none picked yet.
func (t *HoleTool) PickedFace() (FaceHandle, bool) {
	if t.face == nil {
		return FaceHandle{}, false
	}
	return *t.face, true
}

// CanCommit reports whether a face is picked and the diameter/depth are positive.
func (t *HoleTool) CanCommit() bool { return t.face != nil && t.diameter > 0 && t.depth > 0 }

// Commit drills the hole into the active part and recomputes; a sick feature (lost face,
// boolean failure) keeps the tool open by returning an error.
func (t *HoleTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	key := t.face.Face.ReferenceKey()
	d, depth := t.diameter, t.depth
	t.added = feature.NewHoleFeatures(part.Features()).
		AddDrilled(key, func() float64 { return d }, func() float64 { return depth })
	part.Recompute()
	if !t.added.Health().OK() {
		return errors.New("hole: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *HoleTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the hole steps.
func (t *HoleTool) Prompt(*Session) string {
	if t.face == nil {
		return "Select a planar face to place the hole on"
	}
	return "Set the diameter and depth, then click OK"
}

// Cancel restores the default selection filter.
func (t *HoleTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
