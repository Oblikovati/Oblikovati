// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// notchSeg is one point of a host-plane notch detour and the curve of the segment LEAVING it (nil ⇒
// straight). A slice of these, from one receded tangent corner up to the other, replaces the straight
// tangent segment of a host's outer loop so the boundary detours around the boss footprint (Task 10b).
type notchSeg struct {
	pt    math.Point3
	curve geom.Curve3
}

// buildRunoutHostsAndWalls reconstructs the two host planes (their boss footprint re-cut so it no
// longer protrudes) and splits the two boss walls (their closed rim cut so each sub-arc welds to a
// neighbour), folding the four replacement faces (plus host-A's flat-fill triangles) into set by
// body-face ID. ok=false honest-rejects a malformed re-cut so the caller drops the whole edge.
func buildRunoutHostsAndWalls(set *runoutSet, ef edgeFillet, tl runoutTiling, maps filletRebuildMaps) bool {
	if !buildHostBAndWall(set, ef, tl, maps) {
		return false
	}
	if !buildHostAAndWall(set, ef, tl, maps) {
		return false
	}
	return true
}

// buildHostBAndWall reconstructs host plane B (the fillet-cutting r8 boss, a clean notch: no flat fill)
// and splits its boss wall. The notch replaces the tangent segment with the boss host-side arc split at
// the wall's seam point (fbL→seam major, seam→fbR minor), and drops the boss footprint hole.
func buildHostBAndWall(set *runoutSet, ef edgeFillet, tl runoutTiling, maps filletRebuildMaps) bool {
	im := tl.bImp
	wall := otherFace(im.footprintEdge, im.host)
	if wall == nil {
		return false
	}
	seam := im.footprintEdge.StartVertex().Point()
	tanA, tanB := hostTangent(ef.c0, im.hostIsA), hostTangent(ef.c1, im.hostIsA)
	notch, ok := buildHostNotch(im.host, maps, tanA, tanB, hostBDetour(im, tl, seam))
	if !ok {
		return false
	}
	subs, ok := bossWallSubArcs(im, seam, tl.fbL, tl.stL, tl.stR, tl.fbR)
	if !ok {
		return false
	}
	split, ok := buildSplitBossWall(wall, im.footprintEdge, subs)
	if !ok {
		return false
	}
	set.replace[im.host.ID()], set.replace[wall.ID()] = notch, split
	return true
}

// buildHostAAndWall reconstructs host plane A (the narrower r6 boss, whose footprint crossings caL/caR
// sit INSIDE the fillet cuts faL/faR) and splits its boss wall. The notch detour absorbs the flat-fill
// lune directly into the host loop (no separate faces): from each cut it runs the runout curve up to the
// seam bottom, the featA arc down to the crossing, the boss host-side arc (split at the wall seam), and
// back out symmetrically, dropping the boss hole. The wall's three host/featA sub-arcs weld to this
// notch, its cap sub-arc to the central patch (Task 10b step 3).
func buildHostAAndWall(set *runoutSet, ef edgeFillet, tl runoutTiling, maps filletRebuildMaps) bool {
	im := tl.aImp
	wall := otherFace(im.footprintEdge, im.host)
	if wall == nil {
		return false
	}
	seam := im.footprintEdge.StartVertex().Point()
	tanA, tanB := hostTangent(ef.c0, im.hostIsA), hostTangent(ef.c1, im.hostIsA)
	notch, ok := buildHostNotch(im.host, maps, tanA, tanB, hostADetour(im, tl, seam))
	if !ok {
		return false
	}
	subs, ok := bossWallSubArcs(im, seam, tl.caR, tl.sbR, tl.sbL, tl.caL)
	if !ok {
		return false
	}
	split, ok := buildSplitBossWall(wall, im.footprintEdge, subs)
	if !ok {
		return false
	}
	set.replace[im.host.ID()], set.replace[wall.ID()] = notch, split
	return true
}

// hostADetour returns the host-A notch builder: from a receded tangent corner, a straight survivor to
// the fillet cut, the runout curve up to the seam bottom (shared with the flank patch), the featA arc in
// to the boss crossing (shared with the split wall), the boss host-side arc split at the wall seam
// (crossNear→seam, seam→crossFar), the featA arc out, the runout curve down, and the survivor home. Every
// arc is sampled identically to its neighbour so all weld point-for-point; the enclosed lune is the flat
// fill that lifts the area toward OCCT.
func hostADetour(im runoutImprint, tl runoutTiling, seam math.Point3) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		near, far := hostAEnds(tl, from)
		arcs, ok := hostAArcs(im, near, far, seam)
		if !ok {
			return nil, false
		}
		segs := []notchSeg{{from, nil}}
		for _, a := range arcs {
			segs = appendArcSegs(segs, a)
		}
		return append(segs, notchSeg{far.fa, nil}), true
	}
}

