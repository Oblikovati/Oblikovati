// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
)

// The Match feature (M36-F05) rebuilds the running surface body against the previous surface body to
// a chosen continuity (G0/G1/G2/G3) — the defining Class-A construction move. It matches the running
// (source) surface's edge to the prior (target) surface's edge, so the two read as one fair surface
// across the seam. The continuity math is in kernel/geom.MatchSurface; the F13 cross-edge checker is
// the numeric acceptance gate.

// MatchDefinition is the recipe for a surface match: which edge of the source and target to align
// and to what continuity order (0=G0 … 3=G3).
type MatchDefinition struct {
	Order      int
	SourceEdge geom.Boundary
	TargetEdge geom.Boundary
}

// MatchFeature matches the running surface body to the previous one.
type MatchFeature struct {
	def      *MatchDefinition
	featName string
}

// Definition returns the match recipe.
func (m *MatchFeature) Definition() *MatchDefinition { return m.def }

// Kind implements [Feature].
func (m *MatchFeature) Kind() string { return "match-surface" }

// Recompute matches the last surface body's NURBS face to the second-to-last body's face and replaces
// the last body. It errors (→ sick) without two surface bodies or when the match is invalid.
func (m *MatchFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) < 2 {
		return Output{}, errors.New("match surface: needs a target surface body before the one being matched")
	}
	target := in.Bodies[len(in.Bodies)-2]
	src := in.Bodies[len(in.Bodies)-1]
	matched, err := ops.MatchFaceTo(src, target, m.def.SourceEdge, m.def.TargetEdge, m.def.Order)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, matched)}, nil
}

// MatchFeatures adds match features into the engine.
type MatchFeatures struct{ engine *PartFeatures }

// NewMatchFeatures binds the collection to a feature engine.
func NewMatchFeatures(engine *PartFeatures) *MatchFeatures { return &MatchFeatures{engine: engine} }

// Add matches the running surface body's sourceEdge to the previous body's targetEdge at the given
// continuity order.
func (c *MatchFeatures) Add(order int, sourceEdge, targetEdge geom.Boundary) *PartFeature {
	def := &MatchDefinition{Order: order, SourceEdge: sourceEdge, TargetEdge: targetEdge}
	mf := &MatchFeature{def: def, featName: "Match"}
	pf := c.engine.Add(mf)
	mf.featName = pf.name
	return pf
}
