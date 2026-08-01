// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-miter RESULT-FACE assembly — split out of fillet_miter_curved_weld.go (which owns the arm-
// side solve + the shared-face two-rail retrim) so each file stays under the 500-line/one-responsibility
// rule. This file turns two solved miterArmSides into the full result face list: the two trimmed arm
// faces, every bitten host's retrim (accumulated per-host as bite CHAINS — fillet_miter_chain_retrim.go
// owns the chain splice primitives), and every untouched face carried through verbatim — guarded against
// certifying over a contact host whose OWN boundary is already degenerate (simple/P4, P5's pre-existing
// STEP-import defect; see wave-report-C.md).

// curvedMiterFaces builds every result face: the two trimmed arm faces, the twice-bitten shared face,
// each arm's receded outer host and far cap, and every untouched face carried through verbatim. snapTol
// (res.Sew(), the project's own established "close a gap between independently-derived geometry"
// tolerance) is passed to sharedMiterRetrim SEPARATELY from tol: it governs only whether an arm's
// analytically-derived far foot is close enough to the shared face's ORIGINAL loop to be treated as
// already on it, a coarser question than the weld-scale coincidence tol answers elsewhere (see
// sharedRailAnchor).
func curvedMiterFaces(body *topo.Body, m *cornerMiter, tor, cyl miterArmSide, r float64, res Resolution) ([]filletFace, string) {
	tol, snapTol := res.Weld()*r, res.Sew()
	shared, ok := sharedMiterRetrim(m, tor, cyl, tol, snapTol)
	if !ok {
		return nil, "curved miter: shared-face two-rail retrim declined"
	}
	if !sharedRetrimIsSound(shared, tol) {
		return nil, fmt.Sprintf("curved miter: shared host %T's own boundary is degenerate (pre-existing defect, not a fillet gap)", m.shared.Geometry())
	}
	bites, ok := miterHostBiteChains(m, tor, cyl, tol, snapTol)
	if !ok {
		return nil, "curved miter: a host bite chain could not be assembled"
	}
	out := []filletFace{miterArmFace(tor, m.seam), miterArmFace(cyl, m.seam), shared}
	rest, reason := curvedMiterHostFaces(body, m, tor, cyl, bites, tol, snapTol)
	if reason != "" {
		return nil, reason
	}
	return append(out, rest...), ""
}

// curvedMiterHostFaces builds every OTHER result face: each bitten host's retrim, and every untouched
// face carried through verbatim (guarded — passthroughGuardHosts — against certifying over a contact
// host whose own boundary is already degenerate). Split from curvedMiterFaces to stay within funlen.
func curvedMiterHostFaces(body *topo.Body, m *cornerMiter, tor, cyl miterArmSide, bites map[*topo.Face][]miterHostChain, tol, snapTol float64) ([]filletFace, string) {
	guardHosts := passthroughGuardHosts(tor, cyl)
	var out []filletFace
	for _, f := range body.Faces() {
		if f == m.shared {
			continue // replaced by the twice-bitten retrim
		}
		chains, bitten := bites[f]
		if !bitten {
			if containsFace(guardHosts, f) && !passthroughHostIsSound(f, tol) {
				return nil, fmt.Sprintf("curved miter: contact host %T left untouched but its own boundary is degenerate (pre-existing defect, not a fillet gap)", f.Geometry())
			}
			out = append(out, passthroughFace(f))
			continue
		}
		ff, ok := miterHostRetrim(f, chains, tol, snapTol)
		if !ok {
			return nil, fmt.Sprintf("curved miter: host %T retrim declined", f.Geometry())
		}
		out = append(out, ff)
	}
	return out, ""
}

// passthroughGuardHosts is the miter corner's own candidate contact hosts: each arm's outer face and
// far capping face (nil when the runout has none) — the faces this corner's rails and trims are built
// against. A corner whose bite-chain map happens to leave one of THESE untouched still owes it a
// soundness check (passthroughHostIsSound) before trusting it as-is; every other face of the body is
// out of scope (checking the whole body would false-decline on garbage far from this corner).
func passthroughGuardHosts(tor, cyl miterArmSide) []*topo.Face {
	hosts := []*topo.Face{tor.outer, cyl.outer}
	if tor.run.capping != nil {
		hosts = append(hosts, tor.run.capping)
	}
	if cyl.run.capping != nil {
		hosts = append(hosts, cyl.run.capping)
	}
	return hosts
}

// containsFace reports whether f is one of hosts.
func containsFace(hosts []*topo.Face, f *topo.Face) bool {
	for _, h := range hosts {
		if h == f {
			return true
		}
	}
	return false
}

// passthroughHostIsSound reports whether a candidate contact host's OWN (untouched) loops are free of
// the self-touching/degenerate defect ringHasRepeatedPoint detects (fillet_miter_chain_retrim.go).
// simple/P5's wall stays passthrough for this arm pairing yet its raw STEP-imported loop already
// carries a zero-length edge (100,50,0)->(100,50,0) bracketed by one pre-existing edge used twice —
// present verbatim before any fillet runs, so it is an import defect, not a retrim bug. Certifying a
// miter against an unsound host silently ships a ~10% short face and a non-watertight weld; declining
// is the do-no-harm floor.
func passthroughHostIsSound(f *topo.Face, tol float64) bool {
	for _, l := range f.Loops() {
		if ringHasRepeatedPoint(segsFromLoop(l), tol) {
			return false
		}
	}
	return true
}

// sharedRetrimIsSound guards the shared-face two-rail retrim result against the SAME self-touching
// defect pointsHaveDuplicate catches on the host-chain retrim layer (fillet_miter_chain_retrim.go):
// the shared host's own bitten loop must not revisit a point untouched by this corner's rails.
// simple/P5's wall (m.shared) is the proven case — its raw STEP-imported loop already carries a
// zero-length edge (100,50,0)->(100,50,0) bracketed by one pre-existing edge used twice, present
// verbatim before any fillet runs (confirmed by tessellating the RAW pre-fillet face alone: it comes
// out short of its own analytic wall area). sharedMiterRetrim only touches the small corner near the
// vertex, so the far-away defect survives untouched and later corrupts the tessellated result; the
// two rails never come near it, so there is nothing for THIS corner to repair. Declining is the
// do-no-harm floor — the same one chainedHostRetrim already applies to a bitten host.
func sharedRetrimIsSound(shared filletFace, tol float64) bool {
	for _, l := range shared.loops {
		if pointsHaveDuplicate(l.pts, tol) {
			return false
		}
	}
	return true
}

// miterHostRetrim re-clips one bitten host. A single one-segment chain whose feet both lie on the
// host's bitten loop (at TIGHT tol) keeps the pre-chain splice VERBATIM (singleRunoutHostFace — every
// prior curved-miter green is byte-identical through it); everything else — a transit-extended,
// bridged, or double-bitten host, AND a single-segment chain whose feet miss the ring by more than tol
// but land within the model's own construction gap (snapTol; W3/W4's torus-arm far-cap chain) — goes
// through the sequential chain splice (chainedHostRetrim), which carries snapTol to its own landing test.
func miterHostRetrim(f *topo.Face, chains []miterHostChain, tol, snapTol float64) (filletFace, bool) {
	if len(chains) == 1 && len(chains[0].chain) == 1 && legacyBiteOnLoop(f, chains[0], tol) {
		return singleRunoutHostFace(f, chains[0].chain[0], chains[0].consumed, tol)
	}
	return chainedHostRetrim(f, chains, tol, snapTol)
}

// legacyBiteOnLoop reports whether a single-segment bite has both feet on the loop the old splice
// would pick — the exact precondition of the byte-locked singleRunoutHostFace path.
func legacyBiteOnLoop(f *topo.Face, c miterHostChain, tol float64) bool {
	bitten := hostBittenLoop(f, c.consumed, tol)
	if bitten == nil {
		return false
	}
	segs := segsFromLoop(bitten)
	return pointOnRing(segs, c.chain[0].from, tol) && pointOnRing(segs, c.chain[0].to, tol)
}

// miterHostBiteChains maps each arm's outer host and far cap to its bite CHAINS. Bites ACCUMULATE
// per face (P5's pocket wall carries the cylinder arm's rail AND the torus arm's far trim — the
// old single-bite map silently overwrote one). The outer-host chain absorbs the sibling arm's
// transit arc when the seam's sBot lives on this host; a far trim whose shared-side foot is a
// mid-face fresh cut is bridged back to the picked edge's far vertex along shared ∩ capping.
func miterHostBiteChains(m *cornerMiter, tor, cyl miterArmSide, tol, snapTol float64) (map[*topo.Face][]miterHostChain, bool) {
	v := m.vertex.Point()
	bites := map[*topo.Face][]miterHostChain{}
	for _, pair := range [2][2]miterArmSide{{tor, cyl}, {cyl, tor}} {
		s, sibling := pair[0], pair[1]
		chain := []endSeg{reverseEndSegs([]endSeg{s.railOut})[0]} // outer host recedes along sBot/junction→farOuter
		if sibling.transit != nil {
			chain = append([]endSeg{*sibling.transit}, chain...) // junction→sBot→farOuter, all on this host
		}
		bites[s.outer] = append(bites[s.outer], miterHostChain{chain: chain, consumed: v, hasConsumed: true})
		if s.run.capping == nil {
			continue
		}
		trimChain, ok := miterTrimChain(m, s, tol, snapTol)
		if !ok {
			return nil, false
		}
		bites[s.run.capping] = append(bites[s.run.capping], trimChain)
	}
	return bites, true
}

// miterTrimChain builds one arm's far-cap bite chain: the far cross-section trim, preceded by a
// bridge CONFINED TO THE CAPPING FACE'S OWN SURFACE when the trim's shared-side foot is a mid-face
// fresh cut on the capping face (P4/P5's pocket floor: the cut the fillet itself makes at
// (48.148, 0.034, 60) is not on any original floor edge, so the floor grows to it through the far
// vertex). The bridge is built against s.run.capping — the face this chain retrims — never m.shared:
// simple/W3, W4's boss-notch corner is the first case where the two differ (the trim's off-loop foot
// sits on the capping PLANE, nowhere near the shared CYLINDER, so a shared-cylinder latitude arc
// through it does not exist and always declined; the fresh cut IS confined to the capping face by
// construction, so miterChainEndBridgeSeg's dispatch on the capping face's own geometry always has an
// answer there). The consumed marker is the far vertex only when it is NOT the bridge anchor itself.
// fromOn/toOn use snapTol: the SAME "already on the original loop, within the model's own construction
// gap" test sharedRailAnchor runs on the shared-face side of the identical arm (W3/W4's torus foot
// sits ~2e-5 off both the shared face's AND the capping face's original loop — one root, two sides).
// miterChainEndBridgeSeg keeps tol for its own bridge-curve exactness certificate.
func miterTrimChain(m *cornerMiter, s miterArmSide, tol, snapTol float64) (miterHostChain, bool) {
	far := farVertexNotVid(s.edge, m.vertex.ID())
	fromOn := miterCapPointOnLoops(s.run.capping, s.run.trim.from, snapTol)
	toOn := miterCapPointOnLoops(s.run.capping, s.run.trim.to, snapTol)
	if fromOn && toOn {
		return miterHostChain{chain: []endSeg{s.run.trim}, consumed: far, hasConsumed: true}, true
	}
	if fromOn == toOn {
		return miterHostChain{}, false // both feet mid-face — outside the single-bridge scope
	}
	trim, off := s.run.trim, s.run.trim.from
	if !toOn {
		trim, off = reverseEndSegs([]endSeg{s.run.trim})[0], s.run.trim.to
	}
	bridge, ok := miterChainEndBridgeSeg(s.run.capping.Geometry(), off, far, tol)
	if !ok {
		return miterHostChain{}, false
	}
	return miterHostChain{chain: []endSeg{bridge, trim}, hasConsumed: false}, true
}

// miterCapPointOnLoops reports whether p lies on ANY loop of the capping face.
func miterCapPointOnLoops(capping *topo.Face, p math.Point3, tol float64) bool {
	for _, l := range capping.Loops() {
		if pointOnRing(segsFromLoop(l), p, tol) {
			return true
		}
	}
	return false
}

// miterArmFace emits one trimmed arm face: the shared-face rail (farShared→sTop), the seam
// (sTop→sBot, the chord list SHARED with the other arm so the weld is watertight), the transit
// reversed (sBot→junction, only when sBot sits on the OTHER arm's outer plane), the outer-face
// rail reversed (sBot/junction→farOuter), and the far cross-section trim reversed (→farShared).
func miterArmFace(side miterArmSide, seam []math.Point3) filletFace {
	segs := []endSeg{side.railSh}
	segs = append(segs, seamEndSegs(seam)...)
	if side.transit != nil {
		segs = append(segs, reverseEndSegs([]endSeg{*side.transit})...)
	}
	segs = append(segs, reverseEndSegs([]endSeg{side.railOut})...)
	segs = append(segs, reverseEndSegs([]endSeg{side.run.trim})...)
	return filletFace{surface: side.surface, loops: []filletLoop{loopFromSegs(segs)}, parent: filletEdgeProvenance(side.edge)}
}
