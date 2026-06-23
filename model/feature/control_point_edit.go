// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
)

// The control-point edit feature (M36-F03) is one direct manipulation of the running surface
// body's NURBS control net — a single-CV nudge, a row/column drag, or a region pull with falloff,
// all expressed as the per-control-point displacements the tool computed. Each edit is its own
// feature so the recipe-snapshot undo reverts one drag at a time. It edits the actual B-spline
// (degree/knots preserved), distinct from the sub-D cage of [FreeformBody].

// ControlPointEditDefinition is the recipe for one control-net edit: the per-CV displacements
// applied to the running surface body's NURBS face.
type ControlPointEditDefinition struct {
	Deltas []geom.ControlPointDelta
}

// ControlPointEditFeature displaces the running surface body's NURBS control net.
type ControlPointEditFeature struct {
	def      *ControlPointEditDefinition
	featName string
}

// Definition returns the control-net edit recipe.
func (c *ControlPointEditFeature) Definition() *ControlPointEditDefinition { return c.def }

// Kind implements [Feature].
func (c *ControlPointEditFeature) Kind() string { return "control-point-edit" }

// Recompute applies the control-point displacements to the running surface body's NURBS face and
// replaces it. A missing target body or non-NURBS body → sick.
func (c *ControlPointEditFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "control-point edit")
	if err != nil {
		return Output{}, err
	}
	edited, err := ops.EditControlPoints(target, c.def.Deltas)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, edited)}, nil
}

// ControlPointEditFeatures adds control-net edits into the engine.
type ControlPointEditFeatures struct{ engine *PartFeatures }

// NewControlPointEditFeatures binds the collection to a feature engine.
func NewControlPointEditFeatures(engine *PartFeatures) *ControlPointEditFeatures {
	return &ControlPointEditFeatures{engine: engine}
}

// Add records one control-net edit (the given per-CV displacements) as a feature.
func (c *ControlPointEditFeatures) Add(deltas []geom.ControlPointDelta) *PartFeature {
	cf := &ControlPointEditFeature{def: &ControlPointEditDefinition{Deltas: deltas}, featName: "EditControlPoints"}
	pf := c.engine.Add(cf)
	cf.featName = pf.name
	return pf
}
