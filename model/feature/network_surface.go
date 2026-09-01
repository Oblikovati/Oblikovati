// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops/surface"

	"oblikovati.org/math"
)

// The Network feature (M36-F10) builds a single NURBS interpolating a grid of intersecting U- and
// V-direction curves — the Class-A skinning tool for complex panels (a Gordon surface). The curves
// are stored as baked model-space polylines (the tool/op bake them from picked sketch curves); the
// feature fits each and builds the network via surface.NetworkSurfaceBody. It appends a surface body.

// NetworkDefinition is the recipe for a network surface: the U- and V-direction curve polylines.
type NetworkDefinition struct {
	UCurves [][]math.Point3
	VCurves [][]math.Point3
}

// NetworkFeature builds a network surface from a grid of curves.
type NetworkFeature struct {
	def      *NetworkDefinition
	featName string
}

// Definition returns the network recipe.
func (f *NetworkFeature) Definition() *NetworkDefinition { return f.def }

// Kind implements [Feature].
func (f *NetworkFeature) Kind() string { return "network-surface" }

// Recompute builds the network surface and appends it as a surface body. It errors (→ sick) without
// at least two curves each way or when the curves do not form a usable grid.
func (f *NetworkFeature) Recompute(in Input) (Output, error) {
	if len(f.def.UCurves) < 2 || len(f.def.VCurves) < 2 {
		return Output{}, fmt.Errorf("network surface: needs ≥2 curves each way, have %d u and %d v", len(f.def.UCurves), len(f.def.VCurves))
	}
	body, err := surface.NetworkSurfaceBody(f.def.UCurves, f.def.VCurves)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendBody(in.Bodies, body)}, nil
}

// NetworkFeatures adds network features into the engine.
type NetworkFeatures struct{ engine *PartFeatures }

// NewNetworkFeatures binds the collection to a feature engine.
func NewNetworkFeatures(engine *PartFeatures) *NetworkFeatures {
	return &NetworkFeatures{engine: engine}
}

// Add builds a network surface from the U- and V-direction curve polylines.
func (c *NetworkFeatures) Add(uCurves, vCurves [][]math.Point3) *PartFeature {
	def := &NetworkDefinition{UCurves: uCurves, VCurves: vCurves}
	nf := &NetworkFeature{def: def, featName: "Network"}
	pf := c.engine.Add(nf)
	nf.featName = pf.name
	return pf
}
