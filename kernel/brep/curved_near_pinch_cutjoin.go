// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Near-pinch crossing-cylinder CUT and JOIN (#1818). The near-pinch intersect trims the fat wall per lens loop
// so the two ovals never fuse in one arrangement (curved_general_boolean.go). Cut and join are harder: the fat
// wall's OUTSIDE-keep complement is a holed tube whose two lens holes near-touch at the neck, and the near-pinch
// (u,v) arrangement both fuses those holes AND emits their loops as sub-arcs that will not weld to the thin
// wall's faces. The fix builds EVERY near-pinch face from the RAW whole imprint loops, so a shared loop is one
// closed edge that welds as a single topo edge (shared discretization):
//
//   - the fat wall is the keyhole holed tube (crossingHoledTubeFace) with the two whole loops as separate holes
//     — its neck BRIDGE is meshed watertight by the corridor-seeding in kernel/ops (neckCorridorNodes), which
//     stops the wall from chording hole-to-hole across the neck (those chords would double the thin band's
//     matching pinch chords into non-manifold edges);
//   - the thin wall keeps the arrangement tunnel band when it stays INSIDE (already welds to whole loops), or
//     is built as raw two-rim stubs (rawStubBands) when it protrudes/severs OUTSIDE, where the arrangement
//     would fuse the near-severed stubs at the neck.
//
// A global curvedReorient then makes the mixed-source shell consistently oriented. The whole band down to the
// Steinmetz snap ceiling ships watertight (there is no conditioning floor above it — the surfaces stay > weld
// tol apart by Δr); the caller's validBooleanSolid gate is the fail-safe for any unforeseen degenerate case.

// keepsInside reports whether an operand under (op, isB) keeps the part of its wall INSIDE the other solid
// (the lens caps) rather than OUTSIDE it (the holed tube): true for an intersect, and for the tool of a cut.
func keepsInside(op Op, isB bool) bool {
	return op == Intersection || (op == Difference && isB)
}

// fatNearPinchWall builds the fat operand's near-pinch wall: its two lens caps (per loop) when it keeps the
// inside, or the holed tube when it keeps the outside — both from the loops kept separate, so the two ovals
// never share an arrangement (#1818). ok=false when the fat side or its holed tube cannot be resolved.
func fatNearPinchWall(fat ruledOperand, loops []geom.Curve3, op Op, isB bool, other func(math.Point3) bool) ([]curvedFace, bool) {
	f, cyl, band, ok := cylinderSideFace(fat.body)
	if !ok {
		return nil, false
	}
	if keepsInside(op, isB) {
		return keptOrNone(cylinderLensSplit(true, f, cyl, band, loops, isB, other))
	}
	tube, ok := crossingHoledTubeFace(f, cyl, band, loops, op, isB, other)
	if !ok {
		return nil, false
	}
	return []curvedFace{tube}, true
}

// crossingHoledTubeFace builds the fat wall's OUTSIDE-keep holed tube directly: the band's two rims bridged
// into a keyhole outer loop, with the two imprint loops supplied as SEPARATE holes (#1818). Bypassing the
// arrangement is what keeps the two near-touching lens holes from fusing; the holes reuse the shared imprint
// polylines (so they weld to the rod band exactly) with their winding reversed to the outer's sense.
func crossingHoledTubeFace(f curvedFace, cyl geom.Cylinder, band coneSideBand_, loops []geom.Curve3, op Op, isB bool, other func(math.Point3) bool) (curvedFace, bool) {
	c := newCylinderUVSolid(cyl, band, op, isB, other)
	holes := make([]curvedLoop, 0, len(loops))
	for i := range loops {
		holes = append(holes, curvedLoop{edges: []loopEdge{{curve: loops[i], t0: 1, t1: 0}}}) // reversed for a hole
	}
	all := append([]curvedLoop{{edges: c.keyholeOuter()}}, holes...)
	return curvedFace{surface: cyl, reversed: f.reversed, lineage: f.lineage, loops: all}, true
}

