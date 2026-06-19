// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// partingAxisNames are the parting-axis choices in feature.PartingAxis order (Z is the
// common draw direction, so it is the zero value and the default).
var partingAxisNames = []string{"Z", "X", "Y"}

// CoreCavityTool splits the running tooling block into core and cavity solids by a planar
// parting (M10-F04, #701) — parameter-only: choose the parting axis, position and the
// shrinkage allowance in the generic tool dialog, then OK.
type CoreCavityTool struct {
	dialogTool
	axis      feature.PartingAxis
	position  float64
	shrinkage float64
	added     *feature.PartFeature
}

// NewCoreCavityTool returns a core/cavity tool defaulting to a Z parting at height 1.
func NewCoreCavityTool() *CoreCavityTool { return &CoreCavityTool{position: 1} }

// Name implements [Tool].
func (t *CoreCavityTool) Name() string { return "Core/Cavity" }

// Prompt guides the input.
func (t *CoreCavityTool) Prompt(*Session) string {
	return "Set the parting axis, position and shrinkage, then OK."
}

// Start/Pick implement [Tool] (parameter-only — the running block is the input).

// Params exposes the parting inputs for the generic property dialog.
func (t *CoreCavityTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{Label: "Parting position", Get: func() float64 { return t.position }, Set: func(v float64) { t.position = v }},
			{Label: "Shrinkage", Get: func() float64 { return t.shrinkage }, Set: func(v float64) { t.shrinkage = v }},
		},
		Choices: []ChoiceParam{{
			Label: "Parting axis", Options: partingAxisNames,
			Get: func() int { return int(t.axis) },
			Set: func(i int) {
				if i >= 0 && i < len(partingAxisNames) {
					t.axis = feature.PartingAxis(i)
				}
			},
		}},
	}
}

// CanCommit reports whether the shrinkage is sane (the position is validated against the
// block bounds at recompute, where the bounds are known).
func (t *CoreCavityTool) CanCommit() bool { return t.shrinkage >= 0 }

// Commit splits the tooling block at the parting plane and recomputes.
func (t *CoreCavityTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addCoreCavity(part.Features())
	part.Recompute()
	s.recordEdit(part, "Core/Cavity")
	if !t.added.Health().OK() {
		return errors.New("core/cavity: " + t.added.Health().Reason)
	}
	return nil
}

// addCoreCavity builds the core/cavity feature into engine fs — shared by Commit and preview.
func (t *CoreCavityTool) addCoreCavity(fs *feature.PartFeatures) *feature.PartFeature {
	pos := t.position
	return feature.NewCoreCavityFeatures(fs).AddByPartingPlaneFn(t.axis, func() float64 { return pos }, t.shrinkage)
}

// DraftFeature returns the unattached core/cavity feature the viewport previews before commit.
func (t *CoreCavityTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCoreCavity(fs), nil
	})
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *CoreCavityTool) AddedFeature() *feature.PartFeature { return t.added }
