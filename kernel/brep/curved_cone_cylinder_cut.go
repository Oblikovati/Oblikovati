// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone–cylinder cut (M2 Phase 2, Oblikovati/Oblikovati#1335). The two subtraction outcomes when a cone (a
// tapered rod) crosses a fatter cylinder, assembled from the cone–cylinder imprint loops exactly like the
// crossing-cylinder drill (curved_crossing_cut.go) and sharing its builders — the rod is just a cone now (a
// crossRod), so the cone band lofts through the same ruled saddle-band path:
//
//   - Cut(fat, cone): the fat cylinder drilled by the cone — its two caps, its side wall carrying the two
//     lens holes, and the cone-wall band flipped inward as the tapered tunnel (drillFatWithRod).
//   - Cut(cone, fat): the cone − the fat — the two disconnected tapered stubs sticking out either side of the
//     fat, each a closed lump (a cone-wall band, the cone's end cap, and the fat-wall lens), merged into one
//     multi-shell body (cutRodMinusFat).

// ConeCylinderCut returns target − tool for a cone crossing a cylinder, or ok=false to defer (a non-cone/non-
// cylinder operand, equal-axis cylinders, a breach reaching a fat cap, or a cone that does not fully cross)
// so kernel/ops keeps its CSG fallback. The direction is read from which body is the cone: drilling the fat
// when the target is the cylinder, the two cone stubs when the target is the cone.
//
// Example — a radius-1→2.5 frustum on x crossing (and drilling) a radius-3 cylinder on z:
//
//	cone, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 1, 2.5, "cone")
//	cyl, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.ConeCylinderCut(cyl, cone) // fat with a tapered tunnel
//	res, ok = brep.ConeCylinderCut(cone, cyl)  // the two cone stubs
func ConeCylinderCut(target, tool *topo.Body) (*topo.Body, bool) {
	p, ok := coneCrossingPartsOf(target, tool)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // the breach reaches a fat cap (not a clean side breach): out of scope
	}
	if _, _, _, okCone := coneSolidParams(facesOfAny(target)); okCone {
		return cutRodMinusFat(p), true // target is the cone rod: two disconnected tapered stubs
	}
	return drillFatWithRod(p), true // target is the fat cylinder: drill a tapered tunnel
}

// coneCrossingPartsOf resolves a cone body and a cylinder body into crossingParts (with the cone as the rod),
// or ok=false when they are not one bare cone crossing one fatter cylinder in the full-crossing configuration
// (exactly two imprint loops, the cone the rod both loops encircle, both loops between the cone's caps).
func coneCrossingPartsOf(a, b *topo.Body) (*crossingParts, bool) {
	loops, ok := coneCylinderImprint(a, b)
	if !ok || len(loops) != 2 {
		return nil, false
	}
	cone, vMin, vMax, cyl, fatBase, fatH, ok := coneCrossingGeom(a, b)
	if !ok || !allLoopsEncircle(loops, coneAxis(cone)) || allLoopsEncircle(loops, cylAxis(cyl)) {
		return nil, false // the cone must be the rod both loops encircle, the cylinder the lens-capped fat
	}
	if !loopsSpanCone(cone, vMin, vMax, loops) {
		return nil, false // a loop beyond a cone cap is a partial penetration, not a full crossing
	}
	lo, hi, ok := assignRimLoops(coneAxis(cone), cylAxis(cyl), loops)
	if !ok {
		return nil, false
	}
	rodBase := cone.Apex.TranslateBy(cone.AxisDir.AsVector().Scale(vMin))
	return &crossingParts{coneRod{cone}, cylinderRod{cyl}, rodBase, fatBase, vMax - vMin, fatH, lo, hi}, true
}

// coneCrossingGeom resolves the cone (with its apex-distance band [vMin, vMax]) and the fat cylinder (with its
// cap base and height) from two bodies, in either order. ok=false unless one is a bare cone and the other a
// bare cylinder.
func coneCrossingGeom(a, b *topo.Body) (cone geom.Cone, vMin, vMax float64, cyl geom.Cylinder, fatBase math.Point3, fatHeight float64, ok bool) {
	coneBody, cylBody := a, b
	if _, _, _, okCone := coneSolidParams(facesOfAny(a)); !okCone {
		coneBody, cylBody = b, a
	}
	cn, lo, hi, okCone := coneSolidParams(facesOfAny(coneBody))
	cy, base, h, okCyl := cylinderSolidParams(facesOfAny(cylBody))
	if !okCone || !okCyl {
		return geom.Cone{}, 0, 0, geom.Cylinder{}, math.Point3{}, 0, false
	}
	return cn, lo, hi, cy, base, h, true
}
