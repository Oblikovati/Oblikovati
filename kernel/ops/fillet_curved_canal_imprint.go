// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The canal FAR-END host imprints (M6' C4 W3c/F2, derivation: .superpowers/sdd/canal-far-runout-
// derivation.md §5). Each arm's terminal section bites its terminating face F_far (result_10 y=30,
// result_4 z=80, result_2 x=80). When both feet land on F_far's original loop it is the verbatim B3 splice
// (y=30); when the wall-side foot ran off the loop the bite is a CHAIN — the through-vertex extension edge
// (q→off-loop foot) followed by the terminal (foot→trim foot) — so the chain's two extreme endpoints both
// land on the loop. result_2's terminal is the SPIRIC section; cornerBiteArea would chord it, so the chain
// area sampler develops any curve (arc/spiric) via its PointAt, keeping the smaller-area splice pick
// principled. This is the HOST side of F1's arm termini; the single-ball far-runout splice is untouched.

// canalImprintFace retrims one FAR-END host imprint face (derivation §5): every arm whose terminal section
// bites f splices its bite CHAIN into f's outer loop — the terminal alone (both feet on-loop, verbatim),
// or the extension edge + terminal when the wall-side foot runs off the loop. A face no terminal bites
// passes through transformFace verbatim (caps, untouched notch faces). Declines with a diagnostic reason
// (carrying the face + chain length) when a bite chain cannot splice.
func canalImprintFace(f *topo.Face, bundles []canalArmBundle, tol float64) (filletFace, string) {
	chains := imprintBiteChains(f, bundles, tol)
	if len(chains) == 0 {
		return transformFace(f, nil, nil, nil, nil, 0), "" // untouched by the corner — carried through verbatim
	}
	segs := originalHostSegs(f)
	if len(segs) < 3 {
		return filletFace{}, fmt.Sprintf("canal imprint (%T): host loop has %d edges, need ≥3", f.Geometry(), len(segs))
	}
	for _, chain := range chains {
		spliced, ok := spliceCornerBiteChain(segs, chain, tol)
		if !ok {
			return filletFace{}, fmt.Sprintf("canal imprint (%T): bite chain (%d segs) will not splice onto the loop (tol %.3e)", f.Geometry(), len(chain), tol)
		}
		segs = spliced
	}
	loops := append([]filletLoop{loopFromSegs(segs)}, innerHostLoops(f)...)
	return filletFace{surface: f.Geometry(), loops: loops, parent: f.Lineage()}, ""
}

// imprintBiteChains collects the bite chains of every arm whose terminal section runs out onto f — its two
// feet both on f's surface (the same both-endpoints-on-surface test as farArcsBiting). Each is the arm's
// terminal, prefixed with its extension edge when that extension lands on f (derivation §5).
func imprintBiteChains(f *topo.Face, bundles []canalArmBundle, tol float64) [][]endSeg {
	surf := f.Geometry()
	var chains [][]endSeg
	for _, b := range bundles {
		if !onHostSurface(surf, b.far.from, tol) || !onHostSurface(surf, b.far.to, tol) {
			continue
		}
		chains = append(chains, imprintChain(b, tol))
	}
	return chains
}

// imprintChain assembles one arm's bite chain: the terminal alone when the arm carries no extension (both
// feet on the F_far loop — the y=30 verbatim splice), else the extension edge (q→off-loop foot) followed
// by the terminal oriented to START at that off-loop foot, so the chain runs q→foot→trim-foot with both
// extremes (q and the trim foot) on the loop.
func imprintChain(b canalArmBundle, tol float64) []endSeg {
	if b.extHost == nil {
		return []endSeg{b.far}
	}
	foot := b.ext.to // the off-loop (wall-side) foot, shared with the terminal's matching end
	return []endSeg{b.ext, orientedTerminal(b.far, foot, tol)}
}

// orientedTerminal returns the terminal endSeg oriented to START at startAt, KEEPING its concrete curve
// object (never wrapping it in a reversedCurve3, which would erase a SpiricArc's type and defeat the
// chain area sampler's sample-not-chord). Watertightness does not need the reversal baked into the curve:
// the shared far edge's curve comes from the arm face (welded first, ADR-C4-2), and the area sampler reads
// PointAt directionlessly — so swapping the endpoints while threading the SAME object suffices (§7.1).
func orientedTerminal(far endSeg, startAt math.Point3, tol float64) endSeg {
	if float64(far.from.DistanceTo(startAt)) <= tol {
		return far
	}
	return endSeg{from: far.to, to: far.from, curve: far.curve, mid: far.mid, arc: far.arc}
}

