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

// captureFaceAnchors records each picked face key's mint-time anchor (its centroid) by resolving
// the key against the body it was picked on — the face counterpart of captureEdgeAnchors (#1579).
// A key that does not resolve to EXACTLY one face is skipped (degrading to ancestral-only
// recovery), and an empty capture returns nil so the caller stores nil rather than an empty map.
//
// Example: def.FaceAnchors = captureFaceAnchors(tipBody, [][]byte{def.PlacementFaceKey})
func captureFaceAnchors(body *topo.Body, keys [][]byte) map[string]math.Point3 {
	if body == nil || len(keys) == 0 {
		return nil
	}
	anchors := make(map[string]math.Point3, len(keys))
	for _, k := range keys {
		if len(k) == 0 {
			continue
		}
		match := body.FacesByKey(k)
		if len(match) != 1 {
			continue
		}
		if c, ok := faceAnchor(match[0]); ok {
			anchors[string(k)] = c
		}
	}
	if len(anchors) == 0 {
		return nil
	}
	return anchors
}

// faceAnchor returns a face's centroid — the representative point the geometric recovery tier
// ranks siblings against (so capture and ranking, faceEntity.Anchor, use one definition of "the
// face's anchor"). It reuses topo.DescribeFace so kernel/topo owns the centroid math; ok is false
// for a degenerate face whose centroid is the zero point with no contributing vertices.
func faceAnchor(f *topo.Face) (math.Point3, bool) {
	if f == nil {
		return math.Point3{}, false
	}
	return topo.DescribeFace(f).Centroid, true
}

// featuresTipBody returns the engine's current running body — the body a freshly-authored feature
// operates on, the one its picked keys were minted against. It is nil when the part has not been
// recomputed (a batch build before the first recompute), in which case anchor capture is skipped
// and recovery falls back to the ancestral tier. Boss/thread/unwrap authoring share it via the
// engine, so capture is uniform across every authoring path (not GUI-only — the P6b lesson).
func featuresTipBody(fs *PartFeatures) *topo.Body {
	bodies := fs.Result()
	if len(bodies) == 0 {
		return nil
	}
	return bodies[len(bodies)-1]
}

// tipBody returns the engine's current running body for a dress-up authoring path.
func (c *DressUpFeatures) tipBody() *topo.Body { return featuresTipBody(c.engine) }
