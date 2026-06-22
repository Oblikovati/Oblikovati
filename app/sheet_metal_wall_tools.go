// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// Sheet-metal wall tools (M13-F02): the Create-panel walls beyond Face and Flange. Each is an
// interactive Tool that picks geometry and commits a wall at the active rule's gauge.

// SheetMetalHemTool folds a hem (a 180° wall) back on a straight edge.
type SheetMetalHemTool struct {
	edge   *EdgeHandle
	length float64
	added  *feature.PartFeature
}

// NewSheetMetalHemTool returns a hem tool defaulting to a 6 mm fold-back.
func NewSheetMetalHemTool() *SheetMetalHemTool { return &SheetMetalHemTool{length: 0.6} }

func (t *SheetMetalHemTool) Name() string   { return "Sheet Metal Hem" }
func (t *SheetMetalHemTool) Start(*Session) {}

// AcceptedKinds declares the hem picks an edge (the edge to hem).
func (t *SheetMetalHemTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edge for the unified highlight.
func (t *SheetMetalHemTool) Picks() []Selectable {
	if t.edge == nil {
		return nil
	}
	return []Selectable{*t.edge}
}
func (t *SheetMetalHemTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		t.edge = &e
	}
}
func (t *SheetMetalHemTool) SetLength(l float64)                { t.length = l }
func (t *SheetMetalHemTool) Length() float64                    { return t.length }
func (t *SheetMetalHemTool) CanCommit() bool                    { return t.edge != nil && t.length > 0 }
func (t *SheetMetalHemTool) Cancel(*Session)                    {}
func (t *SheetMetalHemTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalHemTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal hem: pick an edge and set a positive length")
	}
	length := t.length
	t.added = feature.NewSheetMetalHemFeatures(part.Features()).Add(&feature.SheetMetalHemDefinition{
		EdgeKey: t.edge.Edge.ReferenceKey(), Length: func() float64 { return length }, Type: feature.ClosedHem,
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Hem")
}

// SheetMetalContourFlangeTool sweeps an open sketch profile along a straight edge.
type SheetMetalContourFlangeTool struct {
	edge    *EdgeHandle
	profile *ProfileHandle
	added   *feature.PartFeature
}

// NewSheetMetalContourFlangeTool returns a contour-flange tool awaiting an edge and a profile.
func NewSheetMetalContourFlangeTool() *SheetMetalContourFlangeTool {
	return &SheetMetalContourFlangeTool{}
}

func (t *SheetMetalContourFlangeTool) Name() string   { return "Sheet Metal Contour Flange" }
func (t *SheetMetalContourFlangeTool) Start(*Session) {}

// AcceptedKinds declares the contour flange picks an edge and a profile (the contour to sweep).
func (t *SheetMetalContourFlangeTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectEdge, SelectProfile}
}

// Picks reports the picked edge and profile for the unified highlight.
func (t *SheetMetalContourFlangeTool) Picks() []Selectable {
	var picks []Selectable
	if t.edge != nil {
		picks = append(picks, *t.edge)
	}
	if t.profile != nil {
		picks = append(picks, *t.profile)
	}
	return picks
}
func (t *SheetMetalContourFlangeTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case EdgeHandle:
		t.edge = &h
	case ProfileHandle:
		t.profile = &h
	}
}
func (t *SheetMetalContourFlangeTool) CanCommit() bool { return t.edge != nil && t.profile != nil }
func (t *SheetMetalContourFlangeTool) Cancel(s *Session) {
}
func (t *SheetMetalContourFlangeTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalContourFlangeTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal contour flange: pick an edge and an open sketch profile")
	}
	t.added = feature.NewSheetMetalContourFlangeFeatures(part.Features()).Add(&feature.SheetMetalContourFlangeDefinition{
		EdgeKey: t.edge.Edge.ReferenceKey(), Profile: t.profile.Sketch,
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Contour Flange")
}

// SheetMetalLoftedFlangeTool lofts a wall between two sketch profiles.
type SheetMetalLoftedFlangeTool struct {
	profiles []ProfileHandle
	added    *feature.PartFeature
}

// NewSheetMetalLoftedFlangeTool returns a lofted-flange tool awaiting two profiles.
func NewSheetMetalLoftedFlangeTool() *SheetMetalLoftedFlangeTool {
	return &SheetMetalLoftedFlangeTool{}
}

func (t *SheetMetalLoftedFlangeTool) Name() string   { return "Sheet Metal Lofted Flange" }
func (t *SheetMetalLoftedFlangeTool) Start(*Session) {}

// AcceptedKinds declares the lofted flange picks two closed sketch regions (profiles).
func (t *SheetMetalLoftedFlangeTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectProfile}
}