// spliceCornerBiteChain replaces the loop corner a bite CHAIN spans with the chain itself — the multi-edge
// generalisation of spliceCornerBite (kept byte-locked for the single-ball path). It splits the loop at the
// chain's two extreme endpoints, removes the SMALLER-AREA span between them (the bitten corner), and closes
// the kept span with the chain, oriented to keep the ring continuous. Enclosed AREA (chainBiteArea) — not
// segment count — is the criterion, so it picks the small corner independent of edge subdivision; a wrong
// pick cracks the shell to the do-no-harm floor, never a silent wrong solid.
func spliceCornerBiteChain(segs []endSeg, chain []endSeg, tol float64) ([]endSeg, bool) {
	p0, p1 := chain[0].from, chain[len(chain)-1].to
	ring := insertSplits(segs, []math.Point3{p0, p1}, tol)
	i, j := indexOfSegFrom(ring, p0, tol), indexOfSegFrom(ring, p1, tol)
	if i < 0 || j < 0 {
		return nil, false // a chain extreme does not lie on the loop — cannot splice
	}
	fwd := segsForward(ring, i, j) // p0 → p1
	bwd := segsForward(ring, j, i) // p1 → p0
	if chainBiteArea(fwd, chain) <= chainBiteArea(bwd, chain) {
		return append(bwd, chain...), true // remove the smaller fwd corner; close p0→p1
	}
	return append(fwd, reverseChainSegs(chain)...), true // remove the smaller bwd corner
}

// reverseChainSegs reverses a chain of segments (order + each seg, curve-preserving via reverseChainSeg),
// so a p0→p1 bite chain can close a p1→p0 span.
func reverseChainSegs(chain []endSeg) []endSeg {
	out := make([]endSeg, len(chain))
	for i, s := range chain {
		out[len(chain)-1-i] = reverseChainSeg(s)
	}
	return out
}

// chainBiteArea is the area of the region a loop span encloses when closed by the bite CHAIN — the corner
// spliceCornerBiteChain removes if it drops that span. The span and the chain's bulge are sampled into a
// point ring (each curved edge — arc OR spiric — contributes its true curvature via PointAt, not a chord),
// and the area is the Newell area-vector magnitude, exact on the planar imprint host. Generalises
// cornerBiteArea from a single arc bite to the multi-edge extension+terminal chain.
func chainBiteArea(span []endSeg, chain []endSeg) float64 {
	if len(span) == 0 {
		return 0
	}
	ring := segPolyline(span)                      // span[0].from … (approaching span's last vertex)
	end := span[len(span)-1].to                    // the span's far chain endpoint
	ring = append(ring, end)                       // close the span's own last vertex explicitly
	ring = append(ring, chainBulge(chain, end)...) // the chain's curvature from `end` back toward ring[0]
	return float64(newellNormal(ring).Length()) / 2
}

// chainBulge samples the bite chain's interior + junction points, ordered from `from` (one chain extreme)
// toward the other, so they fill the chain's shape between the span's two ends when appended to the area
// ring. Each seg is developed by curveInterior (arc/spiric via PointAt), and the interior junctions are
// kept; the two extreme endpoints are already the ring's `from`/`ring[0]`, so they are stripped.
func chainBulge(chain []endSeg, from math.Point3) []math.Point3 {
	poly := chainPolyline(chain)
	if len(poly) <= 2 {
		return nil
	}
	interior := poly[1 : len(poly)-1]
	if float64(from.DistanceTo(poly[len(poly)-1])) <= float64(from.DistanceTo(poly[0])) {
		return reversedPoints(interior) // `from` is the chain's far end — walk back toward ring[0]
	}
	return interior
}

// chainPolyline develops a chain (each seg's to is the next seg's from) into a point list from the chain's
// first `from` to its last `to`, sampling every curved seg's interior via curveInterior — so a spiric or
// arc segment contributes its true shape, and the internal junctions between segments are preserved.
func chainPolyline(chain []endSeg) []math.Point3 {
	pts := []math.Point3{chain[0].from}
	for _, s := range chain {
		pts = append(pts, curveInterior(s)...)
		pts = append(pts, s.to)
	}
	return pts
}

// curveInterior samples a seg's interior points (endpoints excluded), oriented from→to: arcInteriorPoints
// for a circular arc (byte-identical to the single-ball area sampler), and a generic PointAt sweep for any
// other curve (the spiric terminal). A straight seg contributes no interior.
func curveInterior(s endSeg) []math.Point3 {
	if arc, ok := s.curve.(geom.Arc3d); ok && s.arc {
		return arcInteriorPoints(arc, s.from, biteArcSamples)
	}
	if s.curve != nil {
		return curve3InteriorPoints(s.curve, s.from, biteArcSamples)
	}
	return nil
}

// curve3InteriorPoints samples a generic Curve3's interior (endpoints excluded), oriented so the samples
// start at the curve end nearest `from` — the direction-robust analogue of arcInteriorPoints for a
// non-analytic section curve (the spiric), which has no by-three-points reconstruction.
func curve3InteriorPoints(c geom.Curve3, from math.Point3, k int) []math.Point3 {
	lo, hi := c.Domain()
	fwd := float64(c.PointAt(lo).DistanceTo(from)) <= float64(c.PointAt(hi).DistanceTo(from))
	pts := make([]math.Point3, 0, k-1)
	for i := 1; i < k; i++ {
		t := float64(i) / float64(k)
		if !fwd {
			t = 1 - t
		}
		pts = append(pts, c.PointAt(lo+t*(hi-lo)))
	}
	return pts
}

// reversedPoints returns pts in reverse order (a fresh slice).
func reversedPoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
