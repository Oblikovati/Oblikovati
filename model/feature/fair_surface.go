// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/kernel/ops/surface"

// The Fair feature (M36-F04) smooths curvature wrinkles out of the running surface body's NURBS face
// while holding its boundary continuity (G0/G1/G2) to neighbours — the Class-A "take the wrinkles
// out" move. It replaces the running surface. The fairing math is kernel/geom.FairSurface via
// surface.FairFaceSurface; F13 is the boundary-continuity gate.

// FairDefinition is the recipe for a fairing: the boundary continuity to hold (0=G0,1=G1,2=G2), the
// per-iteration relaxation strength (0<s≤1), and the iteration count.
type FairDefinition struct {
	HoldOrder  int
	Strength   float64
	Iterations int
}

// FairFeature fairs the running surface body's NURBS face.
type FairFeature struct {
	def      *FairDefinition
	featName string
}

// Definition returns the fairing recipe.
func (f *FairFeature) Definition() *FairDefinition { return f.def }

// Kind implements [Feature].
func (f *FairFeature) Kind() string { return "fair-surface" }

// Recompute fairs the running surface body's NURBS face and replaces it.
func (f *FairFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "fair surface")
	if err != nil {
		return Output{}, err
	}
	out, err := surface.FairFaceSurface(target, f.def.HoldOrder, f.def.Strength, f.def.Iterations)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, out)}, nil
}

// FairFeatures adds fairing features into the engine.
type FairFeatures struct{ engine *PartFeatures }

// NewFairFeatures binds the collection to a feature engine.
func NewFairFeatures(engine *PartFeatures) *FairFeatures { return &FairFeatures{engine: engine} }

// Add fairs the running surface, holding holdOrder continuity, by strength over iterations.
func (c *FairFeatures) Add(holdOrder int, strength float64, iterations int) *PartFeature {
	def := &FairDefinition{HoldOrder: holdOrder, Strength: strength, Iterations: iterations}
	ff := &FairFeature{def: def, featName: "Fair"}
	pf := c.engine.Add(ff)
	ff.featName = pf.name
	return pf
}
