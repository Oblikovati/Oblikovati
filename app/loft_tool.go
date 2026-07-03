// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// LoftTool is the interactive Loft command: activate it, click two or more cross-sections in
// order — each a sketch region, or a vertex/work point to taper the loft to an apex (a cone or
// dome) — optionally close the loop, set the end conditions, and OK to blend them into a solid.
type LoftTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed loft (see editLoftTool)
	sections        []loftPick
	rails           []PathHandle // guide curves (kLoftWithRails) — picked sketch paths
	centerline      *PathHandle  // spine curve (kLoftWithCenterline)
	mapCurves       []PathHandle // explicit correspondence anchors (MapPointCurves)
	guideKind       int          // where a path pick goes: 0 rail, 1 centerline, 2 map curve
	areaMidScale    float64      // area-graph: cross-section area scale at mid height (0/1 = off)
	closed          bool
	operation       ops.PartFeatureOperation
	first, last     feature.LoftEnd // end-section conditions (zero value = Free); see SetFirstCondition
	added           *feature.PartFeature
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

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *LoftTool) Start(*Session) {}

// AcceptedKinds declares loft picks sketch regions (profiles), points (vertices / work points) for
// an apex, a body face it can leave tangent, or an open sketch path as a guide rail.
func (t *LoftTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectProfile, SelectVertex, SelectWorkPoint, SelectFace, SelectPath}
}

// Picks reports the picked profile cross-sections for the unified highlight (point/face sections
// have nothing to outline, matching the prior PickedProfiles-based highlight).
func (t *LoftTool) Picks() []Selectable { return selectables(t.PickedProfiles()) }

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

// LoftSectionKind names what a picked cross-section is, so the Sections list can show and icon each
// row (a profile region, an apex point, or a tangent body face) — Inventor's Curves-tab section list.
type LoftSectionKind int

const (
	// LoftSectionProfile is a sketch profile region.
	LoftSectionProfile LoftSectionKind = iota
	// LoftSectionPoint is an apex point (a vertex or work point the loft tapers to).
	LoftSectionPoint
	// LoftSectionFace is an existing body face the loft leaves tangent.
	LoftSectionFace
)

// SectionKindAt reports the kind of cross-section i (LoftSectionProfile for an out-of-range index, so
// the UI degrades gracefully).
func (t *LoftTool) SectionKindAt(i int) LoftSectionKind {
	if i < 0 || i >= len(t.sections) {
		return LoftSectionProfile
	}
	switch s := t.sections[i]; {
	case s.isPoint():
		return LoftSectionPoint
	case s.isFace():
		return LoftSectionFace
	default:
		return LoftSectionProfile
	}
}

// SectionLabel returns a human label for cross-section i in display order: a profile shows its source
// sketch name (Inventor lists "Sketch2"), a point shows "Point", a face shows "Face". Empty for an
// out-of-range index.
func (t *LoftTool) SectionLabel(i int) string {
	if i < 0 || i >= len(t.sections) {
		return ""
	}
	switch s := t.sections[i]; {
	case s.isPoint():
		return "Point"
	case s.isFace():
		return "Face"
	case s.profile.Sketch != nil:
		return s.profile.Sketch.Name()
	default:
		return "Profile"
	}
}

// RemoveSection deletes cross-section i — the Sections list's per-row delete. Out-of-range is a no-op.
// The remaining sections keep their order, so the blend sequence stays meaningful.
func (t *LoftTool) RemoveSection(i int) {
	if i < 0 || i >= len(t.sections) {
		return
	}
	t.sections = append(t.sections[:i], t.sections[i+1:]...)
}

// MoveSection relocates cross-section from to index to, shifting the rest — the drag-and-drop reorder
// of the Sections list (the order IS the blend order, so this reshapes the loft). Out-of-range or
// no-op indices leave the order unchanged. Mirrors SelectionFilterState.Move.
func (t *LoftTool) MoveSection(from, to int) {
	n := len(t.sections)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}
	s := t.sections[from]
	t.sections = append(t.sections[:from], t.sections[from+1:]...)
	t.sections = append(t.sections[:to], append([]loftPick{s}, t.sections[to:]...)...)
}

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
// the tool open by returning an error. In edit mode it writes the panel state back into the
// committed loft instead (commitEdit).
func (t *LoftTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addLoft(part.Features())
	part.Recompute()
	s.recordEdit(part, "Loft")
	if !t.added.Health().OK() {
		return errors.New("loft: " + t.added.Health().Reason)
	}
	return nil
}

