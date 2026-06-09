// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mold core/cavity tooling (M10-F04, PBI-116). Phase A splits the tooling block's
// axis-aligned bounds by a planar parting at a given height into two solid halves
// (core below, cavity above), recording a shrinkage allowance. The part-shaped pocket
// (block minus the molded part) and non-planar parting surfaces along the part's
// silhouette are general solid–solid booleans, kernel phase C.

// PartingAxis is the block axis the parting plane is perpendicular to.
type PartingAxis uint8

const (
	// PartingZ splits along Z (the most common draw direction).
	PartingZ PartingAxis = iota
	// PartingX / PartingY split along X / Y.
	PartingX
	PartingY
)

// CoreCavityDefinition is the recipe for a mold split: the parting axis and position,
// and the shrinkage allowance (fractional oversize compensating part shrink).
type CoreCavityDefinition struct {
	Axis      PartingAxis
	Position  func() float64
	Shrinkage float64
}

// CoreCavityFeature splits the running tooling block into a core and a cavity solid.
type CoreCavityFeature struct {
	def      *CoreCavityDefinition
	featName string
}

// Definition returns the mold recipe.
func (m *CoreCavityFeature) Definition() *CoreCavityDefinition { return m.def }

// Kind implements [Feature].
func (m *CoreCavityFeature) Kind() string { return "core-cavity" }

// Recompute splits the block's bounds at the parting plane into core (below) and
// cavity (above) solids, erroring if the parting falls outside the block.
func (m *CoreCavityFeature) Recompute(in Input) (Output, error) {
	block, err := lastBody(in, "core-cavity")
	if err != nil {
		return Output{}, err
	}
	box := block.RangeBox()
	lo, hi := axisRange(box, m.def.Axis)
	pos := m.def.Position()
	if pos <= lo || pos >= hi {
		return Output{}, errors.New("core-cavity: parting plane is outside the tooling block")
	}
	core := boxSolid(box, m.def.Axis, lo, pos, m.featName+"-core")
	cavity := boxSolid(box, m.def.Axis, pos, hi, m.featName+"-cavity")
	return Output{Bodies: []*topo.Body{core, cavity}}, nil
}

// CoreCavityFeatures adds mold core/cavity features into the engine.
type CoreCavityFeatures struct{ engine *PartFeatures }

// NewCoreCavityFeatures binds the collection to a feature engine.
func NewCoreCavityFeatures(engine *PartFeatures) *CoreCavityFeatures {
	return &CoreCavityFeatures{engine: engine}
}

// AddByPartingPlane splits the running block at position along axis, with shrinkage.
func (c *CoreCavityFeatures) AddByPartingPlane(axis PartingAxis, position, shrinkage float64) *PartFeature {
	return c.AddByPartingPlaneFn(axis, constFloat(position), shrinkage)
}

// AddByPartingPlaneFn is AddByPartingPlane with a live (parameter-driven) position.
func (c *CoreCavityFeatures) AddByPartingPlaneFn(axis PartingAxis, position func() float64, shrinkage float64) *PartFeature {
	def := &CoreCavityDefinition{Axis: axis, Position: position, Shrinkage: shrinkage}
	mf := &CoreCavityFeature{def: def, featName: "Mold"}
	pf := c.engine.Add(mf)
	mf.featName = pf.name
	return pf
}

// axisRange returns the box's [min,max] extent along the given axis.
func axisRange(box math.Box, axis PartingAxis) (float64, float64) {
	switch axis {
	case PartingX:
		return box.Min.X, box.Max.X
	case PartingY:
		return box.Min.Y, box.Max.Y
	default:
		return box.Min.Z, box.Max.Z
	}
}

// boxSolid builds an axis-aligned box solid spanning box's cross-section, with the
// chosen axis clamped to [lo,hi] (one half of the split block).
func boxSolid(box math.Box, axis PartingAxis, lo, hi float64, feat string) *topo.Body {
	min, max := box.Min, box.Max
	switch axis {
	case PartingX:
		min.X, max.X = lo, hi
	case PartingY:
		min.Y, max.Y = lo, hi
	default:
		min.Z, max.Z = lo, hi
	}
	cage := subd.Box(max.X-min.X, max.Y-min.Y, max.Z-min.Z)
	shift := math.V3(min.X, min.Y, min.Z)
	for i, p := range cage.Verts {
		cage.Verts[i] = p.TranslateBy(shift)
	}
	return subd.ToBody(cage, feat)
}
