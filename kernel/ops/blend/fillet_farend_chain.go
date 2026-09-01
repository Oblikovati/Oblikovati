// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ROUTING the far-end multi-face trim — the ATOMIC host rebuild.
//
// THE CONTACT CHAIN, ONE SENTENCE PER FACE. Once splitTerminalSection has resolved a terminal section
// into the pieces its faces actually carry, the fillet's whole footprint on the body is a set of contact
// CHAINS, one per host face, and each host's new boundary is its ORIGINAL ring with that chain spliced in
// place of the span the fillet removed — exactly what chainRetrimLoop (f391c182) does. For complex/D8 the
// six chains are: the A tangent line on the x=247.394 wall, the B tangent line on the z=10 top, one trim
// piece on each radius-24 corner round, and one on each flat wall those rounds hand off to.
//
// A FILLET'S TWO ENDS ARE INDEPENDENT. Only the end(s) whose terminal section actually splits contribute
// a stop face and a hand-off neighbour; an end that stops cleanly on its own face contributes nothing but
// its already-settled tangent points. simple/Q5 is the one-ended case — its high-x end runs out onto a
// plane perpendicular to the band's own axis, which trimTerminalSection returns byte-for-byte, while its
// low-x end crosses between the two faces of a radius-3000 wall — so it rebuilds FOUR faces, not six.
//
// WHY IT MUST BE ATOMIC, AND WHY IT IS. The band's cap and the hosts' boundaries are the SAME curves; if
// one side adopts the split and the other does not, the shell opens along the difference — measured on
// simple/Y2, where routing the host plane alone took it 8475 → 8450 while its band still claimed the old
// span (chain-retrim-report.md §5.2). So this file decides ONCE, for the whole edge fillet, and either
// every host rebuild succeeds and edgeFillet.splitEnds is set (which is the only thing the band's own cap
// reads), or nothing is touched and the body is byte-identical to the single-face trim.
//
// WHY transformLoop CANNOT DO IT. Its ends/subs branches substitute one loop VERTEX, so a rebuilt
// boundary still passes through the same edges in the same order. D8's round must lose its whole far rim
// ARC and stop part-way down the next ruling, and D8's top plane must lose two whole rim arcs and shorten
// two straight edges. That is a chain, not a vertex (selfcross-trim-report.md §5.2).

// commitFarEndSplits accepts or rejects one body's far-end multi-face split as a whole, returning the
// rebuilt host faces keyed by their original face id (nil = declined, nothing mutated). On acceptance it
// stamps edgeFillet.splitEnds and adopts the split's own tangent points, so the band's cap and every host
// speak of the same trim.
//
// Example: complex/D8 returns six faces — the two corner rounds at their closed-form 3307.1168, the two
// flat walls each less the 1.2073 the band now bites, the top plane less the two 1.5144 corner detours it
// used to carry, and the x=247.394 wall unchanged at 32399.0345. simple/Q5 returns FOUR — the small
// radius-3000 wall piece at its closed-form 8121170.18 (it shipped 7645850.16, −5.85 %, without this),
// the big wall piece at 47038978.00, the z=6000 top plane at 49368722.08, and the y=0 plane unchanged
// at 10500000.
func commitFarEndSplits(body *topo.Body, fils []edgeFillet, taken map[uint64]filletFace, handled map[uint64]bool) map[uint64]filletFace {
	if !soleSplitFillet(fils, handled) {
		return nil
	}
	chains, ok := splitHostChains(fils[0])
	if !ok || chainsCollide(chains, taken) {
		return nil
	}
	out, ok := rebuildSplitHosts(chains, opstol.ForBody(body).Weld())
	if !ok {
		return nil
	}
	adoptSplitTangents(&fils[0])
	return out
}

// splitHostFaceCount is how many of the body's own faces one split edge fillet rebuilds: its two host
// faces, plus a stop face and a hand-off neighbour at EACH END WHOSE SECTION SPLIT. Anything else — a
// face serving two of those roles at once, a hole loop where a boundary is expected — is a configuration
// this slice has not measured, and is declined.
//
// ★ It is a per-fillet census, not the constant 6 it started as. complex/D8 splits at both ends and so
// rebuilds 6; simple/Q5 splits at ONE end — its other terminal section is a flat cap already lying on
// its stop PLANE, which trimTerminalSection returns byte-for-byte — and so rebuilds 4. Fixing the count
// at 6 was the clause that excluded Q5 (see splitEndCount).
func splitHostFaceCount(ef edgeFillet) int {
	return 2 + 2*splitEndCount(ef)
}

// splitEndCount is how many of the fillet's two terminal sections the far-end trim resolved into a
// two-piece chain.
//
// ★ WHY "AT LEAST ONE" IS THE RIGHT GATE, MEASURED. The requirement used to be that BOTH ends split, and
// that clause — not the single-fillet one — is what declined simple/Q5: an instrumented corpus-wide sweep
// of all 475 records shows exactly two cases ever produce a splitting terminal section, complex/D8 (2 and
// 2) and simple/Q5 (2 and 0), and Q5 reaches commitFarEndSplits with len(fils) == 1. An end that does NOT
// split needs no rebuild: its cap already lies on its stop face, so its tangent points are final and its
// stop face's own ring is untouched. The two halves are independent — each end contributes its own stop
// face and neighbour to the chain census, and its own two tangent points to the A/B host chains.
func splitEndCount(ef edgeFillet) int {
	n := 0
	for _, c := range []corner{ef.c0, ef.c1} {
		if len(c.endPieces) == 2 {
			n++
		}
	}
	return n
}

