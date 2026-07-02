// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// geomEdgeBindTol is the model-space distance within which a geometric edge descriptor's
// midpoint must match a running-body edge's midpoint to bind. Generous enough to absorb
// unit-conversion and tessellation drift, tight enough that distinct edges stay
// unambiguous (the resolver also requires direction alignment and is unique-or-fail).
const geomEdgeBindTol = math.Scalar(1e-3)

// bindGeomEdges resolves any geometric edge descriptors against the running body and folds
// the bound edges' current lineage keys into keys, so the rest of the dress-up pipeline is
// unchanged. An external author (the NX exporter, ADR-0040) supplies these because it
// cannot mint Oblikovati lineage keys. A descriptor that does not bind is an error so the
// feature goes Sick honestly (rather than silently dropping the selection).
func bindGeomEdges(in Input, keys [][]byte, geom []topo.GeometricEdgeRef, feat string) ([][]byte, error) {
	if len(geom) == 0 {
		return keys, nil
	}
	body, err := runningBody(in)
	if err != nil {
		return nil, err
	}
	out := append([][]byte(nil), keys...)
	for _, g := range geom {
		e, ok := body.FindEdgeByGeometry(g, geomEdgeBindTol)
		if !ok {
			return nil, fmt.Errorf("%s: geometric edge reference did not bind near midpoint %v", feat, g.Midpoint)
		}
		out = append(out, e.ReferenceKey())
	}
	return out, nil
}

// bindGeomFaces is the face counterpart of [bindGeomEdges]: it resolves geometric face
// descriptors (centroid + outward normal) against the running body via
// topo.Body.FindFaceByGeometry and folds the bound faces' lineage keys into keys, so the
// shell/draft/hole pipelines are unchanged. A descriptor that binds nothing is an error.
func bindGeomFaces(in Input, keys [][]byte, geom []topo.GeometricFaceRef, feat string) ([][]byte, error) {
	if len(geom) == 0 {
		return keys, nil
	}
	body, err := runningBody(in)
	if err != nil {
		return nil, err
	}
	out := append([][]byte(nil), keys...)
	for _, g := range geom {
		f, ok := body.FindFaceByGeometry(g, geomEdgeBindTol)
		if !ok {
			return nil, fmt.Errorf("%s: geometric face reference did not bind near centroid %v", feat, g.Centroid)
		}
		out = append(out, f.ReferenceKey())
	}
	return out, nil
}

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

// concaveFill maps the public concave-edge strategy to the kernel's fill direction (zero ⇒ outward).
func concaveFill(t types.FilletConcaveStrategy) ops.ConcaveFill {
	if t == types.FilletConcaveInward {
		return ops.FillConcaveInward
	}
	return ops.FillConcaveOutward
}

// filletBody rounds the selected convex edges of the running body to the given radius via
// ops.FilletEdgesCorner (a real rolling-ball blend with cylinder faces), replacing it in the body
// list. corner selects how a 2-edge corner is treated. A lost edge, a non-convex edge, or a
// non-positive radius is an error so the feature goes Sick. See kernel/ops/fillet.go for the geometry.
// blendProfile is the cross-section shape (M36-F08) carried from the feature into the kernel picks:
// the section type (arc/G2/conic) and a conic's fullness rho. The zero value is the circular arc.
type blendProfile struct {
	cross FilletCrossSection
	rho   float64
}

func filletBody(in Input, edgeKeys [][]byte, radius float64, corner FilletCornerType, concave types.FilletConcaveStrategy, prof blendProfile, feat string, anchors map[string]math.Point3) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if radius <= 0 {
		return Output{}, fmt.Errorf("%s: radius %g must be > 0", feat, radius)
	}
	// Resolve once at the top so a lost reference can heal through the tiered binder
	// (ADR-0043 P6; anchors drive its geometric tier). The downstream kernel ops re-resolve by
	// EXACT key, so feed them the recovered edges' CURRENT keys — identical to the stored key
	// for an exact match, the live key of the recovered sibling for a healed one — and carry
	// the heals to the Output.
	edges, heals, err := resolveEdges(body, edgeKeys, anchors)
	if err != nil {
		return Output{}, err
	}
	keys0 := currentKeys(edges)
	// The analytic fast path builds exact cylinder/torus surfaces (circular arc only); a G2/conic
	// cross-section must go through the swept ruling band instead.
	if prof.cross.IsArc() {
		if out, ok, err := analyticFilletFastPath(in, body, keys0, radius, feat); ok || err != nil {
			out.Heals = heals
			return out, err
		}
	}
	return blendFilletEdges(in, body, keys0, radius, corner, concave, prof, feat, heals)
}

// blendFilletEdges runs the general rolling-ball blend for already-resolved edge keys: it
// planarizes a curved body where needed (#129), builds the per-edge radius picks, and applies
// FilletEdgesCorner, carrying any reference heals onto the result (ADR-0043 P6).
func blendFilletEdges(in Input, body *topo.Body, keys0 [][]byte, radius float64, corner FilletCornerType, concave types.FilletConcaveStrategy, prof blendProfile, feat string, heals []ReferenceHeal) (Output, error) {
	work, keys := planarizedFillet(body, keys0, feat)
	picks := make([]ops.EdgeFilletRadii, len(keys))
	cross := prof.cross
	for i, k := range keys {
		picks[i] = ops.EdgeFilletRadii{Key: k, R0: radius, R1: radius, Cross: cross, Rho: prof.rho}
	}
	result, err := ops.FilletEdgesCornerDiag(work, picks, cornerStrategy(corner), concaveFill(concave), in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result), Heals: heals}, nil
}

// analyticFilletFastPath rounds a curved fillet target directly on the ANALYTIC body, before the
// planarize step that would re-facet the cylinder and destroy the circle/arc it needs: a SIMPLE
// cylinder rim becomes a surface of revolution (#127); the cylinder/cap RIM or ARC a prior fillet
// leaves becomes a toroidal band / torus + setback caps. ok=false means no analytic case applied.
func analyticFilletFastPath(in Input, body *topo.Body, edgeKeys [][]byte, radius float64, feat string) (Output, bool, error) {
	if origEdges, _, e := resolveEdges(body, edgeKeys, nil); e == nil {
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
	origEdges, _, err := resolveEdges(body, edgeKeys, nil)
	if err != nil {
		return body, edgeKeys
	}
	if !anyEdgeNeedsPlanarize(origEdges) {
		return body, edgeKeys // #1494: every selected edge is straight between planar faces — blend on the analytic body
	}
	pb, mapped := planarizeCylinderForEdges(body, origEdges, feat)
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
func filletBodySets(in Input, sets []FilletEdgeSet, corner FilletCornerType, concave types.FilletConcaveStrategy, prof blendProfile, feat string) (Output, error) {
	if len(sets) == 1 && !sets[0].variable() {
		return filletBody(in, sets[0].EdgeKeys, callOrZero(sets[0].Radius), corner, concave, prof, feat, nil)
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	picks, err := filletPicksOf(sets, prof, feat)
	if err != nil {
		return Output{}, err
	}
	// Heal each pick's edge key before the kernel pass: substitute the recovered edge's
	// current key (exact for a clean match) so FilletEdgesCorner re-resolves it, and carry
	// the heals to the Output (ADR-0043 P6).
	heals, err := healPickKeys(body, picks)
	if err != nil {
		return Output{}, err
	}
	work := planarizeFilletPicks(body, picks, feat)
	result, err := ops.FilletEdgesCornerDiag(work, picks, cornerStrategy(corner), concaveFill(concave), in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result), Heals: heals}, nil
}

// healPickKeys resolves every pick's edge key against the body and rewrites it to the
// resolved edge's CURRENT key (so a healed reference becomes the live sibling's key the
// kernel can exact-match), returning the heals that occurred. See resolveEdges.
func healPickKeys(body *topo.Body, picks []ops.EdgeFilletRadii) ([]ReferenceHeal, error) {
	keys := make([][]byte, len(picks))
	for i, p := range picks {
		keys[i] = p.Key
	}
	edges, heals, err := resolveEdges(body, keys, nil)
	if err != nil {
		return nil, err
	}
	for i, e := range edges {
		picks[i].Key = e.ReferenceKey()
	}
	return heals, nil
}

// filletPicksOf flattens the edge sets into per-edge radius picks, rejecting a variable set
// that holds more than one edge (a variable radius runs along ONE edge; tangent chains are a
// follow-up).
func filletPicksOf(sets []FilletEdgeSet, prof blendProfile, feat string) ([]ops.EdgeFilletRadii, error) {
	var out []ops.EdgeFilletRadii
	cross := prof.cross
	for _, s := range sets {
		if !s.variable() {
			r := callOrZero(s.Radius)
			for _, k := range s.EdgeKeys {
				out = append(out, ops.EdgeFilletRadii{Key: k, R0: r, R1: r, Cross: cross, Rho: prof.rho})
			}
			continue
		}
		if len(s.EdgeKeys) != 1 {
			return nil, fmt.Errorf("%s: a variable-radius set must hold exactly 1 edge, got %d (tangent chains are a follow-up)", feat, len(s.EdgeKeys))
		}
		out = append(out, ops.EdgeFilletRadii{
			Key: s.EdgeKeys[0], R0: callOrZero(s.StartRadius), R1: callOrZero(s.EndRadius),
			Mids: midRadiiOf(s.RadiusPoints), Cross: cross, Rho: prof.rho,
		})
	}
	return out, nil
}

// midRadiiOf evaluates the feature-layer radius-point closures into kernel FilletRadiusPoints (#695).
func midRadiiOf(pts []FilletRadiusPoint) []ops.FilletRadiusPoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]ops.FilletRadiusPoint, len(pts))
	for i, p := range pts {
		out[i] = ops.FilletRadiusPoint{T: p.T, R: callOrZero(p.Radius)}
	}
	return out
}

// planarizeFilletPicks re-facets a curved body for the picks' edges (remapping each pick's
// key to the faceted segment, see planarizeForEdges); a planar body — or an unresolvable
// key, surfaced later by the kernel — passes through unchanged.
func planarizeFilletPicks(body *topo.Body, picks []ops.EdgeFilletRadii, feat string) *topo.Body {
	keys := make([][]byte, len(picks))
	for i, p := range picks {
		keys[i] = p.Key
	}
	origEdges, _, err := resolveEdges(body, keys, nil)
	if err != nil {
		return body
	}
	if !anyEdgeNeedsPlanarize(origEdges) {
		return body // #1494: every selected edge is straight between planar faces — blend on the analytic body
	}
	pb, mapped := planarizeCylinderForEdges(body, origEdges, feat)
	if pb == body {
		return body
	}
	for i, me := range mapped {
		picks[i].Key = me.ReferenceKey()
	}
	return pb
}
