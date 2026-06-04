// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/renderer"
)

// LoftTool is the interactive Loft command: activate it, click two or more sketch
// regions in order (each on its own plane) to be the cross-sections, optionally close
// the loop, and OK to blend them into a solid. Each picked region maps directly to a
// [feature.LoftSection], so no extra plumbing is needed beyond the profile picks.
type LoftTool struct {
	sections  []ProfileHandle
	closed    bool
	operation ops.PartFeatureOperation
	added     *feature.PartFeature
}

// NewLoftTool returns a loft tool that creates a new body.
func NewLoftTool() *LoftTool { return &LoftTool{operation: ops.NewBody} }

// Name implements [Tool].
func (t *LoftTool) Name() string { return "Loft" }

// Start sets the selection filter to profiles so clicks pick regions.
func (t *LoftTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick appends the clicked region as the next cross-section (ignoring a region already
// in the list, so a double-click does not duplicate a section).
func (t *LoftTool) Pick(_ *Session, sel Selectable) {
	p, ok := sel.(ProfileHandle)
	if !ok || indexOfProfile(t.sections, p) >= 0 {
		return
	}
	t.sections = append(t.sections, p)
}

// SetClosed toggles a closed loft (the last section blends back to the first).
func (t *LoftTool) SetClosed(on bool) { t.closed = on }
func (t *LoftTool) Closed() bool      { return t.closed }

// SetOperation/Operation choose the boolean applied against existing material.
func (t *LoftTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *LoftTool) Operation() ops.PartFeatureOperation      { return t.operation }

// Sections returns the picked cross-sections in order (for the UI to highlight/list).
func (t *LoftTool) Sections() []ProfileHandle {
	return append([]ProfileHandle(nil), t.sections...)
}

// CanCommit reports whether at least two cross-sections have been picked.
func (t *LoftTool) CanCommit() bool { return len(t.sections) >= 2 }

// Commit adds the loft feature to the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *LoftTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	sections := make([]feature.LoftSection, len(t.sections))
	for i, h := range t.sections {
		sections[i] = feature.LoftSection{Sketch: h.Sketch, ProfileIndex: h.ProfileIndex}
	}
	t.added = feature.NewLoftFeatures(part.Features()).Add(sections, t.closed, t.operation)
	part.Recompute()
	s.recordEdit(part, "Loft")
	if !t.added.Health().OK() {
		return errors.New("loft: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *LoftTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the loft steps.
func (t *LoftTool) Prompt(*Session) string {
	if len(t.sections) < 2 {
		return "Click two or more cross-section regions to loft (in order)"
	}
	return "Add more sections or click OK"
}

// Preview outlines each picked cross-section so the user sees what will be blended.
func (t *LoftTool) Preview(*Session) []renderer.DrawItem {
	var items []renderer.DrawItem
	for _, h := range t.sections {
		if h.ProfileIndex >= h.Sketch.Profiles().Count() {
			continue
		}
		poly := h.Sketch.Profiles().Item(h.ProfileIndex).OuterLoop().Polygon()
		plane := h.Sketch.Plane()
		pts := make([]math.Point3, len(poly))
		idx := make([]int, 0, 2*len(poly))
		for i, p := range poly {
			pts[i] = plane.ToModel(p)
			idx = append(idx, i, (i+1)%len(poly))
		}
		items = append(items, renderer.DrawItem{Primitive: renderer.Lines, Positions: pts, Indices: idx, Color: [4]float32{1, 0.6, 0, 1}})
	}
	return items
}

// Cancel restores the default selection filter.
func (t *LoftTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