// rawStubBands builds a thin operand's OUTSIDE-keep protruding/severed stubs directly as raw two-closed-rim
// bands — one per imprint loop, each bounded by the thin cap on that loop's axial side and the WHOLE raw loop
// (#1818). Bypassing the (u,v) arrangement matters twice over: near the neck the two stubs nearly meet (a
// near-severed rod) and the arrangement fuses/degrades there; and its OUTSIDE emission splits the loop into
// sub-arcs that do NOT weld to the fat wall's whole-loop holes. A whole-loop closed edge welds to the fat
// wall's matching hole as ONE topo edge, so the discretization is shared and the stub closes watertight.
func rawStubBands(thin ruledOperand, loops []geom.Curve3) ([]curvedFace, bool) {
	if len(loops) != 2 {
		return nil, false
	}
	_, cyl, _, ok := cylinderSideFace(thin.body)
	if !ok {
		return nil, false
	}
	axis := cyl.AxisDir.AsVector()
	lc := [2]float64{loopCentroidAxial(loops[0], cyl.Origin, axis), loopCentroidAxial(loops[1], cyl.Origin, axis)}
	out := make([]curvedFace, 0, 2)
	for i := range loops {
		// A stub extends AWAY from the other loop: it is capped by the cap on the far side of this loop's
		// centroid from the other loop's. wantHigh selects the higher-axial cap for the higher-centroid loop.
		capCirc, okC := capOnAxialSide(planarCapFaces(thin.body), cyl, lc[i] >= lc[1-i])
		if !okC {
			return nil, false
		}
		// Winding is left to the global orientation pass (curvedReorient) — the stub's two rims are emitted in
		// their native sense and the flood-fill flips whichever faces a consistent shell needs.
		out = append(out, curvedFace{
			surface: cyl, lineage: thin.face.lineage,
			loops: []curvedLoop{
				{edges: []loopEdge{{curve: capCirc, t0: 0, t1: 1}}},
				{edges: []loopEdge{{curve: loops[i], t0: 0, t1: 1}}},
			},
		})
	}
	return out, true
}

// capOnAxialSide returns the planar cap circle at the higher (wantHigh) or lower axial end of the cylinder —
// the cap that closes a stub extending toward that end. ok=false if no planar cap circle is found.
func capOnAxialSide(caps []curvedFace, cyl geom.Cylinder, wantHigh bool) (geom.Circle, bool) {
	axis := cyl.AxisDir.AsVector()
	best, bestAxial, found := geom.Circle{}, 0.0, false
	for _, c := range caps {
		circ, ok := capCircleOf(c)
		if !ok {
			continue
		}
		a := float64(cyl.Origin.VectorTo(circ.Center).Dot(axis))
		if !found || (wantHigh && a > bestAxial) || (!wantHigh && a < bestAxial) {
			best, bestAxial, found = circ, a, true
		}
	}
	return best, found
}

// capCircleOf extracts the bounding circle of a planar cap face, or ok=false when the cap is not a single
// full-circle loop.
func capCircleOf(cap curvedFace) (geom.Circle, bool) {
	if len(cap.loops) == 0 || len(cap.loops[0].edges) != 1 {
		return geom.Circle{}, false
	}
	c, ok := cap.loops[0].edges[0].curve.(geom.Circle)
	return c, ok
}

// loopCentroidAxial is a loop's mean axial coordinate (distance along the cylinder axis from its origin).
func loopCentroidAxial(loop geom.Curve3, origin math.Point3, axis math.Vector3) float64 {
	pts := imprintLoopPoints(loop)
	var sum float64
	for _, p := range pts {
		sum += float64(origin.VectorTo(p).Dot(axis))
	}
	return sum / float64(len(pts))
}

// fatterOperand returns the two operands ordered (fat, thin) by side-cylinder radius, or ok=false when a
// side radius cannot be resolved — the near-pinch fat wall is the LARGER-radius wall (its two lens ovals
// nearly touch), the thin wall being the well-separated full-wrap band.
func fatterOperand(a, b ruledOperand) (fat, thin ruledOperand, aIsFat, ok bool) {
	_, ca, _, okA := cylinderSideFace(a.body)
	_, cb, _, okB := cylinderSideFace(b.body)
	if !okA || !okB {
		return ruledOperand{}, ruledOperand{}, false, false
	}
	if ca.Radius >= cb.Radius {
		return a, b, true, true
	}
	return b, a, false, true
}

