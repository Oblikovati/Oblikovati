// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mint-time anchor capture is the create-time half of the geometric recovery tier (ADR-0043
// P6b). The tier disambiguates several surviving same-parent siblings of a lost edge reference
// by nearness to the edge midpoint witnessed when the edge was first picked. That witness has
// to be recorded at authoring time, against the body the edge was picked on — and uniformly
// across EVERY authoring path (GUI, wire API, assembly, programmatic), not just the GUI, or the
// tier silently degrades to ancestral-only depending on how the feature was created. This file
// is that single, creation-path-agnostic capture seam; the recipe restore deliberately does NOT
// go through it, so reopening a document never rewrites its recipe.

// captureEdgeAnchors records each picked edge key's mint-time anchor (its midpoint) by
// resolving the key against the body it was picked on. It is the model-layer counterpart of
// the GUI's edgeHandleAnchors: the GUI holds resolved edge handles, every other authoring path
// holds only keys, so they resolve here. A key that does not resolve to EXACTLY one edge is
// skipped (a fresh selection always resolves; anything else degrades to ancestral-only
// recovery), and an empty capture returns nil so the caller stores nil rather than an empty map.
//
// Example: def.EdgeAnchors = captureEdgeAnchors(tipBody, def.EdgeKeys)
func captureEdgeAnchors(body *topo.Body, keys [][]byte) map[string]math.Point3 {
	if body == nil || len(keys) == 0 {
		return nil
	}
	anchors := make(map[string]math.Point3, len(keys))
	for _, k := range keys {
		match := body.EdgesByKey(k)
		if len(match) != 1 {
			continue
		}
		if mid, ok := edgeMidpoint(match[0]); ok {
			anchors[string(k)] = mid
		}
	}
	if len(anchors) == 0 {
		return nil
	}
	return anchors
}

// edgeMidpoint returns the midpoint of an edge's two end vertices — the representative point the
// geometric recovery tier ranks siblings against (so capture and ranking, edgeEntity.Anchor,
// use one definition of "the edge's anchor"). ok is false for a degenerate edge missing an endpoint.
func edgeMidpoint(e *topo.Edge) (math.Point3, bool) {
	s, end := e.StartVertex(), e.EndVertex()
	if s == nil || end == nil {
		return math.Point3{}, false
	}
	return s.Point().Midpoint(end.Point()), true
}

// tipBody returns the engine's current running body — the body a freshly-authored dress-up
// operates on, the one its keys were minted against. It is nil when the part has not been
// recomputed (a batch build before the first recompute), in which case anchor capture is
// skipped and recovery falls back to the ancestral tier.
func (c *DressUpFeatures) tipBody() *topo.Body {
	bodies := c.engine.Result()
	if len(bodies) == 0 {
		return nil
	}
	return bodies[len(bodies)-1]
}
