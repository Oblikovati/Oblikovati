// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// The Bridge feature (M36-F09) connects the last two surface bodies with a clean NURBS transition
// meeting each at a chosen continuity (G0/G1/G2) — the everyday Class-A "connect these two panels"
// move. Like Fill it appends a new surface body (the two neighbours stay). The bridge math is
// kernel/geom.BridgeSurface via ops.BridgeBodies; F13 is the numeric continuity gate.

// BridgeDefinition is the recipe for a bridge: the continuity order to hold to each neighbour
// (0=G0/position, 1=G1/tangent, 2=G2/curvature).
type BridgeDefinition struct {
	OrderA, OrderB int
}

// BridgeFeature connects the last two surface bodies with a transition surface.
type BridgeFeature struct {
	def      *BridgeDefinition
	featName string
}

// Definition returns the bridge recipe.
func (f *BridgeFeature) Definition() *BridgeDefinition { return f.def }

// Kind implements [Feature].
func (f *BridgeFeature) Kind() string { return "bridge-surface" }

// Recompute bridges the last two surface bodies and appends the bridge. It errors (→ sick) without
// two surface bodies or when the bridge is invalid.
func (f *BridgeFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) < 2 {
		return Output{}, fmt.Errorf("bridge surface: needs two surface bodies, have %d", len(in.Bodies))
	}
	n := len(in.Bodies)
	bridge, err := ops.BridgeBodies(in.Bodies[n-2], in.Bodies[n-1], f.def.OrderA, f.def.OrderB)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendBody(in.Bodies, bridge)}, nil
}

// BridgeFeatures adds bridge features into the engine.
type BridgeFeatures struct{ engine *PartFeatures }

// NewBridgeFeatures binds the collection to a feature engine.
func NewBridgeFeatures(engine *PartFeatures) *BridgeFeatures { return &BridgeFeatures{engine: engine} }

// Add bridges the last two surface bodies at the given per-side continuity orders.
func (c *BridgeFeatures) Add(orderA, orderB int) *PartFeature {
	def := &BridgeDefinition{OrderA: orderA, OrderB: orderB}
	bf := &BridgeFeature{def: def, featName: "Bridge"}
	pf := c.engine.Add(bf)
	bf.featName = pf.name
	return pf
}
