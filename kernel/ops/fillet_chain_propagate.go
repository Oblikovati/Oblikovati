// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Pick propagation (OCCT ChFi3d_Builder::PerformElement parity). OCCT's `blend` never fillets a
// picked edge alone: the pick SEEDS a spine that PerformElement extends over the whole
// tangent-continuous run (FaceTangency + the no-turn-back guard, ChFi3d_Builder_1.cxx), and the
// blend is built — and its area scored — over that whole spine. Filleting only the picked edge
// therefore compares two different solids (complex/D8's 18-face DRAWEXE result vs an 11-face
// single-edge blend, the ratchet TestEveryPickBlendsItsWholeTangentChain's exemplar). Every pick
// entering the fillet dispatch is expanded here first, so downstream capability gaps decline over
// the WIDENED chain — the honest geometry — rather than building a solid OCCT never builds.

// expandPicksAlongTangentSpines returns picks with every constant-radius plain-arc pick expanded
// over its full tangent-continuous spine (TangentEdgeChain — the PerformElement walk); chain
// members not already picked join at the seed's radius. It expands nothing — returning picks
// unchanged — when any pick is variable/profiled (a chain-wide law needs the Task-12 arc-length
// mapping) or when two picks' chains would meet at different radii (OCCT treats radii on one spine
// as a law along it, which is the same follow-up).
func expandPicksAlongTangentSpines(body *topo.Body, picks []EdgeFilletRadii) []EdgeFilletRadii {
	if anyProfiledPick(picks) {
		return picks
	}
	radiusByEdge, order, ok := chainRadiusUnion(body, picks)
	if !ok {
		return picks
	}
	if len(order) == len(picks) {
		return picks // every chain is already fully picked — nothing to widen
	}
	picked := pickIndexByKey(picks)
	out := make([]EdgeFilletRadii, 0, len(order))
	for _, key := range order {
		if i, dup := picked[key]; dup {
			out = append(out, picks[i])
			continue
		}
		out = append(out, EdgeFilletRadii{Key: []byte(key),
			R0: radiusByEdge[key], R1: radiusByEdge[key], Cross: picks[0].Cross, Rho: picks[0].Rho})
	}
	return out
}

// anyProfiledPick reports a pick propagation cannot widen: per-end taper, intermediate radius
// points, or a non-arc cross-section.
func anyProfiledPick(picks []EdgeFilletRadii) bool {
	for _, p := range picks {
		if p.R0 != p.R1 || len(p.Mids) > 0 || !p.Cross.IsArc() {
			return true
		}
	}
	return false
}

// chainRadiusUnion walks every pick's spine and unions the members, each carrying its seed's
// radius, in first-seen spine order. ok=false when a chain member is claimed at two different
// radii or a seed key cannot be resolved — propagation then stands down entirely.
func chainRadiusUnion(body *topo.Body, picks []EdgeFilletRadii) (map[string]float64, []string, bool) {
	radius := make(map[string]float64, len(picks))
	var order []string
	for _, p := range picks {
		keys, _, err := TangentEdgeChain(body, p.Key, DefaultTangentChainAngle)
		if err != nil {
			return nil, nil, false
		}
		for _, k := range dropSeedCircleSiblings(body, p.Key, keys) {
			ks := string(k)
			if r, seen := radius[ks]; seen {
				if r != p.R0 {
					return nil, nil, false // one spine, two radii — a radius law, not a constant chain
				}
				continue
			}
			radius[ks] = p.R0
			order = append(order, ks)
		}
	}
	return radius, order, true
}

// dropSeedCircleSiblings removes spine members lying on the SAME CIRCLE as their seed — the arcs a
// circular rim was split into at periodic-wall seams or feature junctions. The rim machinery treats
// a rim pick as the WHOLE rim already (simple/N4's green is acquitted face-for-face against
// DRAWEXE's own 14-face result, tangent_chain_debt_test.go — its unpicked same-circle sibling is
// blended by the build of the pick itself), so re-picking a sibling only double-arms the corner
// solver the picks feed. Members on OTHER circles (a rounded-rectangle rim's corner arcs) and
// siblings that are THEMSELVES picked (simple/Y9's three-arc rim, which routes through the stripe)
// are untouched — this drops only what the seed's own whole-rim convention already covers.
func dropSeedCircleSiblings(body *topo.Body, seedKey []byte, keys [][]byte) [][]byte {
	seed, okS := body.FindEdgeByKey(seedKey)
	if !okS {
		return keys
	}
	tol := float64(ResolutionForBody(body).Weld())
	out := make([][]byte, 0, len(keys))
	for _, k := range keys {
		e, ok := body.FindEdgeByKey(k)
		if ok && e != seed && arcsOfOneCircle(seed.Geometry(), e.Geometry(), tol) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// arcsOfOneCircle reports whether two curves are circular arcs of the same circle (equal centre,
// radius, and axis line within tol).
func arcsOfOneCircle(ca, cb geom.Curve3, tol float64) bool {
	aC, aOK := circleOfCurve(ca)
	bC, bOK := circleOfCurve(cb)
	if !aOK || !bOK {
		return false
	}
	sameCentre := float64(aC.Center.DistanceTo(bC.Center)) <= tol
	sameR := stdmath.Abs(aC.Radius-bC.Radius) <= tol
	axisDot := stdmath.Abs(float64(aC.Normal.AsVector().Dot(bC.Normal.AsVector())))
	return sameCentre && sameR && axisDot >= 1-1e-9
}

// circleOfCurve extracts the underlying circle of a circular curve (a full circle or an arc).
func circleOfCurve(c geom.Curve3) (geom.Circle, bool) {
	switch g := c.(type) {
	case geom.Circle:
		return g, true
	case geom.Arc3d:
		return geom.Circle{Center: g.Center, Normal: g.Normal, RefDir: g.RefDir, Radius: g.Radius}, true
	}
	return geom.Circle{}, false
}

// pickIndexByKey indexes the original picks by edge key so expansion preserves them verbatim.
func pickIndexByKey(picks []EdgeFilletRadii) map[string]int {
	out := make(map[string]int, len(picks))
	for i, p := range picks {
		out[string(p.Key)] = i
	}
	return out
}
