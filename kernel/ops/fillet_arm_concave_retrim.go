// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CONCAVE arm host/cap retrim machinery (split out of fillet_arm_concave.go — CLAUDE.md files-under-500
// — once the armTorus dispatch pushed that file over the line; a pure move, no behaviour change). Unlike
// the convex recede-and-splice (feet land INTERIOR to the host loop), a concave fillet ADDS the fill
// wedge: both the two arm hosts and the two end caps GROW, so every retrim here re-terminates a flanking
// edge OUTWARD onto a contact foot rather than trimming one away. See fillet_arm_concave.go for the arm
// surface construction (concaveTorusArmEdge / concaveCylinderArmEdge) that feeds the rail/arc this
// machinery consumes.

// concaveArmHostRetrim re-clips one CONCAVE arm host (ef.a or ef.b) to the contact rail. Unlike the
// convex recede-and-splice (feet land INTERIOR to the host loop), a concave fillet ADDS the fill wedge:
// the host GROWS, its two contact feet landing on the flanking rim edges EXTENDED past the picked
// vertex (N9's rim arcs; N3's plane top edge). So the retrim replaces the picked-edge segment with the
// straight contact rail and RE-TERMINATES the two flanking rim edges onto the rail's feet (a straight
// ruling or a rim arc grown/receded to the foot on its OWN line/circle). Declines when the picked
// segment is absent from the loop or a foot is off a flanking edge's supporting line/circle (do-no-harm).
func concaveArmHostRetrim(host *topo.Face, rail endSeg, edge *topo.Edge, tol float64) (filletFace, bool) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	bitten := hostBittenLoop(host, v0, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false // malformed host (no loops) — do-no-harm
	}
	retrim, ok := concaveRetrimLoop(bitten, rail, v0, v1, tol)
	if !ok {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(host, bitten, outer, retrim)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// concaveRetrimLoop rebuilds the bitten loop for a concave arm host: it finds the picked-edge segment
// (endpoints {v0,v1}), replaces it with the straight contact rail, and re-terminates the two flanking
// segments onto the rail's feet (the rail was built foot0↔v0 → foot1↔v1). Every other segment is
// carried through verbatim, so the ring stays closed by construction. Declines when the picked segment
// is not on the loop or a foot leaves a flanking edge's supporting line/circle.
func concaveRetrimLoop(bitten *topo.Loop, rail endSeg, v0, v1 math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	n := len(segs)
	i := indexOfPickedEdge(segs, v0, v1, tol)
	if i < 0 || n < 3 {
		return filletLoop{}, false
	}
	fFrom, fTo := railFeetForPicked(segs[i], rail, v0, tol)
	prev, okp := reterminateSegTo(segs[(i-1+n)%n], fFrom, tol)
	next, okn := reterminateSegFrom(segs[(i+1)%n], fTo, tol)
	if !okp || !okn {
		return filletLoop{}, false
	}
	return loopFromSegs(spliceConcaveRing(segs, i, prev, endSeg{from: fFrom, to: fTo}, next)), true
}

// indexOfPickedEdge returns the loop segment whose endpoints are the picked edge's vertices {v0,v1}
// (either orientation), or −1 when the picked edge is not a single segment of this loop.
func indexOfPickedEdge(segs []endSeg, v0, v1 math.Point3, tol float64) int {
	for i, s := range segs {
		if pointsMatch(s.from, s.to, v0, v1, tol) || pointsMatch(s.from, s.to, v1, v0, tol) {
			return i
		}
	}
	return -1
}

// pointsMatch reports whether (a,b) coincide with (c,d) within tol, respectively.
func pointsMatch(a, b, c, d math.Point3, tol float64) bool {
	return float64(a.DistanceTo(c)) <= tol && float64(b.DistanceTo(d)) <= tol
}

// railFeetForPicked orders the rail's feet to the picked segment's own traversal: the rail runs
// foot0↔v0 → foot1↔v1, so a picked segment traversed v0→v1 keeps (foot0,foot1), v1→v0 swaps them.
func railFeetForPicked(picked, rail endSeg, v0 math.Point3, tol float64) (math.Point3, math.Point3) {
	if float64(picked.from.DistanceTo(v0)) <= tol {
		return rail.from, rail.to
	}
	return rail.to, rail.from
}

// spliceConcaveRing rebuilds the ordered ring with the picked segment i replaced by the straight rail
// and its two neighbours replaced by their re-terminated forms — positions preserved so the ring stays
// closed (prev.to == rail.from, rail.to == next.from by construction).
func spliceConcaveRing(segs []endSeg, i int, prev, rail, next endSeg) []endSeg {
	n := len(segs)
	out := make([]endSeg, n)
	for k := range n {
		switch k {
		case (i - 1 + n) % n:
			out[k] = prev
		case i:
			out[k] = rail
		case (i + 1) % n:
			out[k] = next
		default:
			out[k] = segs[k]
		}
	}
	return out
}

// reterminateSegTo moves segment s's FAR endpoint to newTo along its own supporting line (straight) or
// circle (arc) — growing or receding the flanking rim edge to the contact foot. Declines when newTo
// leaves that line/circle (the foot is not co-linear/co-circular — not a valid concave grow).
func reterminateSegTo(s endSeg, newTo math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newTo, tol) {
			return endSeg{}, false
		}
		return endSeg{from: s.from, to: newTo}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), s.from, newTo, tol)
}

// reterminateSegFrom moves segment s's NEAR endpoint to newFrom along its own line/circle — the mirror
// of reterminateSegTo for the flanking edge on the picked segment's far side.
func reterminateSegFrom(s endSeg, newFrom math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newFrom, tol) {
			return endSeg{}, false
		}
		return endSeg{from: newFrom, to: s.to}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), newFrom, s.to, tol)
}

// rebuildArcSeg builds the sub-arc of arc's own circle between from and to (both required on that
// circle within tol) — the arc analogue of extending a straight rim edge to the foot. A MAJOR (>π)
// span is carried from the PARENT's own parameters (subArcMajor); only a minor span uses the
// shorter-arc three-point re-fit. Declines when either endpoint is off the circle or the fit fails.
//
// ★ The major arm is the fix for the M8 complementary-arc defect: the shorter-arc midpoint below
// makes Arc3dByThreePoints return the COMPLEMENTARY MINOR arc whenever the retained span exceeds a
// semicircle (the N7 whole-curve-sub-span lesson). M8's boss rim is a 270° arc trimmed to 255.52° by
// the fillet setback, and this function handed the edge catalog its 104.48° complement — a curve
// 22.01 away from the one the top plane offered for the SAME welded edge, arbitrated only by build
// order. subSeg and retainedRimCurve already guarded this way; this call site never got the guard.
func rebuildArcSeg(arc geom.Arc3d, from, to math.Point3, tol float64) (endSeg, bool) {
	if !onCircle(arc, from, tol) || !onCircle(arc, to, tol) {
		return endSeg{}, false
	}
	if sub, mid, major := subArcMajor(arc, from, to); major {
		return endSeg{from: from, to: to, curve: sub, mid: mid, arc: true}, true
	}
	mid := arcMidBetween(arc.Center, arc.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: from, to: to, curve: sub, mid: mid, arc: true}, true
}

// onCircle reports whether p lies on arc's circle (centre distance ≈ radius within tol).
func onCircle(arc geom.Arc3d, p math.Point3, tol float64) bool {
	return stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) <= tol
}

// pointOnLine reports whether p lies on the INFINITE line through a→b within tol (co-linear, so an
// extension past b is admitted — the concave grow).
func pointOnLine(a, b, p math.Point3, tol float64) bool {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return false
	}
	t := float64(a.VectorTo(p).Dot(d)) / l2
	return float64(a.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p)) <= tol
}