// hostAEnd bundles one side's fillet cut, seam bottom, and boss crossing (the runout curve runs fa→sb,
// the featA arc sb→ca).
type hostAEnd struct{ fa, sb, ca math.Point3 }

// hostAEnds orders the two sides so `near` is the one the traversal enters from (nearer `from`).
func hostAEnds(tl runoutTiling, from math.Point3) (near, far hostAEnd) {
	left := hostAEnd{tl.faL, tl.sbL, tl.caL}
	right := hostAEnd{tl.faR, tl.sbR, tl.caR}
	if from.DistanceTo(tl.faR) < from.DistanceTo(tl.faL) {
		return right, left
	}
	return left, right
}

// hostAArcs is the ordered arc chain of the host-A detour interior (near.fa → … → far.fa, exclusive of
// far.fa): runout up, featA in, boss host arc split at the seam, featA out, runout down.
func hostAArcs(im runoutImprint, near, far hostAEnd, seam math.Point3) ([]geom.Curve3, bool) {
	featIn, ok0 := featureSubArc(im, near.sb, near.ca)
	host0, ok1 := hostSideArc(im, near.ca, seam)
	host1, ok2 := hostSideArc(im, seam, far.ca)
	featOut, ok3 := featureSubArc(im, far.ca, far.sb)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return nil, false
	}
	return []geom.Curve3{
		planeARunoutCurve(near.fa, near.sb), featIn, host0, host1, featOut, planeARunoutCurve(far.sb, far.fa),
	}, true
}

// hostTangent is the fillet's tangent point on a host face at corner c: ta when the host is the fillet's
// a face (hostIsA), tb otherwise — the receded tangent-line endpoints the host notch re-cuts between.
func hostTangent(c corner, hostIsA bool) math.Point3 {
	if hostIsA {
		return c.ta
	}
	return c.tb
}

// hostBDetour returns the host-B notch detour builder: from one receded tangent corner, a straight
// survivor to the near fillet-cut crossing (fbL/fbR), the boss host-side arc (major then minor, split at
// the wall seam point so both pieces weld to the split wall), and a straight survivor to the other
// corner. Sampled identically to the wall's host sub-arcs (hostSideArc) so the two weld point-for-point.
func hostBDetour(im runoutImprint, tl runoutTiling, seam math.Point3) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		fbFrom, fbTo := tl.fbL, tl.fbR
		if from.DistanceTo(tl.fbR) < from.DistanceTo(tl.fbL) {
			fbFrom, fbTo = tl.fbR, tl.fbL
		}
		arc1, ok1 := hostSideArc(im, fbFrom, seam)
		arc2, ok2 := hostSideArc(im, seam, fbTo)
		if !ok1 || !ok2 {
			return nil, false
		}
		segs := []notchSeg{{from, nil}}
		segs = appendArcSegs(segs, arc1)
		segs = appendArcSegs(segs, arc2)
		return append(segs, notchSeg{fbTo, nil}), true
	}
}

// appendArcSegs appends arc's open samples (excluding the far endpoint) as STRAIGHT chords (curve nil).
// The samples are the same points the neighbour (wall/patch) tiles from the identical arc, so the faces
// weld; a nil curve keeps each welded edge a LineSegment between two on-surface points — sampleEdgeCurve
// would otherwise re-trace the FULL arc over each 1/6-span sub-edge and self-cross the boundary (10b).
func appendArcSegs(segs []notchSeg, arc geom.Curve3) []notchSeg {
	for _, p := range sampleCurve3Open(arc, false) {
		segs = append(segs, notchSeg{p, nil})
	}
	return segs
}

// buildHostNotch rebuilds a host plane by transformFace (its receded outer loop, identical to the non-
// obstacle path) and then replaces the straight receded-tangent segment (the one between the two host
// tangent points tanA,tanB) with the detour, dropping every inner loop (the boss footprint hole is now
// merged into the outer boundary). ok=false honest-rejects a host whose tangent segment is not found.
func buildHostNotch(host *topo.Face, maps filletRebuildMaps, tanA, tanB math.Point3,
	detour func(from, to math.Point3) ([]notchSeg, bool)) (filletFace, bool) {
	base := transformFace(host, maps.abSubst[host], maps.endCorner[host], maps.edgeInserts[host], maps.spreads[host])
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
	for k := 0; k < n; k++ {
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
				out.addID(s.pt, s.curve, 0, 0)
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
