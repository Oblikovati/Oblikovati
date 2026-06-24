// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone–cone cut (M2 Phase 2, Oblikovati/Oblikovati#1335). The two subtraction outcomes when a cone (a
// tapered rod) crosses a fatter cone, assembled from the cone–cone imprint loops exactly like the
// crossing-cylinder and cone–cylinder drills (curved_crossing_cut.go) — both the rod AND the fat are now
// cones (crossRods), so the fat's caps take the cone's radius at each end and its holed side wall is a holed
// CONE wall (a cone is developable, so the unroll-and-CDT mesher meshes it):
//
//   - Cut(fat, cone): the fat cone drilled by the rod cone — its two caps, its side wall carrying the two
//     lens holes, and the rod-cone band flipped inward as the tapered tunnel (drillFatWithRod).
//   - Cut(cone, fat): the rod cone − the fat cone — the two disconnected tapered stubs, each a closed lump
//     (a rod-cone band, the rod's end cap, and the fat-cone lens), merged into one multi-shell body
//     (cutRodMinusFat).

// ConeConeCut returns target − tool for a cone crossing a fatter cone, or ok=false to defer (a non-cone
// operand, the cone–cylinder case, a breach reaching a fat cap, or a rod cone that does not fully cross) so
// kernel/ops keeps its CSG fallback. The direction is read from which cone is the rod the loops encircle:
// drilling the fat when the target is the fat cone, the two rod stubs when the target is the rod cone.
//
// Example — a radius-0.8→1.5 frustum on x crossing (and drilling) a radius-2→4 frustum on z:
//
//	thin, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 0.8, 1.5, "thin")
//	fat, _  := brep.SolidCylinderCone(math.P3(0,0,-6), math.P3(0,0,6), 2, 4, "fat")
//	res, ok := brep.ConeConeCut(fat, thin) // fat cone with a tapered tunnel
//	res, ok = brep.ConeConeCut(thin, fat)  // the two rod-cone stubs
func ConeConeCut(target, tool *topo.Body) (*topo.Body, bool) {
	p, targetIsRod, ok := coneConeCrossingPartsOf(target, tool)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // the breach reaches a fat cap (not a clean side breach): out of scope
	}
	if targetIsRod {
		return cutRodMinusFat(p), true // target is the rod cone: two disconnected tapered stubs
	}
	return drillFatWithRod(p), true // target is the fat cone: drill a tapered tunnel
}

// coneConeCrossingPartsOf resolves two cone bodies into crossingParts (both rod and fat as coneRods), also
// reporting whether the FIRST body (the Cut target) is the rod cone. ok=false when they are not one cone
// crossing one fatter cone in the full-crossing configuration (exactly two imprint loops, one cone the rod
// both loops encircle, both loops between the rod cone's caps).
func coneConeCrossingPartsOf(a, b *topo.Body) (parts *crossingParts, aIsRod, ok bool) {
	loops, imOK := coneConeImprint(a, b)
	if !imOK || len(loops) != 2 {
		return nil, false, false
	}
	rod, fat, rodVMin, rodVMax, fatVMin, fatVMax, firstIsRod, rfOK := rodAndFatCones(a, b, loops)
	if !rfOK || !loopsSpanCone(rod, rodVMin, rodVMax, loops) {
		return nil, false, false // a loop beyond a rod-cone cap is a partial penetration, not a crossing
	}
	lo, hi, asOK := assignRimLoops(coneAxis(rod), coneAxis(fat), loops)
	if !asOK {
		return nil, false, false
	}
	rodBase := rod.Apex.TranslateBy(rod.AxisDir.AsVector().Scale(math.Scalar(rodVMin)))
	fatBase := fat.Apex.TranslateBy(fat.AxisDir.AsVector().Scale(math.Scalar(fatVMin)))
	parts = &crossingParts{coneRod{rod}, coneRod{fat}, rodBase, fatBase, rodVMax - rodVMin, fatVMax - fatVMin, lo, hi}
	return parts, firstIsRod, true
}