// commitEdit writes the panel state back into the committed loft's definition — the same inputs the
// create path passes to AddGuided. Sections, closure, operation, end conditions and the area graph
// are always rewritten. The guide providers (rails / centerline / map curves) are opaque live
// closures the panel cannot reverse into re-pickable handles, so they are PRESERVED untouched unless
// the user re-picked them this edit. Setting the static end conditions clears LiveEnds so the panel
// values are authoritative (a parametric end-angle linkage is replaced by the explicit edit).
func (t *LoftTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.LoftFeature).Definition()
	def.Sections = t.loftSections()
	def.Closed, def.Operation = t.closed, t.operation
	def.First, def.Last, def.LiveEnds = t.first, t.last, nil
	def.AreaGraph = t.areaStops()
	t.applyRepickedGuides(def)
	return commitFeatureEdit(s, t.target)
}

// applyRepickedGuides overwrites only the guides the user actually re-picked this edit, preserving the
// committed loft's existing opaque guide providers otherwise. A centerline (spine) is exclusive with
// rails, matching addLoft.
func (t *LoftTool) applyRepickedGuides(def *feature.LoftDefinition) {
	if len(t.rails) > 0 {
		def.Rails = t.railProviders()
	}
	if t.centerline != nil {
		def.Rails, def.Centerline = nil, t.centerlineProvider()
	}
	if len(t.mapCurves) > 0 {
		def.MapCurves = t.mapCurveProviders()
	}
}

// loftSections maps the picked cross-sections to feature.LoftSections — a profile region, a point
// (apex), or a body face (by key). Shared by addLoft (create) and commitEdit (re-edit).
func (t *LoftTool) loftSections() []feature.LoftSection {
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
	return sections
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
// addLoft assembles the cross-sections and guide set and builds the loft feature into engine
// fs — the shared constructor used by both Commit (the part's engine) and DraftFeature (a
// scratch engine), so the preview matches the committed result.
func (t *LoftTool) addLoft(fs *feature.PartFeatures) *feature.PartFeature {
	sections := t.loftSections()
	guides := feature.LoftGuideSet{
		Rails:     t.railProviders(),
		MapCurves: t.mapCurveProviders(),
		AreaGraph: t.areaStops(),
	}
	if t.centerline != nil { // a centerline (spine) is exclusive with rails
		guides.Rails, guides.Centerline = nil, t.centerlineProvider()
	}
	return feature.NewLoftFeatures(fs).AddGuided(sections, t.closed, t.operation, t.first, t.last, guides)
}

// DraftFeature returns the unattached loft feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addLoft the commit uses. Empty until at
// least two cross-sections are picked.
func (t *LoftTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addLoft(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *LoftTool) Cancel(*Session) {}

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

// AutomaticMapping reports whether section correspondence is automatic — no map curves are picked,
// so the loft aligns sections to minimise twist. The Transition tab toggles between this and an
// explicit point mapping (#1521).
func (t *LoftTool) AutomaticMapping() bool { return len(t.mapCurves) == 0 }

// ArmMapCurvePicking routes subsequent viewport path picks to the map-curve point mapping — the
// Transition tab's "pick map curves" action. Each picked open path carries one anchor point per
// section, overriding the automatic minimum-twist alignment so chosen points line up across the loft.
func (t *LoftTool) ArmMapCurvePicking() { t.guideKind = loftGuideMapCurve }

// ClearMapCurves drops the point-mapping picks, returning the loft to automatic section alignment —
// the Transition tab's reset. Rails and the centerline are left untouched.
func (t *LoftTool) ClearMapCurves() { t.mapCurves = nil }
