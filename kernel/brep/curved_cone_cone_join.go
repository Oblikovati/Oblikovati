// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/topo"

// Cone–cone join (M2 Phase 2, Oblikovati/Oblikovati#1335). The union of a cone (a tapered rod) passing right
// through a fatter cone: one connected solid — the fat cone's two caps and its holed side wall (the two lens
// holes), plus a rod-cone STUB protruding from each lens hole out to the rod cone's own end cap. It reuses
// the crossing/cone–cylinder join builder (joinFatAndRod) unchanged now that both the rod and the fat are
// crossRods: each stub band lofts through the ruled saddle-band path, the fat-cone wall meshes through the
// holed-cone-wall unroll, and the frustum ends carry their own radii.

// ConeConeJoin builds fat ∪ cone for a cone crossing a fatter cone (the fat cone with a tapered stub each
// side), or ok=false to defer (a non-cone operand, the cone–cylinder case, a rod cone that does not fully
// cross, or a breach that reaches a fat cap) so kernel/ops keeps its CSG fallback.
//
// Example — a radius-0.8→1.5 frustum on x joined with a crossing radius-2→4 frustum on z gives the fat cone
// with two tapered stubs:
//
//	thin, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 0.8, 1.5, "thin")
//	fat, _  := brep.SolidCylinderCone(math.P3(0,0,-6), math.P3(0,0,6), 2, 4, "fat")
//	res, ok := brep.ConeConeJoin(fat, thin)
func ConeConeJoin(a, b *topo.Body) (*topo.Body, bool) {
	p, _, ok := coneConeCrossingPartsOf(a, b)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // a stub would breach a fat cap, not the side wall: out of scope
	}
	return joinFatAndRod(p), true
}
