// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Two-pass radial plan for the unified boolean stitch (ADR-0058). Pass 1 welds every loop edge's
// endpoints and midpoint and buckets the directed uses by GEOMETRIC edge — endpoints plus midpoint, so
// two arcs joining the same vertices stay distinct. Pass 2 resolves each over-used bucket — a
// tangent/grazing contact where coincident faces meet along one curve — into manifold pairs with the
// surface-agnostic Weiler radial sew (boolean_radial_edge.go): the azimuth axis is the CURVE TANGENT at
// the edge midpoint and each face's direction is its outward surface normal evaluated there (OCCT
// BOPTools GetFaceOff/GetFaceDir), then partitionVertexDisks splits pinched vertices per radial disk.
// The plan is naming-free; the minter (curved_stitch.go) builds entities from it lazily in first-use
// order, so the edge and vertex ordinals of every previously-manifold case are unchanged.

// stitchKeyUses accumulates one geometric edge's directed uses: the canonical representative (the
// first-encountered loopEdge, whose own direction defines "not reversed"), whether it is closed (a
// full seam circle), the welded endpoint pair AS TRAVERSED BY THE REPRESENTATIVE (pair[0] is the
// rep's start weld — cached so no later stage re-probes the welder), and every use tagged relative
// to the representative.
type stitchKeyUses struct {
	rep    loopEdge
	closed bool
	pair   [2]int
	uses   []loopEdgeUse
}

// stitchSlot is one loop-edge use's resolved plan entry: the edge group it belongs to and its
// rep-relative direction — indexed [face][loop][pos] so the mint stage does no map lookups.
type stitchSlot struct {
	gi  int
	rev bool
}

// curvedStitchPlan is the radial-sew plan for a curved face set plus the per-group data the minter
// needs: each group's canonical representative loopEdge, its traversal-order endpoint welds, the
// closedness of its geometric edge, and every use's slot — all cached from pass 1 so the mint stage
// runs probe-free on the hot path. tangent reports whether any geometric edge was over-used (a
// tangent/grazing contact was resolved).
type curvedStitchPlan struct {
	sew     sewPlan
	rep     []loopEdge
	repEnds [][2]int // per group: the rep's (start, end) welded indices
	closed  []bool
	slots   [][][]stitchSlot // [face][loop][pos] → group + rep-relative direction
	tangent bool
}

// buildCurvedStitchPlan collects every loop-edge use — welding endpoints and midpoints in traversal
// order, so welder indices match the retired single-pass welder — and resolves each geometric edge
// into manifold groups plus the per-vertex disk partition.
func buildCurvedStitchPlan(faces []curvedFace, pw *welder3) curvedStitchPlan {
	keys, byKey, slots := collectStitchUses(faces, pw)
	p := curvedStitchPlan{slots: slots}
	var groups []edgeGroup
	for _, k := range keys {
		ku := byKey[k]
		p.tangent = p.tangent || len(ku.uses) > 2
		for _, g := range resolveEdgeUses(ku.uses, stitchAxisOf(ku.rep), stitchNormalAt(faces)) {
			gi := len(groups)
			groups = append(groups, edgeGroup{pair: sortedPair(ku.pair), uses: g})
			p.rep = append(p.rep, ku.rep)
			p.repEnds = append(p.repEnds, ku.pair)
			p.closed = append(p.closed, ku.closed)
			for _, u := range g {
				slots[u.face][u.ring][u.pos].gi = gi
			}
		}
	}
	p.sew = sewPlan{groups: groups, disks: partitionVertexDisks(groups)}
	return p
}

// sortedPair orders a welded endpoint pair ascending (the disk partition's canonical pair form).
func sortedPair(p [2]int) [2]int {
	if p[0] > p[1] {
		return [2]int{p[1], p[0]}
	}
	return p
}

// collectStitchUses is pass 1: bucket every loop edge use by geometric-edge key, in deterministic
// first-encounter order (the face/loop/edge traversal), recording each use's rep-relative direction
// in its slot.
func collectStitchUses(faces []curvedFace, pw *welder3) ([][3]int, map[[3]int]*stitchKeyUses, [][][]stitchSlot) {
	byKey := map[[3]int]*stitchKeyUses{}
	slots := make([][][]stitchSlot, len(faces))
	var keys [][3]int
	for fi := range faces {
		slots[fi] = collectFaceStitchUses(fi, faces[fi], pw, byKey, &keys)
	}
	return keys, byKey, slots
}

