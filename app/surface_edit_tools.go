// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
)

// The M10-F01/F02 surface-editing tool shells (#697): Ruled Surface, Surface Offset and
// Mid-Surface. The geometry lives in model/feature (tested there); these gather the inputs
// interactively — a profile pick for the ruled band, parameters through the generic tool
// dialog for all three — and add the feature on OK, so the wiring is covered headlessly.

// ruledDirectionNames are the ruled-surface direction choices in RuledSurfaceType order
// (tangent/perpendicular resolve their inputs then defer, per #339 — they read as Warning).
var ruledDirectionNames = []string{"Normal", "Tangent", "Perpendicular"}

// RuledSurfaceTool rules a closed sketch profile's edges into a band: pick the profile,
// choose the direction mode and distance, then OK.
type RuledSurfaceTool struct {
	profile  *ProfileHandle
	kind     feature.RuledSurfaceType
	distance float64
	added    *feature.PartFeature
}

// NewRuledSurfaceTool returns a ruled-surface tool defaulting to a 1-unit normal ruling.
func NewRuledSurfaceTool() *RuledSurfaceTool {
	return &RuledSurfaceTool{kind: feature.RuledNormal, distance: 1}
}

// Name implements [Tool].
func (t *RuledSurfaceTool) Name() string { return "Ruled Surface" }

// Prompt guides the pick.
func (t *RuledSurfaceTool) Prompt(*Session) string {
	return "Select a closed profile, set the direction and distance, then OK."
}

// Start filters selection to closed regions.
func (t *RuledSurfaceTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile))
}

// Pick captures the clicked closed region.
func (t *RuledSurfaceTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		pc := p
		t.profile = &pc
	}
}

// PickedProfile returns the clicked region (if any) for the unified tool highlight.
func (t *RuledSurfaceTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// SetDistance/Distance drive how far the rulings run.
func (t *RuledSurfaceTool) SetDistance(d float64) { t.distance = d }
func (t *RuledSurfaceTool) Distance() float64     { return t.distance }

// SetDirection/Direction drive the ruling direction mode (index into ruledDirectionNames).
func (t *RuledSurfaceTool) SetDirection(i int) {
	if i >= 0 && i < len(ruledDirectionNames) {
		t.kind = feature.RuledSurfaceType(i)
	}
}

// Direction returns the selected ruling mode index.
func (t *RuledSurfaceTool) Direction() int { return int(t.kind) }

// Params exposes the distance and direction mode for the generic property dialog.
func (t *RuledSurfaceTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Distance", Get: t.Distance, Set: t.SetDistance}},
		Choices: []ChoiceParam{{
			Label: "Direction", Options: ruledDirectionNames,
			Get: t.Direction, Set: t.SetDirection,
		}},
	}
}

// CanCommit reports whether a profile is picked and the distance is positive.
func (t *RuledSurfaceTool) CanCommit() bool { return t.profile != nil && t.distance > 0 }

// Commit rules the picked profile into a band and recomputes. A deferred direction mode
// (tangent/perpendicular) reads as Warning, which is not an error — only Sick is.
func (t *RuledSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d := t.distance
	t.added = feature.NewRuledSurfaceFeatures(part.Features()).
		AddByDistance(t.profile.Sketch, t.profile.ProfileIndex, t.kind, func() float64 { return d })
	part.Recompute()
	s.recordEdit(part, "Ruled Surface")
	if t.added.Health().Status == health.Sick {
		return errors.New("ruled surface: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *RuledSurfaceTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RuledSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }

// SurfaceOffsetTool offsets the running surface body along its normal by a distance —
// parameter-only (the running surface is the input, nothing is picked).
type SurfaceOffsetTool struct {
	dialogTool
	distance float64
	added    *feature.PartFeature
}

// NewSurfaceOffsetTool returns a surface-offset tool defaulting to a 1-unit offset.
func NewSurfaceOffsetTool() *SurfaceOffsetTool { return &SurfaceOffsetTool{distance: 1} }

// Name implements [Tool].
func (t *SurfaceOffsetTool) Name() string { return "Offset Surface" }

// Prompt guides the input.
func (t *SurfaceOffsetTool) Prompt(*Session) string {
	return "Set the offset distance for the running surface, then OK."
}

// Start implements [Tool] (no pick — the running surface is the input).

// Pick implements [Tool] (no pick).

// SetDistance/Distance drive how far the surface offsets (sign picks the side).
func (t *SurfaceOffsetTool) SetDistance(d float64) { t.distance = d }
func (t *SurfaceOffsetTool) Distance() float64     { return t.distance }

// Params exposes the offset distance for the generic property dialog.
func (t *SurfaceOffsetTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{Label: "Distance", Get: t.Distance, Set: t.SetDistance}}}
}

// CanCommit reports whether the offset is non-zero (sign is a valid side choice).
func (t *SurfaceOffsetTool) CanCommit() bool { return t.distance != 0 }

// Commit offsets the running surface and recomputes.
func (t *SurfaceOffsetTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d := t.distance
	t.added = feature.NewSurfaceOffsetFeatures(part.Features()).AddByDistance(func() float64 { return d })
	part.Recompute()
	s.recordEdit(part, "Offset Surface")
	if !t.added.Health().OK() {
		return errors.New("offset surface: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel implements [Tool] (nothing to restore — no selection filter was set).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SurfaceOffsetTool) AddedFeature() *feature.PartFeature { return t.added }

// MidSurfaceTool extracts mid-plane patches from the running solid's thin walls —
// parameter-only (face pairs within the threshold are found automatically).
type MidSurfaceTool struct {
	dialogTool
	maxThickness float64
	added        *feature.PartFeature
}

// NewMidSurfaceTool returns a mid-surface tool defaulting to a 2-unit wall threshold.
func NewMidSurfaceTool() *MidSurfaceTool { return &MidSurfaceTool{maxThickness: 2} }

// Name implements [Tool].
func (t *MidSurfaceTool) Name() string { return "Mid-Surface" }

// Prompt guides the input.
func (t *MidSurfaceTool) Prompt(*Session) string {
	return "Set the maximum wall thickness to pair faces under, then OK."
}

// Start implements [Tool] (no pick — the running solid is the input).

// Pick implements [Tool] (no pick).

// SetMaxThickness/MaxThickness drive the wall-pairing threshold.
func (t *MidSurfaceTool) SetMaxThickness(d float64) { t.maxThickness = d }
func (t *MidSurfaceTool) MaxThickness() float64     { return t.maxThickness }

// Params exposes the wall threshold for the generic property dialog.
func (t *MidSurfaceTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{
		Label: "Max thickness", Get: t.MaxThickness, Set: t.SetMaxThickness,
	}}}
}

// CanCommit reports whether the threshold is positive.
func (t *MidSurfaceTool) CanCommit() bool { return t.maxThickness > 0 }

// Commit extracts the mid-surfaces from the running solid and recomputes.
func (t *MidSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewMidSurfaceFeatures(part.Features()).AddByThickness(t.maxThickness)
	part.Recompute()
	s.recordEdit(part, "Mid-Surface")
	if !t.added.Health().OK() {
		return errors.New("mid-surface: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *MidSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }
