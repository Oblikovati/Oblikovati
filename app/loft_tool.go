// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// LoftTool is the interactive Loft command: activate it, click two or more cross-sections in
// order — each a sketch region, or a vertex/work point to taper the loft to an apex (a cone or
// dome) — optionally close the loop, set the end conditions, and OK to blend them into a solid.
type LoftTool struct {
	sections     []loftPick
	rails        []PathHandle // guide curves (kLoftWithRails) — picked sketch paths
	centerline   *PathHandle  // spine curve (kLoftWithCenterline)
	mapCurves    []PathHandle // explicit correspondence anchors (MapPointCurves)
	guideKind    int          // where a path pick goes: 0 rail, 1 centerline, 2 map curve
	areaMidScale float64      // area-graph: cross-section area scale at mid height (0/1 = off)
	closed       bool
	operation    ops.PartFeatureOperation
	first, last  feature.LoftEnd // end-section conditions (zero value = Free); see SetFirstCondition
	added        *feature.PartFeature
}

// Loft guide-path kinds: where a picked open path is routed.
const (
	loftGuideRail = iota
	loftGuideCenterline
	loftGuideMapCurve
)

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
// a body face the loft can leave tangent, or an open sketch path as a guide rail.
func (t *LoftTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile, SelectVertex, SelectWorkPoint, SelectFace, SelectPath))
}

// Pick routes the clicked entity: a profile/point/face is the next cross-section (a profile
// already in the list is ignored so a double-click doesn't duplicate it); an open path is a
// guide rail.
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
	case PathHandle:
		switch t.guideKind {
		case loftGuideCenterline:
			ph := h
			t.centerline = &ph
		case loftGuideMapCurve:
			t.mapCurves = append(t.mapCurves, h)
		default:
			t.rails = append(t.rails, h)
		}
	}
}

// RailCount / MapCurveCount / HasCenterline report what guides have been picked.
func (t *LoftTool) RailCount() int      { return len(t.rails) }
func (t *LoftTool) MapCurveCount() int  { return len(t.mapCurves) }
func (t *LoftTool) HasCenterline() bool { return t.centerline != nil }

// SetGuideKind routes subsequent path picks to rails (0), the centerline (1), or map curves (2);
// GuideKind reports the current routing. Centerline and rails are mutually exclusive on commit (a
// centerline takes precedence). See the loftGuide* constants.
func (t *LoftTool) SetGuideKind(kind int) { t.guideKind = kind }
func (t *LoftTool) GuideKind() int        { return t.guideKind }

// SetUseCenterline is a convenience over SetGuideKind for the rails/centerline toggle.
func (t *LoftTool) SetUseCenterline(on bool) {
	if on {
		t.guideKind = loftGuideCenterline
	} else {
		t.guideKind = loftGuideRail
	}
}

// SetAreaMidScale sets the area-graph mid-height area scale (a barrel/waist); 0 or 1 disables it.
func (t *LoftTool) SetAreaMidScale(s float64) { t.areaMidScale = s }
func (t *LoftTool) AreaMidScale() float64     { return t.areaMidScale }

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
	guides := feature.LoftGuideSet{
		Rails:     t.railProviders(),
		MapCurves: t.mapCurveProviders(),
		AreaGraph: t.areaStops(),
	}
	if t.centerline != nil { // a centerline (spine) is exclusive with rails
		guides.Rails, guides.Centerline = nil, t.centerlineProvider()
	}
	t.added = feature.NewLoftFeatures(part.Features()).AddGuided(sections, t.closed, t.operation, t.first, t.last, guides)
	part.Recompute()
	s.recordEdit(part, "Loft")
	if !t.added.Health().OK() {
		return errors.New("loft: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// railProviders turns the picked guide-rail paths into live model-space polyline providers
// (re-derived each recompute, like the sweep path), for AddRailed.
func (t *LoftTool) railProviders() []func() []math.Point3 {
	var rails []func() []math.Point3
	for _, h := range t.rails {
		ph := h
		rails = append(rails, func() []math.Point3 {
			p, err := resolveSweepPath(&ph)
			if err != nil || p == nil {
				return nil
			}
			return p.Points()
		})
	}
	return rails
}

// mapCurveProviders turns the picked map-curve paths into live anchor-per-section providers.
func (t *LoftTool) mapCurveProviders() []func() []math.Point3 {
	var out []func() []math.Point3
	for _, h := range t.mapCurves {
		ph := h
		out = append(out, func() []math.Point3 {
			p, err := resolveSweepPath(&ph)
			if err != nil || p == nil {
				return nil
			}
			return p.Points()
		})
	}
	return out
}

// areaStops builds the area graph from the mid-scale control (one mid stop; off at 0/1).
func (t *LoftTool) areaStops() []feature.LoftAreaStop {
	if t.areaMidScale <= 0 || t.areaMidScale == 1 {
		return nil
	}
	return []feature.LoftAreaStop{{T: 0.5, Scale: t.areaMidScale}}
}

// centerlineProvider turns the picked spine path into a live model-space polyline provider.
func (t *LoftTool) centerlineProvider() func() []math.Point3 {
	ph := *t.centerline
	return func() []math.Point3 {
		p, err := resolveSweepPath(&ph)
		if err != nil || p == nil {
			return nil
		}
		return p.Points()
	}
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

// ClearSections empties the picked cross-sections — the property panel's selector
// clear (⊗) on the Sections chip.
func (t *LoftTool) ClearSections() { t.sections = nil }

// ClearGuides drops every guide pick — rails, centerline, and map curves — the
// property panel's selector clear (⊗) on the Guides chip.
func (t *LoftTool) ClearGuides() {
	t.rails = nil
	t.centerline = nil
	t.mapCurves = nil
}