// collectFaceStitchUses records one face's directed loop-edge uses into the key buckets and returns
// the face's slot table. Consecutive loop edges share a point (edge i's end IS edge i+1's start), so
// the previous end's weld is carried instead of re-probed.
func collectFaceStitchUses(fi int, f curvedFace, pw *welder3, byKey map[[3]int]*stitchKeyUses, keys *[][3]int) [][]stitchSlot {
	slots := make([][]stitchSlot, len(f.loops))
	for li, loop := range f.loops {
		slots[li] = make([]stitchSlot, len(loop.edges))
		prevEnd, prevKb := math.Point3{}, -1
		for ei, le := range loop.edges {
			ku, ka, kb, end := stitchKeyFor(le, pw, byKey, keys, prevEnd, prevKb)
			prevEnd, prevKb = end, kb
			rev := stitchUseReversed(le, ka, ku)
			slots[li][ei].rev = rev
			ku.uses = append(ku.uses, loopEdgeUse{face: fi, ring: li, pos: ei, reversed: rev})
		}
	}
	return slots
}

// stitchKeyFor welds the edge's endpoints — and, for a curved edge, its midpoint, so two arcs joining
// the same vertices stay distinct; a straight segment is determined by its endpoints, so its midpoint
// weld is skipped (pure cost) — in traversal order, matching the retired welder's insertion order for
// curved edges. The previous edge's end weld is reused when this edge starts exactly there (the loop
// invariant). It returns the accumulator for its geometric-edge key — created, with this edge as the
// canonical representative, on first encounter — plus this use's start/end welds and end point.
func stitchKeyFor(le loopEdge, pw *welder3, byKey map[[3]int]*stitchKeyUses, keys *[][3]int,
	prevEnd math.Point3, prevKb int) (*stitchKeyUses, int, int, math.Point3) {
	a, b := le.start(), le.end()
	ka := prevKb
	if prevKb < 0 || a != prevEnd {
		ka = pw.add(a)
	}
	kb := pw.add(b)
	km := -1
	switch le.curve.(type) {
	case geom.LineSegment, geom.Line:
	default:
		km = pw.add(le.curve.PointAt((le.t0 + le.t1) / 2))
	}
	lo, hi := ka, kb
	if lo > hi {
		lo, hi = hi, lo
	}
	key := [3]int{lo, hi, km}
	if ku, ok := byKey[key]; ok {
		return ku, ka, kb, b
	}
	ku := &stitchKeyUses{rep: le, closed: ka == kb, pair: [2]int{ka, kb}}
	byKey[key] = ku
	*keys = append(*keys, key)
	return ku, ka, kb, b
}

// stitchUseReversed reports whether this use traverses the geometric edge opposite its canonical
// representative: by sweep sign for a closed edge (a full seam circle), by start weld otherwise —
// both from pass-1 cached indices, no welder re-probe.
func stitchUseReversed(le loopEdge, ka int, ku *stitchKeyUses) bool {
	if ku.closed {
		return (le.t1 < le.t0) != (ku.rep.t1 < ku.rep.t0)
	}
	return ka != ku.pair[0]
}

// stitchAxisOf is the lazy radial axis of a geometric edge: the canonical representative's unit curve
// tangent at the edge midpoint, oriented along its traversal, with the endpoint chord as the
// degenerate-tangent fallback. Evaluated only for an over-used (tangent-contact) edge.
func stitchAxisOf(rep loopEdge) func() (math.Vector3, math.Point3) {
	return func() (math.Vector3, math.Point3) {
		tm := (rep.t0 + rep.t1) / 2
		mid := rep.curve.PointAt(tm)
		tan := rep.curve.TangentAt(tm)
		if rep.t1 < rep.t0 {
			tan = tan.Scale(-1)
		}
		if u, err := math.UnitVector3FromVector(tan); err == nil {
			return u.AsVector(), mid
		}
		return stitchChordAxis(rep), mid
	}
}

// stitchChordAxis is the degenerate-tangent fallback: the edge's endpoint chord, or an arbitrary (but
// total) axis when even the chord is degenerate — the radial sort is then arbitrary yet deterministic,
// and an unpairable contact declines downstream as a non-solid rather than crashing.
func stitchChordAxis(rep loopEdge) math.Vector3 {
	if u, err := math.UnitVector3FromVector(rep.start().VectorTo(rep.end())); err == nil {
		return u.AsVector()
	}
	return math.V3(1, 0, 0)
}

// stitchNormalAt is the unified stitch's faceDirAt: the using face's OUTWARD surface normal at the edge
// point — flipped for a reversed face, whose surface normal points into the material (OCCT GetFaceDir).
func stitchNormalAt(faces []curvedFace) faceDirAt {
	return func(h loopEdgeUse, p math.Point3) math.Vector3 {
		f := faces[h.face]
		u, v := f.surface.ParamAt(p)
		n := f.surface.NormalAt(u, v)
		if f.reversed {
			return n.Scale(-1)
		}
		return n
	}
}
