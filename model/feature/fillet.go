// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// cornerStrategy maps the public corner-type discriminator to the kernel's 2-edge corner strategy.
func cornerStrategy(t types.FilletCornerType) ops.CornerStrategy {
	switch t {
	case types.FilletCornerSetback:
		return ops.CornerSetback
	case types.FilletCornerRound:
		return ops.CornerRound
	default:
		return ops.CornerMiter
	}
}

// filletBody rounds the selected convex edges of the running body to the given radius via
// ops.FilletEdgesCorner (a real rolling-ball blend with cylinder faces), replacing it in the body
// list. corner selects how a 2-edge corner is treated. A lost edge, a non-convex edge, or a
// non-positive radius is an error so the feature goes Sick. See kernel/ops/fillet.go for the geometry.
func filletBody(in Input, edgeKeys [][]byte, radius float64, corner FilletCornerType, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if radius <= 0 {
		return Output{}, fmt.Errorf("%s: radius %g must be > 0", feat, radius)
	}
	if out, ok, err := analyticFilletFastPath(in, body, edgeKeys, radius, feat); ok || err != nil {
		return out, err
	}
	work, keys := planarizedFillet(body, edgeKeys, feat)
	picks := make([]ops.EdgeFilletRadii, len(keys))
	for i, k := range keys {
		picks[i] = ops.EdgeFilletRadii{Key: k, R0: radius, R1: radius}
	}
	result, err := ops.FilletEdgesCorner(work, picks, cornerStrategy(corner))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// analyticFilletFastPath rounds a curved fillet target directly on the ANALYTIC body, before the
// planarize step that would re-facet the cylinder and destroy the circle/arc it needs: a SIMPLE
// cylinder rim becomes a surface of revolution (#127); the cylinder/cap RIM or ARC a prior fillet
// leaves becomes a toroidal band / torus + setback caps. ok=false means no analytic case applied.
func analyticFilletFastPath(in Input, body *topo.Body, edgeKeys [][]byte, radius float64, feat string) (Output, bool, error) {
	if origEdges, e := resolveEdges(body, edgeKeys); e == nil {
		if res, ok := analyticCylinderFillet(body, origEdges, radius, feat); ok {
			return Output{Bodies: replaceBody(in.Bodies, body, res)}, true, nil
		}
	}
	if !ops.IsLoneCurvedAdjacentEdge(body, edgeKeys) {
		return Output{}, false, nil
	}
	res, err := ops.FilletEdges(body, edgeKeys, radius)
	if err != nil {
		return Output{}, true, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res)}, true, nil
}

// planarizedFillet re-facets a curved body for the selected edges, remapping each key to its faceted
// segment so the rolling-ball blend works instead of failing on a degenerate closed edge (#129/#127).
// A planar body — or an unresolvable key, surfaced later by the kernel — passes through unchanged.
func planarizedFillet(body *topo.Body, edgeKeys [][]byte, feat string) (*topo.Body, [][]byte) {
	origEdges, err := resolveEdges(body, edgeKeys)
	if err != nil {
		return body, edgeKeys
	}
	pb, mapped := planarizeForEdges(body, origEdges, feat)
	if pb == body {
		return body, edgeKeys
	}
	keys := make([][]byte, len(mapped))
	for i, me := range mapped {
		keys[i] = me.ReferenceKey()
	}
	return pb, keys
}

// filletBodySets rounds the definition's edge sets in one kernel pass: each constant set
// contributes its edges at one radius, each variable set one edge with a start→end radius.
// A single constant set routes through filletBody to keep the analytic cylinder-rim path.
func filletBodySets(in Input, sets []FilletEdgeSet, corner FilletCornerType, feat string) (Output, error) {
	if len(sets) == 1 && !sets[0].variable() {
		return filletBody(in, sets[0].EdgeKeys, callOrZero(sets[0].Radius), corner, feat)
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	picks, err := filletPicksOf(sets, feat)
	if err != nil {
		return Output{}, err
	}
	work := planarizeFilletPicks(body, picks, feat)
	result, err := ops.FilletEdgesCorner(work, picks, cornerStrategy(corner))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// filletPicksOf flattens the edge sets into per-edge radius picks, rejecting a variable set
// that holds more than one edge (a variable radius runs along ONE edge; tangent chains are a
// follow-up).
func filletPicksOf(sets []FilletEdgeSet, feat string) ([]ops.EdgeFilletRadii, error) {
	var out []ops.EdgeFilletRadii
	for _, s := range sets {
		if !s.variable() {
			r := callOrZero(s.Radius)
			for _, k := range s.EdgeKeys {
				out = append(out, ops.EdgeFilletRadii{Key: k, R0: r, R1: r})
			}
			continue
		}
		if len(s.EdgeKeys) != 1 {
			return nil, fmt.Errorf("%s: a variable-radius set must hold exactly 1 edge, got %d (tangent chains are a follow-up)", feat, len(s.EdgeKeys))
		}
		out = append(out, ops.EdgeFilletRadii{Key: s.EdgeKeys[0], R0: callOrZero(s.StartRadius), R1: callOrZero(s.EndRadius)})
	}
	return out, nil
}

// planarizeFilletPicks re-facets a curved body for the picks' edges (remapping each pick's
// key to the faceted segment, see planarizeForEdges); a planar body — or an unresolvable
// key, surfaced later by the kernel — passes through unchanged.
func planarizeFilletPicks(body *topo.Body, picks []ops.EdgeFilletRadii, feat string) *topo.Body {
	keys := make([][]byte, len(picks))
	for i, p := range picks {
		keys[i] = p.Key
	}
	origEdges, err := resolveEdges(body, keys)
	if err != nil {
		return body
	}
	pb, mapped := planarizeForEdges(body, origEdges, feat)
	if pb == body {
		return body
	}
	for i, me := range mapped {
		picks[i].Key = me.ReferenceKey()
	}
	return pb
}
