// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops/heal"
)

// The Fill feature (M36-F07) closes a four-sided opening bounded by the last four surface bodies with
// a single clean NURBS that interpolates their inner edges (Coons) and meets each neighbour at the
// chosen continuity (G0/G1/G2 — the Class-A boundary-fill move). Unlike Match/Extend it appends a new
// surface body rather than replacing the running one, so the four neighbours stay in the model. The
// fill math is kernel/geom.FillSurface via heal.FillFourSided; the F13 cross-edge checker is the
// numeric acceptance gate.

// DefaultFillSides is the classic four-sided boundary fill — the side count assumed when none is
// given (Sides ≤ 0), shared by the fill feature, tool and MCP op.
const DefaultFillSides = 4

// FillDefinition is the recipe for a boundary fill: the continuity order to impose on every side
// (0=G0/position … 2=G2/curvature) and the number of bounding sides (0 or 4 = the classic four-sided
// fill; 3, 5, 6… = an N-sided fill of the last Sides surface bodies).
type FillDefinition struct {
	Order int
	Sides int
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

// Recompute fills the opening bounded by the last Sides surface bodies and appends the fill body. It
// errors (→ sick) without that many surface bodies or when the edges do not form a closed loop.
func (f *FillFeature) Recompute(in Input) (Output, error) {
	sides := f.def.Sides
	if sides <= 0 {
		sides = DefaultFillSides
	}
	if len(in.Bodies) < sides {
		return Output{}, fmt.Errorf("fill surface: needs %d bounding surface bodies, have %d", sides, len(in.Bodies))
	}
	neighbours := in.Bodies[len(in.Bodies)-sides:]
	fill, err := heal.FillNSided(neighbours, f.def.Order)
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
func (c *FillFeatures) Add(order int) *PartFeature { return c.AddSides(order, 4) }

// AddSides fills the opening bounded by the last sides surface bodies (3, 4, 5, …) at the given
// continuity order. With sides other than 4 the opening is mapped onto four logical sides (N-sided
// fill, #1300); merged or split sides fill position-only (G0).
func (c *FillFeatures) AddSides(order, sides int) *PartFeature {
	def := &FillDefinition{Order: order, Sides: sides}
	ff := &FillFeature{def: def, featName: "Fill"}
	pf := c.engine.Add(ff)
	ff.featName = pf.name
	return pf
}
