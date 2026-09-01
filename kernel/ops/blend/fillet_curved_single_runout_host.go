// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Building the FACES of a single-arm run-out, once its rails and trims are solved (split out of
// fillet_curved_single_runout.go for #2221).
//
// The blend face itself is straightforward; the work is the host, which has a bite taken out of it
// and must be re-trimmed around that bite without disturbing its other loops. A host face the
// run-out does not touch passes through unchanged.

// singleRunoutFaces builds every result face: the trimmed arm face bounded by the both-ends loop [railA,
// trim@cap1, railB(rev), trim@cap0(rev)], plus the retrimmed hosts. Each of the FOUR bitten faces (the two
// arm hosts ef.a/ef.b, and the two caps run0.capping/run1.capping) has its own rail/trim spliced into its
// loop (farRunoutFace); every other body face passes through verbatim. A retrim decline floors honestly.
func singleRunoutFaces(body *topo.Body, ef edgeFillet, arm geom.Surface, railA, railB endSeg, run0, run1 armRunout, r float64, res opstol.Resolution) ([]filletFace, string) {
	loop := append([]endSeg{railA, run1.trim}, reverseEndSegs([]endSeg{railB})...)
	loop = append(loop, reverseEndSegs([]endSeg{run0.trim})...)
	armFace := filletFace{surface: arm, loops: []filletLoop{loopFromSegs(loop)}, parent: filletEdgeProvenance(ef.edge)}
	hosts, reason := singleRunoutHostFaces(body, ef, railA, railB, run0, run1, r, res)
	if reason != "" {
		return nil, reason
	}
	return append(hosts, armFace), ""
}

// singleRunoutHostFaces retrims the four bitten hosts and carries every other face through unchanged. Each
// bitten host keeps the boundary span AWAY from the sharp feature the fillet consumed and splices in the
// bite: an arm host (ef.a/ef.b) recedes along its contact rail, dropping the span carrying the PICKED EDGE
// (identified by either of its end vertices, which lie in that span); a cap drops the small corner carrying
// the FAR VERTEX its far cross-section arc bites. The removed span is chosen by which vertex it contains
// (farPathSegs), NOT by area — a single-arm runout removes ~half of an arm host, so the far-runout "smaller
// corner" heuristic would splice the wrong side. Each bite is the SAME curve object the arm face carries, so
// the two sides weld watertight.
func singleRunoutHostFaces(body *topo.Body, ef edgeFillet, railA, railB endSeg, run0, run1 armRunout, r float64, res opstol.Resolution) ([]filletFace, string) {
	tol := res.Weld() * r
	v0, v1 := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	bites := map[*topo.Face]endSeg{ef.a: railA, ef.b: railB, run0.capping: run0.trim, run1.capping: run1.trim}
	// The vertex the removed span carries: the picked edge (either endpoint) for an arm host; the far vertex
	// the arc bites for a cap. run0 terminates the START end (its capping bites v0), run1 the END end (v1).
	avoid := map[*topo.Face]math.Point3{ef.a: v0, ef.b: v0, run0.capping: v0, run1.capping: v1}
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		bite, bitten := bites[f]
		if !bitten {
			out = append(out, passthroughFace(f)) // untouched by the runout — verbatim (coordinate-welded)
			continue
		}
		ff, ok := runoutHostRetrim(f, ef, bite, avoid[f], tol)
		if !ok {
			// P4-class capped end: a bite foot is a mid-face fresh cut — bridge it back to the
			// picked edge's end vertex along wall ∩ cap and splice the chain (do-no-harm on decline).
			ff, ok = bridgedRunoutHostFace(f, ef, bite, tol)
		}
		if !ok {
			return nil, fmt.Sprintf("single-arm runout: host %T retrim declined (bite %v→%v)", f.Geometry(), bite.from, bite.to)
		}
		out = append(out, ff)
	}
	return out, ""
}

// runoutHostRetrim dispatches one bitten host's retrim: a CONCAVE arm host (ef.a/ef.b on an armConcave
// fillet — N3/M4/N9) GROWS to the contact rail via concaveArmHostRetrim (feet on the rim-edge
// extensions), while every convex host and every cap keeps the byte-identical recede-and-splice
// singleRunoutHostFace. Gating on ef.armConcave keeps the convex single-arm runout greens bit-identical.
func runoutHostRetrim(f *topo.Face, ef edgeFillet, bite endSeg, avoid math.Point3, tol float64) (filletFace, bool) {
	if !ef.armConcave {
		return singleRunoutHostFace(f, bite, avoid, tol)
	}
	if f == ef.a || f == ef.b {
		return concaveArmHostRetrim(f, bite, ef.edge, tol) // arm host GROWS to the contact rail
	}
	return concaveCapRetrim(f, bite, avoid, tol) // end cap GAINS the fill wedge (variant b)
}

// passthroughFace carries a face the runout does not touch through UNCHANGED, but with COORDINATE-welded
// (op-generated, id-0) loop points rather than transformFace's source-id-carrying loops. The retrimmed
// hosts weld by coordinate (loopFromSegs drops source ids), and the point-welder never merges an id-carrying
// point onto an id-0 one — so a source-id pass-through face would NOT weld to its retrimmed neighbour and
// split the shared edge (the B6 other-radial face open-shell). Provenance (the face's own lineage) is kept.
func passthroughFace(f *topo.Face) filletFace {
	loops := make([]filletLoop, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		loops = append(loops, loopFromSegs(segsFromLoop(l)))
	}
	return filletFace{surface: f.Geometry(), loops: loops, parent: f.Lineage()}
}

