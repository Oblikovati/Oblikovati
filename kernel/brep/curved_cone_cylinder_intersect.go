// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Cone–cylinder intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). The split/classify/stitch stage
// after the cone–cylinder imprint: a cone (or frustum) crossing a fatter cylinder, like a tapered rod
// through a fat tube. The result is the same three-face shape as the crossing-cylinder intersection — the
// rod-wall BAND inside the fat (between the two loops) plus the two fat-wall LENS caps — only here the rod
// is a CONE. Because the band loft and the loop classification work off the rod's AXIS (not its surface
// type, see crossAxis), the cone reuses the whole crossing-cylinder pipeline: the cone is the rod (both
// loops encircle its axis), the cylinder the fat, and the cone band lofts through the same saddle-band path
// (a cone is ruled along its slant, exactly like a cylinder).

// ConeCylinderIntersect builds the exact intersection of a cone crossing a cylinder (the cone band inside
// the cylinder plus the two cylinder-wall lens caps), or ok=false when the configuration is outside the
// wired cone-through-cylinder case (not exactly two imprint loops, the cone is not the rod both loops
// encircle, or a loop lies beyond the cone's caps) so kernel/ops keeps its fallback.
//
// Example — a radius-1→2.5 frustum on x crossing a radius-3 cylinder on z gives a three-face solid:
//
//	cone, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 1, 2.5, "cone")
//	cyl, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.ConeCylinderIntersect(cone, cyl) // cone band capped by two lenses
func ConeCylinderIntersect(a, b *topo.Body) (*topo.Body, bool) {
	cone, cyl, vMin, vMax, ok := coneAndCylinder(a, b)
	if !ok {
		return nil, false
	}
	loops, ok := coneCylinderImprint(a, b)
	if !ok || len(loops) != 2 {
		return nil, false
	}
	if !allLoopsEncircle(loops, coneAxis(cone)) || allLoopsEncircle(loops, cylAxis(cyl)) {
		return nil, false // the cone must be the rod the loops encircle, the cylinder the lens-capped fat
	}
	if !loopsSpanCone(cone, vMin, vMax, loops) {
		return nil, false // a loop beyond a cone cap: the cone does not fully cross (a partial penetration)
	}
	lo, hi, ok := assignRimLoops(coneAxis(cone), cylAxis(cyl), loops)
	if !ok {
		return nil, false
	}
	return stitchRodWithSaddleCaps(cone, cyl, lo, hi), true
}

// loopsSpanCone reports whether every imprint loop lies within the cone's finite apex-distance band
// [vMin, vMax] (between its caps). A loop beyond a cap means the cone ends inside the cylinder (a partial
// penetration), so the full-crossing assembler must defer.
func loopsSpanCone(cone geom.Cone, vMin, vMax float64, loops []geom.Polyline) bool {
	axis := cone.AxisDir.AsVector()
	for _, lp := range loops {
		for _, p := range lp.Vertices {
			v := float64(cone.Apex.VectorTo(p).Dot(axis))
			if v < vMin-1e-7 || v > vMax+1e-7 {
				return false
			}
		}
	}
	return true
}
