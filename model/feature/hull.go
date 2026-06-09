// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// HullFeature replaces the running solids with their single convex hull — OpenSCAD's hull()
// of the part's bodies. It wraps dissimilar primitives into one convex solid (two cylinders →
// a capsule; a post + a dome → a rounded standoff), which has no direct Inventor equivalent
// but is the idiom many printed parts are built from. Concave inputs are filled to their hull.
type HullFeature struct {
	featName string
}

// Kind implements [Feature].
func (f *HullFeature) Kind() string { return "hull" }

// Recompute hulls every running solid body into one. It errors if there is nothing solid to
// hull or the points are degenerate (collinear/coplanar), which the engine turns into Sick.
func (f *HullFeature) Recompute(in Input) (Output, error) {
	solids := solidBodies(in.Bodies)
	if len(solids) == 0 {
		return Output{}, fmt.Errorf("hull: no solid bodies to hull (have %d)", len(in.Bodies))
	}
	// The hull reads body vertices; an analytic cylinder has only its 2 seam vertices, so re-facet any
	// curved solid into a planar B-rep first (#129).
	for i, s := range solids {
		solids[i] = planarized(s, featOr(f.featName, "hull"))
	}
	hull, err := ops.ConvexHullOf(featOr(f.featName, "hull"), solids...)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: []*topo.Body{hull}}, nil
}

// solidBodies returns the subset of bodies that are closed solids (surfaces are skipped — a
// hull is taken over volume, not sheets).
func solidBodies(bodies []*topo.Body) []*topo.Body {
	out := make([]*topo.Body, 0, len(bodies))
	for _, b := range bodies {
		if b != nil && b.IsSolid() {
			out = append(out, b)
		}
	}
	return out
}

// HullFeatures adds hull features into the engine.
type HullFeatures struct{ engine *PartFeatures }

// NewHullFeatures binds the collection to an engine.
func NewHullFeatures(engine *PartFeatures) *HullFeatures { return &HullFeatures{engine} }

// Add appends a hull of the part's running solids.
func (c *HullFeatures) Add() *PartFeature {
	hf := &HullFeature{}
	pf := c.engine.Add(hf)
	pf.SetName(c.engine.UniqueName("Hull"))
	hf.featName = pf.name
	return pf
}