// singleRunoutHostFace re-clips one bitten host: it retrims the loop the bite actually consumes (the loop
// carrying the removed feature vertex — the OUTER boundary for a simple host, but M7's flush-cut cap is bitten
// on its INNER footprint loop), keeping that loop's span from the bite's far foot back to its near foot that
// AVOIDS the removed vertex, then closing with the bite (the contact rail / far cross-section arc). Every OTHER
// loop (e.g. M7 cap's unrelated outer box boundary) is carried through unchanged. Loop ROLES are preserved —
// the OUTER loop stays at index 0 (assembleBody keys outer-ness on that) so the retrimmed inner footprint
// stays a hole, not a phantom outer. Declines when the bitten loop is too small or a bite foot is off it.
func singleRunoutHostFace(host *topo.Face, bite endSeg, avoid math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(host, avoid, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false // malformed host (no loops) — do-no-harm
	}
	retrim, ok := retrimBittenLoop(bitten, bite, avoid, tol)
	if !ok {
		return filletFace{}, false // a bite foot is not on the bitten loop, or the far path cannot close
	}
	loops := hostLoopsWithRetrim(host, bitten, outer, retrim)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// retrimBittenLoop closes the retrimmed bitten loop: the surviving far-path span (bite's far foot back to its
// near foot, avoiding the removed vertex) plus the bite (contact rail / far cross-section arc). Declines when
// the loop is too small or a foot is off it. loopFromSegs drops source ids (coordinate weld) as before.
func retrimBittenLoop(bitten *topo.Loop, bite endSeg, avoid math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	// ≥2, not ≥3: a two-arc "lens" loop (two intersecting-cylinder caps, e.g. blend/simple/O8's
	// top cap) is a legitimate bitten wire — farPathSegs splits it at the bite feet and the far
	// path closes exactly as on a many-seg loop. Only a single-seg (whole-circle) wire, which the
	// split machinery cannot anchor on, stays declined.
	if len(segs) < 2 {
		return filletLoop{}, false
	}
	far, ok := farPathSegs(segs, bite.to, bite.from, avoid, tol)
	if !ok {
		return filletLoop{}, false
	}
	return loopFromSegs(append([]endSeg{bite}, far...)), true
}

// hostLoopsWithRetrim rebuilds the host's loop set with the bitten loop replaced by its retrim and every
// other loop carried through unchanged, EMITTING THE OUTER LOOP FIRST (index 0) because assembleBody marks
// loops[0] as the outer boundary. On a single-loop host (every prior single-arm green) this is just the
// retrimmed outer loop — byte-identical to the previous [retrim]+no-inner emission.
func hostLoopsWithRetrim(host *topo.Face, bitten, outer *topo.Loop, retrim filletLoop) []filletLoop {
	out := []filletLoop{loopForRole(outer, bitten, retrim)}
	for _, l := range host.Loops() {
		if l != outer {
			out = append(out, loopForRole(l, bitten, retrim))
		}
	}
	return out
}

// loopForRole yields the retrim for the bitten loop and a COORDINATE-welded (loopFromSegs, id-0) pass-through
// for any other loop — matching passthroughFace, so M7 cap's surviving outer box boundary welds to the id-0
// pass-through box walls that share it (an id-carrying loop would not merge onto an id-0 neighbour, splitting
// the shared edge).
func loopForRole(l, bitten *topo.Loop, retrim filletLoop) filletLoop {
	if l == bitten {
		return retrim
	}
	return loopFromSegs(segsFromLoop(l))
}

// hostBittenLoop selects the host loop the fillet bite actually consumes — the loop carrying the picked
// feature vertex the bite removes. It PREFERS the outer loop whenever that carries the vertex (every prior
// single-arm green — B6/C9/C1/B7 and M7's three non-cap hosts — so their retrim stays byte-identical); it
// drops to an inner loop only when the vertex is NOT on the outer boundary. That is M7's flush-cut cap (plane
// x=60 through the cylinder axis): the picked-edge vertex lives on the cap's INNER footprint-hole loop, so
// that hole (not the untouched outer box square) is the wire that recedes along the bite. nil for a loopless host.
func hostBittenLoop(host *topo.Face, avoid math.Point3, tol float64) *topo.Loop {
	outer := outerHostLoop(host)
	if outer != nil && loopHasVertex(outer, avoid, tol) {
		return outer
	}
	for _, l := range host.Loops() {
		if loopHasVertex(l, avoid, tol) {
			return l
		}
	}
	return outer
}

// loopHasVertex reports whether p coincides (within the model-relative tol) with one of the loop's vertices.
func loopHasVertex(l *topo.Loop, p math.Point3, tol float64) bool {
	for _, u := range l.EdgeUses() {
		if float64(useFromVertex(u).Point().DistanceTo(p)) <= tol {
			return true
		}
	}
	return false
}