// concaveCapRetrim re-clips one CONCAVE end cap (run0/run1.capping): for a reentrant fillet the cap
// GAINS the fill wedge instead of losing a corner (variant (b), opposite sign to the convex cap bite),
// so the cap's far-vertex corner is REPLACED by the cross-section arc and the two edges meeting there
// are re-terminated onto the arc's feet (each on its own host-shared line/circle). Declines when the
// far vertex is not a loop corner or an arc foot is off a flanking edge — the do-no-harm floor.
func concaveCapRetrim(cap *topo.Face, arc endSeg, far math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(cap, far, tol)
	outer := outerHostLoop(cap)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	retrim, ok := concaveCapLoop(bitten, arc, far, tol)
	if !ok {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(cap, bitten, outer, retrim)
	return filletFace{surface: cap.Geometry(), loops: loops, parent: cap.Lineage()}, true
}

// concaveCapLoop rebuilds a cap's bitten loop by replacing the far-vertex CORNER with the cross-section
// arc: the two flanking edges (each shared with one host) are re-terminated onto the arc's feet and the
// arc is spliced between them, growing the loop by one segment. Declines when the far vertex is not a
// loop corner or the arc feet do not land on the two flanking edges' supporting line/circle.
func concaveCapLoop(bitten *topo.Loop, arc endSeg, far math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	n := len(segs)
	j := indexOfVertex(segs, far, tol)
	if j < 0 || n < 3 {
		return filletLoop{}, false
	}
	prevIdx := (j - 1 + n) % n
	arcSeg, ok := matchArcFeet(segs[prevIdx], segs[j], arc, tol)
	if !ok {
		return filletLoop{}, false
	}
	prev, okp := reterminateSegTo(segs[prevIdx], arcSeg.from, tol)
	next, okn := reterminateSegFrom(segs[j], arcSeg.to, tol)
	if !okp || !okn {
		return filletLoop{}, false
	}
	return loopFromSegs(spliceCapRing(segs, prevIdx, j, prev, arcSeg, next)), true
}

// indexOfVertex returns the segment index whose FROM endpoint coincides with p (the loop corner p
// starts), or −1 when no segment leaves p.
func indexOfVertex(segs []endSeg, p math.Point3, tol float64) int {
	for j, s := range segs {
		if float64(s.from.DistanceTo(p)) <= tol {
			return j
		}
	}
	return -1
}

// matchArcFeet orients the cross-section arc onto the two flanking cap edges: its `from` foot must land on
// prev's supporting line/circle and its `to` foot on next's, so the returned segment runs prev-side →
// next-side. Tries both endpoint orderings; ok=false when neither pairing lands each foot on its flanking
// edge.
//
// The swapped pairing returns the arc REVERSED (reversedEndSeg), not merely its endpoints exchanged: it
// used to hand back the two points while the caller re-wrapped `arc.curve` UNCHANGED, so the spliced
// segment's curve ran to→from. discretizeEdge then forced that edge's polyline ends onto the edge's own
// vertices and produced a doubled-back boundary whose developed loop self-crosses — simple/M4 (pinching off
// 59.58 of its cap), simple/N3 (4.554) and simple/N9 (4.150), each of which also carried the corpus's worst
// curve-vs-vertex gap for its case (17.3205 / 6.32456 / 8.16497, knownEdgeSpanDebt). Only the swapped branch
// changes; the aligned pairing is returned untouched, so every aligned case stays byte-identical.
func matchArcFeet(prev, next, arc endSeg, tol float64) (endSeg, bool) {
	if segSupportsPoint(prev, arc.from, tol) && segSupportsPoint(next, arc.to, tol) {
		return arc, true
	}
	if segSupportsPoint(prev, arc.to, tol) && segSupportsPoint(next, arc.from, tol) {
		return reversedEndSeg(arc), true
	}
	return endSeg{}, false
}

// segSupportsPoint reports whether p lies on segment s's supporting geometry — its infinite line
// (straight) or its circle (arc) — the co-linear/co-circular test the concave grow re-terminates onto.
func segSupportsPoint(s endSeg, p math.Point3, tol float64) bool {
	if !s.arc {
		return pointOnLine(s.from, s.to, p, tol)
	}
	return onCircle(s.curve.(geom.Arc3d), p, tol)
}

// spliceCapRing rebuilds the ordered ring with the far-vertex corner (edges prevIdx→j) replaced by
// [prev, arc, next] — the arc INSERTED between the re-terminated flanking edges (the loop grows by one
// segment). prevIdx and j are adjacent (j = prevIdx+1 mod n), so order and closure are preserved.
func spliceCapRing(segs []endSeg, prevIdx, j int, prev, arc, next endSeg) []endSeg {
	out := make([]endSeg, 0, len(segs)+1)
	for k := range segs {
		switch k {
		case prevIdx:
			out = append(out, prev, arc)
		case j:
			out = append(out, next)
		default:
			out = append(out, segs[k])
		}
	}
	return out
}
