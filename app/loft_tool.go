// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/feature"
	"oblikovati/renderer"
)

// LoftTool is the interactive Loft command: activate it, click two or more cross-sections in
// order — each a sketch region, or a vertex/work point to taper the loft to an apex (a cone or
// dome) — optionally close the loop, set the end conditions, and OK to blend them into a solid.
type LoftTool struct {
	sections    []loftPick
	closed      bool
	operation   ops.PartFeatureOperation
	first, last feature.LoftEnd // end-section conditions (zero value = Free); see SetFirstCondition
	added       *feature.PartFeature
}

// loftPick is one picked cross-section: a profile region, a point (apex), or a body face (by
// reference key) the loft can leave tangent.
type loftPick struct {
	profile ProfileHandle
	apex    *math.Point3
	faceKey []byte
}

func (p loftPick) isPoint() bool { return p.apex != nil }
func (p loftPick) isFace() bool  { return len(p.faceKey) > 0 }

// NewLoftTool returns a loft tool that creates a new body.
func NewLoftTool() *LoftTool { return &LoftTool{operation: ops.NewBody} }

// Name implements [Tool].
func (t *LoftTool) Name() string { return "Loft" }

// Start lets clicks pick sketch regions (profiles), points (vertices / work points) for an apex,
// or a body face the loft can leave tangent.
func (t *LoftTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile, SelectVertex, SelectWorkPoint, SelectFace))
}

// Pick appends the clicked profile, point, or face as the next cross-section (ignoring a profile
// already in the list, so a double-click does not duplicate a section).
func (t *LoftTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		if t.hasProfile(h) {
			return
		}
		t.sections = append(t.sections, loftPick{profile: h})
	case VertexHandle:
		p := h.Vertex.Point()
		t.sections = append(t.sections, loftPick{apex: &p})
	case WorkPointHandle:
		p := h.Point.Point()
		t.sections = append(t.sections, loftPick{apex: &p})
	case FaceHandle:
		t.sections = append(t.sections, loftPick{faceKey: h.Face.ReferenceKey()})
	}
}

// hasProfile reports whether profile h is already a picked section.
func (t *LoftTool) hasProfile(h ProfileHandle) bool {
	for _, s := range t.sections {
		if !s.isPoint() && s.profile == h {
			return true
		}
	}
	return false
}

// SetClosed toggles a closed loft (the last section blends back to the first).
func (t *LoftTool) SetClosed(on bool) { t.closed = on }
func (t *LoftTool) Closed() bool      { return t.closed }

// SetOperation/Operation choose the boolean applied against existing material.
func (t *LoftTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *LoftTool) Operation() ops.PartFeatureOperation      { return t.operation }

// SetFirstCondition/SetLastCondition set the start/end-section conditions (how the surface
// leaves each end). An Angle/Direction condition curves a two-section loft; the zero value
// is Free (ruled). Ignored when the loft is closed (no end sections).
func (t *LoftTool) SetFirstCondition(e feature.LoftEnd) { t.first = e }
func (t *LoftTool) SetLastCondition(e feature.LoftEnd)  { t.last = e }
func (t *LoftTool) FirstCondition() feature.LoftEnd     { return t.first }
func (t *LoftTool) LastCondition() feature.LoftEnd      { return t.last }

// SectionCount returns how many cross-sections (profiles + points) have been picked.
func (t *LoftTool) SectionCount() int { return len(t.sections) }

// PickedProfiles is the unified-tool-highlight accessor: the head outlines every picked PROFILE
// cross-section (point sections have nothing to outline).
func (t *LoftTool) PickedProfiles() []ProfileHandle {
	var out []ProfileHandle
	for _, s := range t.sections {
		if !s.isPoint() && !s.isFace() {
			out = append(out, s.profile)
		}
	}
	return out
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
		switch {
		case h.isPoint():
			sections[i] = feature.LoftSection{Point: h.apex}
		case h.isFace():
			sections[i] = feature.LoftSection{FaceKey: h.faceKey}
		default:
			sections[i] = feature.LoftSection{Sketch: h.profile.Sketch, ProfileIndex: h.profile.ProfileIndex}
		}
	}
	t.added = feature.NewLoftFeatures(part.Features()).AddConditioned(sections, t.closed, t.operation, t.first, t.last)
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
		return "Click two or more cross-sections to loft (regions, or a vertex/point for an apex)"
	}
	return "Add more sections or click OK"
}

// Preview outlines each picked profile cross-section so the user sees what will be blended (a
// point section is a single vertex, already drawn by the selection highlight).
func (t *LoftTool) Preview(*Session) []renderer.DrawItem {
	var items []renderer.DrawItem
	for _, s := range t.sections {
		if s.isPoint() || s.isFace() {
			continue // a point/face section has no profile polygon to outline
		}
		h := s.profile
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