// soleSplitFillet reports whether the body carries exactly one constant-radius edge fillet, not already
// owned by the obstacle/runout rebuilds, with AT LEAST ONE terminal section split in two. The
// single-fillet precondition is not caution but correctness: each host is rebuilt from its ORIGINAL ring,
// which would discard a second fillet's substitutions on the same face. It excludes 27 corpus cases and
// would admit none of them — none has a splitting terminal section at all.
func soleSplitFillet(fils []edgeFillet, handled map[uint64]bool) bool {
	if len(fils) != 1 || fils[0].varying || handled[fils[0].edge.ID()] {
		return false
	}
	return splitEndCount(fils[0]) > 0
}

// splitHostChains is the fillet's contact chain on each face it touches: one trim piece per stop face and
// per hand-off neighbour, and the A/B tangent LINES between the two corners' tangent points on the two
// hosts. It declines when any face would carry two chains (splitHostFaceCount is the whole census).
func splitHostChains(ef edgeFillet) (map[*topo.Face][]endSeg, bool) {
	m := map[*topo.Face][]endSeg{}
	put := func(f *topo.Face, chain []endSeg) bool {
		if f == nil || m[f] != nil {
			return false
		}
		m[f] = chain
		return true
	}
	for _, c := range []corner{ef.c0, ef.c1} {
		for _, p := range c.endPieces {
			if !put(p.face, []endSeg{p.seg}) {
				return nil, false
			}
		}
	}
	ok := put(ef.a, []endSeg{{from: splitTangentA(ef.c0), to: splitTangentA(ef.c1)}}) &&
		put(ef.b, []endSeg{{from: splitTangentB(ef.c0), to: splitTangentB(ef.c1)}})
	return m, ok && len(m) == splitHostFaceCount(ef)
}

// splitTangentA is the corner's A-face tangent point as the SPLIT resolved it — the head of its chain —
// or, for an end that did not split, the tangent point the section solve already settled on. An unsplit
// end's cap lies on its stop face, so its ta is final and the host chain simply ends there.
func splitTangentA(c corner) math.Point3 {
	if len(c.endPieces) == 0 {
		return c.ta
	}
	return c.endPieces[0].seg.from
}

// splitTangentB is the corner's B-face tangent point as the SPLIT resolved it — the tail of its chain —
// or the settled tb for an end that did not split. It is the one that MOVES: D8's runs from
// (217.394, 35.856, 10), 0.762 inside the top face, onto that face's own y = 35.09378 boundary edge,
// which is where the solid actually ends.
func splitTangentB(c corner) math.Point3 {
	if len(c.endPieces) == 0 {
		return c.tb
	}
	return c.endPieces[len(c.endPieces)-1].seg.to
}

// chainsCollide reports whether any host the split would rebuild has already been replaced by the
// obstacle / run-out rebuild — the one way two owners could disagree about a face.
func chainsCollide(chains map[*topo.Face][]endSeg, taken map[uint64]filletFace) bool {
	for f := range chains {
		if _, dup := taken[f.ID()]; dup {
			return true
		}
	}
	return false
}

// rebuildSplitHosts re-trims every host's ORIGINAL ring against its contact chain. All or nothing: one
// decline abandons the whole split, so the caller can fall back to the single-face trim intact.
func rebuildSplitHosts(chains map[*topo.Face][]endSeg, tol float64) (map[uint64]filletFace, bool) {
	out := make(map[uint64]filletFace, len(chains))
	for f, chain := range chains {
		segs, ok := chainRetrimLoop(f.Geometry(), originalHostSegs(f), chain, tol)
		if !ok {
			return nil, false
		}
		loops := append([]filletLoop{loopFromSegs(segs)}, innerHostLoops(f)...)
		out[f.ID()] = filletFace{surface: f.Geometry(), loops: loops, parent: f.Lineage()}
	}
	return out, true
}

// The local from→to curve repair this rebuild used to carry (alignSegCurves) is GONE. It existed because
// spliceCornerBiteChain reversed a chain with a primitive that swapped a non-arc segment's endpoints and
// left its curve pointing the ORIGINAL way, so exactly one of complex/D8's two mirror rounds (whichever
// the ring orientation reversed) shipped a boundary that leapt to the far end and walked back — −6.8 % on
// the round and +31.7 % on the band. reversedEndSeg now reverses the curve with the endpoints, so the
// repair had nothing left to do: instrumented over the whole corpus it flipped 0 curves on 124 shipped
// bodies, against 2 (both D8's) with the old reversal. The invariant is now held by the primitive that
// creates the segment rather than patched by every consumer that reads one.

// adoptSplitTangents moves the fillet's own tangent points onto the split's chain extremes and arms the
// band's cap. Called only after every host rebuild has succeeded.
func adoptSplitTangents(ef *edgeFillet) {
	for _, c := range []*corner{&ef.c0, &ef.c1} {
		c.ta, c.tb = splitTangentA(*c), splitTangentB(*c)
	}
	ef.splitEnds = true
}

// splitEndSegs is the band's own terminal cap when the split committed: the same chain, ta → tb, sharing
// the very curve objects the hosts were rebuilt with, so band and host weld curve-for-curve.
func splitEndSegs(c corner) []endSeg {
	out := make([]endSeg, 0, len(c.endPieces))
	for _, p := range c.endPieces {
		out = append(out, p.seg)
	}
	return out
}