// nearPinchGate resolves a crossing to its two operands and imprint loops ONLY when it is the unequal-radius
// near-pinch band this file handles: two loops, radii unequal (equal is the Steinmetz constructor's), a narrow
// neck (nearPinchLoops), and the breach clear of BOTH operands' caps (so every cap survives whole, as the
// ordinary ruled cut/join require). ok=false otherwise, so the caller falls through to the ordinary pipeline.
func nearPinchGate(a, b *topo.Body, rec *diag.Recorder) (fat, thin ruledOperand, aIsFat bool, loops []geom.Curve3, ok bool) {
	loops, okL := crossingCylinderLoops(a, b, rec)
	if !okL || len(loops) != 2 || !unequalRadiusCrossing(a, b) || !nearPinchLoops(loops) {
		return ruledOperand{}, ruledOperand{}, false, nil, false
	}
	oa, okA := cylinderOperand(a)
	ob, okB := cylinderOperand(b)
	if !okA || !okB {
		return ruledOperand{}, ruledOperand{}, false, nil, false
	}
	fat, thin, aIsFat, okF := fatterOperand(oa, ob)
	if !okF || !loopsClearOfCaps(fat.newUV(Difference, false, thin.inside), loops) ||
		!loopsClearOfCaps(thin.newUV(Difference, false, fat.inside), loops) {
		return ruledOperand{}, ruledOperand{}, false, nil, false
	}
	return fat, thin, aIsFat, loops, true
}

// nearPinchCrossingCut builds target − tool for an unequal-radius NEAR-PINCH crossing (below the #1781 gate),
// where the ordinary (u,v) arrangement fuses the two lens holes. Every near-pinch face is built from the raw
// whole imprint loops so each shared loop welds as ONE topo edge: the fat wall is the corridor-seeded keyhole
// holed tube (fat − thin) or its two per-loop lens caps (thin − fat), and the thin wall is the arrangement
// tunnel band (which already welds to whole loops) or the raw two-rim stubs. A global curvedReorient fixes the
// mixed-source winding. ok=false when the pair is not the near-pinch band (the ordinary path then runs). The
// caller's validBooleanSolid gate is the fail-safe: an unforeseen degenerate result declines to CSG (#1818).
func nearPinchCrossingCut(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	fat, thin, _, loops, ok := nearPinchGate(target, tool, rec)
	if !ok {
		return nil, false
	}
	tgt, tl := fat, thin
	if fat.body != target {
		tgt, tl = thin, fat
	}
	imprint := loops
	var keptT, keptL []curvedFace
	var okT, okL bool
	if tgt.body == fat.body { // fat − thin: keyhole holed fat wall + reversed thin tunnel band
		keptT, okT = fatNearPinchWall(fat, loops, Difference, false, tl.inside)
		keptL, okL = tl.split(imprint, Difference, true, tgt.inside)
	} else { // thin − fat: raw thin stubs + reversed fat lens caps
		keptT, okT = rawStubBands(tgt, loops)
		keptL, okL = fatNearPinchWall(fat, loops, Difference, true, tgt.inside)
	}
	if !okT || !okL {
		return nil, false
	}
	return curvedStitch(curvedReorient(cutFaces(tgt, keptT, tl, keptL))), true
}

// nearPinchCrossingJoin builds a ∪ b for an unequal-radius near-pinch crossing: the corridor-seeded keyhole
// holed fat wall + the two raw thin stubs + every cap, no reversal (a union keeps all walls facing out). Same
// raw-whole-loop construction and reorient as the cut. ok=false outside the near-pinch band (#1818).
func nearPinchCrossingJoin(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	fat, thin, aIsFat, loops, ok := nearPinchGate(a, b, rec)
	if !ok {
		return nil, false
	}
	wallFat, okWF := fatNearPinchWall(fat, loops, Union, !aIsFat, thin.inside)
	wallThin, okWT := rawStubBands(thin, loops)
	if !okWF || !okWT {
		return nil, false
	}
	return curvedStitch(curvedReorient(joinFaces(fat, wallFat, thin, wallThin))), true
}
