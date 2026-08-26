// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// notchSeg is one point of a host-plane notch detour and the curve of the segment LEAVING it (nil ⇒
// straight). A slice of these, from one receded tangent corner up to the other, replaces the straight
// tangent segment of a host's outer loop so the boundary detours around the boss footprint. Built by
// the intact-boss re-clip (fillet_setback_close.go) and spliced in by buildHostNotch.
type notchSeg struct {
	pt    math.Point3
	curve geom.Curve3
	// srcV is the source-vertex id this detour point welds under (0 ⇒ weld by coordinate). It is
	// set only for a boss footprint SEAM point re-inserted by the intact-boss re-clip: that seam is
	// shared with the kept-intact boss wall, whose transformFace carries it under its real vertex id,
	// so the notch must weld the same point under the SAME id or addID's distinct-id rule (#1600)
	// keeps them separate and the wall rim never welds to the host notch (S4's cone/cyl runout left
	// 4 open edges at the inner-boss seam before this).
	srcV uint64
}

// hostTangent is the fillet's tangent point on a host face at corner c: ta when the host is the fillet's
// a face (hostIsA), tb otherwise — the receded tangent-line endpoints the host notch re-cuts between.
func hostTangent(c corner, hostIsA bool) math.Point3 {
	if hostIsA {
		return c.ta
	}
	return c.tb
}

// appendTrimmedArcSegs appends a detour curve's open samples (excluding the far endpoint) at chord
// count n, each sample carrying its own sub-span of the curve (geom.TrimmedCurve3) as the segment's
// leaving curve. The samples are the same points the neighbour (wall rim/patch) tiles from the
// identical curve AT THE SAME COUNT (sampleCurveNTrimmed's pts are byte-identical to sampleCurveN's),
// so the faces weld; n lets a torus host arc densify (rimSubArcChordCount) while lines/ruled arcs stay
// at ringSegSamples. For a boss's host-side FOOTPRINT arc the carried sub-span makes the notch bound
// the TRUE rim instead of its inscribed chords — the chords left the re-clipped host plane the whole
// inscribed-polygon surplus Σ (r²/2)(θ−sinθ) (T3's plane +4.38498, t3-plane-sliver-report.md). The
// per-segment restriction (never the full curve on a sub-edge) is the same N7 rule
// sampleCurve3OpenTrimmed states — a full curve per sub-edge would make sampleEdgeCurve re-trace it
// once per segment and self-cross the boundary. Contact LOCI ride the same rule since the railB
// interpolated-locus carry (railb-locus-report.md): their segments carry sub-spans of the locus rail —
// the very TrimmedCurve3 values the patch loop offers — where the old degree-1 locus (whose straight
// sub-spans made a nil chord equivalent) sat a sagitta off the patch.
func appendTrimmedArcSegs(segs []notchSeg, arc geom.Curve3, n int) []notchSeg {
	pts, curves := sampleCurveNTrimmed(arc, n, false)
	for i, p := range pts {
		segs = append(segs, notchSeg{pt: p, curve: curves[i]})
	}
	return segs
}

// buildHostNotch rebuilds a host plane by transformFace (its receded outer loop, identical to the non-
// obstacle path) and then replaces the straight receded-tangent segment (the one between the two host
// tangent points tanA,tanB) with the detour, dropping every inner loop (the boss footprint hole is now
// merged into the outer boundary). ok=false honest-rejects a host whose tangent segment is not found.
func buildHostNotch(host *topo.Face, maps filletRebuildMaps, tanA, tanB math.Point3,
	detour func(from, to math.Point3) ([]notchSeg, bool)) (filletFace, bool) {
	base := transformFace(host, maps.forFace(host, 0))
	oi, ok := outerLoopIndex(host)
	if !ok {
		return filletFace{}, false
	}
	weld := ResolutionForPoints(base.loops[oi].pts).Weld()
	i, ok := tangentSegmentIndex(base.loops[oi], tanA, tanB, weld)
	if !ok {
		return filletFace{}, false
	}
	segs, ok := detour(base.loops[oi].pts[i], base.loops[oi].pts[(i+1)%len(base.loops[oi].pts)])
	if !ok {
		return filletFace{}, false
	}
	notch := spliceNotch(base.loops[oi], i, segs)
	return filletFace{surface: base.surface, loops: []filletLoop{notch}, parent: base.parent}, true
}

// tangentSegmentIndex finds the loop segment index k whose endpoints (pts[k], pts[k+1]) are the two
// host tangent points (unordered) within weld — the receded filleted edge the notch detour replaces.
func tangentSegmentIndex(l filletLoop, tanA, tanB math.Point3, weld float64) (int, bool) {
	n := len(l.pts)
	for k := range n {
		p, q := l.pts[k], l.pts[(k+1)%n]
		if matchPair(p, q, tanA, tanB, weld) {
			return k, true
		}
	}
	return 0, false
}

// matchPair reports whether {p,q} equals {a,b} (either order) within weld.
func matchPair(p, q, a, b math.Point3, weld float64) bool {
	return (p.DistanceTo(a) < weld && q.DistanceTo(b) < weld) ||
		(p.DistanceTo(b) < weld && q.DistanceTo(a) < weld)
}

// spliceNotch rebuilds loop l with segment i (pts[i]→pts[i+1]) replaced by the detour segs (which start
// at pts[i] and end just before pts[i+1]); every other point keeps its source identity. The dropped
// inner boss loop is not copied — the caller emits only this notched outer loop.
func spliceNotch(l filletLoop, i int, segs []notchSeg) filletLoop {
	var out filletLoop
	for k := 0; k < len(l.pts); k++ {
		if k == i {
			for _, s := range segs {
				out.addID(s.pt, s.curve, s.srcV, 0) // srcV pins a shared boss seam to the intact wall's vertex
			}
			continue
		}
		out.addID(l.pts[k], l.curves[k], idAt(l.srcV, k), idAt(l.srcE, k))
	}
	return out
}

// idAt returns ids[k] or 0 when the slice is shorter (an op-generated point carries no source id).
func idAt(ids []uint64, k int) uint64 {
	if k < len(ids) {
		return ids[k]
	}
	return 0
}