// Picks reports the picked regions for the unified highlight.
func (t *SheetMetalLoftedFlangeTool) Picks() []Selectable { return profileSelectables(t.profiles) }
func (t *SheetMetalLoftedFlangeTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok && len(t.profiles) < 2 {
		t.profiles = append(t.profiles, p)
	}
}
func (t *SheetMetalLoftedFlangeTool) CanCommit() bool { return len(t.profiles) == 2 }
func (t *SheetMetalLoftedFlangeTool) Cancel(s *Session) {
}
func (t *SheetMetalLoftedFlangeTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalLoftedFlangeTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal lofted flange: pick two sketch profiles")
	}
	t.added = feature.NewSheetMetalLoftedFlangeFeatures(part.Features()).Add(&feature.SheetMetalLoftedFlangeDefinition{
		ProfileA: t.profiles[0].Sketch, ProfileB: t.profiles[1].Sketch, Operation: ops.Join,
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Lofted Flange")
}

// SheetMetalContourRollTool revolves an open profile about an axis line in the same sketch.
type SheetMetalContourRollTool struct {
	profile *ProfileHandle
	axis    *SketchEntityHandle
	angle   float64
	added   *feature.PartFeature
}

// NewSheetMetalContourRollTool returns a contour-roll tool defaulting to a 90° roll.
func NewSheetMetalContourRollTool() *SheetMetalContourRollTool {
	return &SheetMetalContourRollTool{angle: halfPiAngle}
}

func (t *SheetMetalContourRollTool) Name() string   { return "Sheet Metal Contour Roll" }
func (t *SheetMetalContourRollTool) Start(*Session) {}

// AcceptedKinds declares the contour roll picks a profile (the contour) and a sketch entity (the
// roll axis line).
func (t *SheetMetalContourRollTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectProfile, SelectSketchEntity}
}

// Picks reports the picked profile for the unified highlight (the axis is a sketch line).
func (t *SheetMetalContourRollTool) Picks() []Selectable {
	if t.profile == nil {
		return nil
	}
	return []Selectable{*t.profile}
}
func (t *SheetMetalContourRollTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		t.profile = &h
	case SketchEntityHandle:
		t.axis = &h
	}
}
func (t *SheetMetalContourRollTool) SetAngle(a float64) { t.angle = a }
func (t *SheetMetalContourRollTool) Angle() float64     { return t.angle }
func (t *SheetMetalContourRollTool) CanCommit() bool {
	return t.profile != nil && t.axis != nil && t.angle > 0
}
func (t *SheetMetalContourRollTool) Cancel(s *Session) {
}
func (t *SheetMetalContourRollTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalContourRollTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal contour roll: pick an open profile, an axis line, and set an angle")
	}
	_, axisIndex, ok := lineHandleInSketch(t.profile.Sketch, t.axis.Entity)
	if !ok {
		return errors.New("sheet-metal contour roll: the axis line must belong to the profile sketch")
	}
	angle := t.angle
	t.added = feature.NewSheetMetalContourRollFeatures(part.Features()).Add(&feature.SheetMetalContourRollDefinition{
		Profile: t.profile.Sketch, AxisLine: axisIndex, Angle: func() float64 { return angle }, Operation: ops.Join,
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Contour Roll")
}
