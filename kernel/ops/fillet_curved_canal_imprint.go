// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
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
		return transformFace(f, faceFilletInputs{}), "" // untouched by the corner — carried through verbatim
	}
	segs := originalHostSegs(f)
	if len(segs) < 3 {
		return filletFace{}, fmt.Sprintf("canal imprint (%T): host loop has %d edges, need ≥3", f.Geometry(), len(segs))
	}
	for _, chain := range chains {
		spliced, ok := spliceCornerBiteChain(f.Geometry(), segs, chain, tol)
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
// object rather than reversing it through reversedEndSeg like every other reversal in this layer.
//
// ★ THIS IS THE SAME BUG CLASS reversedEndSeg fixes, and it is left standing DELIBERATELY, with its cost
// measured rather than assumed. Reversing here is a one-line change and it goes red immediately:
// TestCanalImprint_X80SpiricExtension requires the spliced loop to carry a bare geom.SpiricArc, because
// three consumers dispatch on that CONCRETE type — spiricBandMesh's twoOvalEdges (the tessellator's own
// two-oval band path), curved_stitch's newEdge and curved_halfspace_torus_uv's sectionUV. A reversedCurve3
// wrapper hides the type from all three, and the fix for that is to tessellate.Unwrap with geom.InnerCurve at each —
// a change to the TESSELLATION dispatch, not to this splice, and one that has to be measured on its own.
//
// What the exposure is, measured: the swap branch fires on ONE corpus case (simple/N7's s_5 torus arm),
// and its terminal is the SpiricArc the sampler reads directionlessly (curve3InteriorPoints orients itself
// from `from`). It is not free — a consumer that trims this segment BY PARAMETER would keep the wrong half
// — but nothing does today, and the far-end split path that does (rebuildSplitHosts) never reaches here.
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
//
// host is the surface the loop lives on: the area is measured in ITS metric, so the pick is the developed
// one on a curved host rather than the projected shadow (fillet_chain_span_area.go). Pass nil only when
// the caller genuinely has no surface — that falls back to the planar (Newell) measure.
func spliceCornerBiteChain(host geom.Surface, segs []endSeg, chain []endSeg, tol float64) ([]endSeg, bool) {
	p0, p1 := chain[0].from, chain[len(chain)-1].to
	ring := insertSplits(segs, []math.Point3{p0, p1}, tol)
	i, j := indexOfSegFrom(ring, p0, tol), indexOfSegFrom(ring, p1, tol)
	if i < 0 || j < 0 {
		return nil, false // a chain extreme does not lie on the loop — cannot splice
	}
	fwd := segsForward(ring, i, j) // p0 → p1
	bwd := segsForward(ring, j, i) // p1 → p0
	if chainBiteArea(host, fwd, chain) <= chainBiteArea(host, bwd, chain) {
		return append(bwd, chain...), true // remove the smaller fwd corner; close p0→p1
	}
	return append(fwd, reverseEndSegs(chain)...), true // remove the smaller bwd corner
}

// chainBiteArea is the area of the region a loop span encloses when closed by the bite CHAIN — the corner
// spliceCornerBiteChain removes if it drops that span. The span and the chain's bulge are sampled into a
// point ring (each curved edge — arc OR spiric — contributes its true curvature via PointAt, not a chord),
// and that ring is measured in the HOST's own metric (developedSpanArea): the Newell area-vector magnitude
// on a plane, the developed chart area on a curved host, where the Newell one is only the projected
// shadow. Generalises cornerBiteArea from a single arc bite to the multi-edge extension+terminal chain.
func chainBiteArea(host geom.Surface, span []endSeg, chain []endSeg) float64 {
	if len(span) == 0 {
		return 0
	}
	ring := segPolyline(span)                      // span[0].from … (approaching span's last vertex)
	end := span[len(span)-1].to                    // the span's far chain endpoint
	ring = append(ring, end)                       // close the span's own last vertex explicitly
	ring = append(ring, chainBulge(chain, end)...) // the chain's curvature from `end` back toward ring[0]
	return developedSpanArea(host, ring)
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
		return probe.ReversedPoints(interior) // `from` is the chain's far end — walk back toward ring[0]
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
