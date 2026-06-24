// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/topo"

// Cone–cylinder join (M2 Phase 2, Oblikovati/Oblikovati#1335). The union of a cone (a tapered rod) passing
// right through a fatter cylinder: one connected solid — the fat's two caps and its holed side wall (the two
// lens holes), plus a cone-wall STUB protruding from each lens hole out to the cone's own end cap. It reuses
// the crossing-cylinder join builder (joinFatAndRod) unchanged: the rod is just a cone (a crossRod), so each
// stub band lofts through the same ruled saddle-band path and the frustum's two ends carry their own radii.

// ConeCylinderJoin builds fat ∪ cone for a cone crossing a cylinder (the fat with a tapered stub each side),
// or ok=false to defer (a non-cone/non-cylinder operand, a cone that does not fully cross, or a breach that
// reaches a fat cap) so kernel/ops keeps its CSG fallback.
//
// Example — a radius-1→2.5 frustum on x joined with a crossing radius-3 cylinder on z gives the fat with two
// tapered stubs:
//
//	cone, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 1, 2.5, "cone")
//	cyl, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.ConeCylinderJoin(cyl, cone)
func ConeCylinderJoin(a, b *topo.Body) (*topo.Body, bool) {
	p, ok := coneCrossingPartsOf(a, b)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // a stub would breach a fat cap, not the side wall: out of scope
	}
	return joinFatAndRod(p), true
}
