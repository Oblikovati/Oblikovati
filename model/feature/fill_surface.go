// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The Fill feature (M36-F07) closes a four-sided opening bounded by the last four surface bodies with
// a single clean NURBS that interpolates their inner edges (Coons) and meets each neighbour at the
// chosen continuity (G0/G1/G2 — the Class-A boundary-fill move). Unlike Match/Extend it appends a new
// surface body rather than replacing the running one, so the four neighbours stay in the model. The
// fill math is kernel/geom.FillSurface via ops.FillFourSided; the F13 cross-edge checker is the
// numeric acceptance gate.

// FillDefinition is the recipe for a boundary fill: the continuity order to impose on every side
// (0=G0/position … 2=G2/curvature).
type FillDefinition struct {
	Order int
}

// FillFeature fills the opening bounded by the last four surface bodies.
type FillFeature struct {
	def      *FillDefinition
	featName string
}

// Definition returns the fill recipe.
func (f *FillFeature) Definition() *FillDefinition { return f.def }

// Kind implements [Feature].
func (f *FillFeature) Kind() string { return "fill-surface" }

// Recompute fills the opening bounded by the last four surface bodies and appends the fill body. It
// errors (→ sick) without four surface bodies or when the four edges do not form a closed loop.
func (f *FillFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) < 4 {
		return Output{}, fmt.Errorf("fill surface: needs four bounding surface bodies, have %d", len(in.Bodies))
	}
	n := len(in.Bodies)
	neighbours := [4]*topo.Body{in.Bodies[n-4], in.Bodies[n-3], in.Bodies[n-2], in.Bodies[n-1]}
	fill, err := ops.FillFourSided(neighbours, f.def.Order)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendBody(in.Bodies, fill)}, nil
}

// FillFeatures adds fill features into the engine.
type FillFeatures struct{ engine *PartFeatures }

// NewFillFeatures binds the collection to a feature engine.
func NewFillFeatures(engine *PartFeatures) *FillFeatures { return &FillFeatures{engine: engine} }

// Add fills the opening bounded by the last four surface bodies at the given continuity order.
func (c *FillFeatures) Add(order int) *PartFeature {
	def := &FillDefinition{Order: order}
	ff := &FillFeature{def: def, featName: "Fill"}
	pf := c.engine.Add(ff)
	ff.featName = pf.name
	return pf
}
