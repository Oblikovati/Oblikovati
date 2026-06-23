// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The NURBS Plane tool (M36-F03) drops a flat, clean degree-3 NURBS plane patch into the part — a
// base freeform surface to start Class-A styling from (then shape it with Edit Control Points). It
// is parameter-only (size + control-point count), driven by the generic property dialog.

// NurbsPlaneTool creates a flat NURBS plane patch as a surface body.
type NurbsPlaneTool struct {
	dialogTool
	width, height  float64
	uCount, vCount int
	added          *feature.PartFeature
}

// NewNurbsPlaneTool returns the NURBS plane tool defaulting to a 10×10, 4×4-control-point patch.
func NewNurbsPlaneTool() *NurbsPlaneTool {
	return &NurbsPlaneTool{width: 10, height: 10, uCount: 4, vCount: 4}
}

// Name implements [Tool].
func (t *NurbsPlaneTool) Name() string { return "NURBS Plane" }

// Prompt guides the input.
func (t *NurbsPlaneTool) Prompt(*Session) string {
	return "Set the size and control-point count, then OK to create the NURBS plane patch."
}

// Params exposes the size and control-point count for the generic dialog.
func (t *NurbsPlaneTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{Label: "Width", Get: func() float64 { return t.width }, Set: func(v float64) { t.width = v }},
			{Label: "Height", Get: func() float64 { return t.height }, Set: func(v float64) { t.height = v }},
		},
		Ints: []IntParam{
			{Label: "U Control Points", Get: func() int { return t.uCount }, Set: func(v int) { t.uCount = v }},
			{Label: "V Control Points", Get: func() int { return t.vCount }, Set: func(v int) { t.vCount = v }},
		},
	}
}

// CanCommit reports whether the patch is valid (positive size, cubic-capable control counts).
func (t *NurbsPlaneTool) CanCommit() bool {
	return t.width > 0 && t.height > 0 && t.uCount >= 4 && t.vCount >= 4
}

// Commit creates the NURBS plane patch and recomputes.
func (t *NurbsPlaneTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewNurbsPlaneFeatures(part.Features()).Add(t.width, t.height, t.uCount, t.vCount)
	part.Recompute()
	s.recordEdit(part, "NURBS Plane")
	if !t.added.Health().OK() {
		return errors.New("nurbs plane: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *NurbsPlaneTool) AddedFeature() *feature.PartFeature { return t.added }
