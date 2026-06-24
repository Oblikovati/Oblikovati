// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Cone–cone intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). The split/classify/stitch stage after the
// cone–cone imprint: a cone (or frustum) crossing a fatter cone, like a tapered rod through a tapered tube.
// The result is the same three-face shape as the crossing-cylinder and cone–cylinder intersections — the
// rod-wall BAND inside the fat (between the two loops) plus the two fat-wall LENS caps — only here BOTH the
// rod and the fat are CONES. Because the loop classification works off each operand's axis (crossAxis) and
// the stitch takes the rod and fat as geom.Surface, the cones reuse the whole pipeline: the rod cone band
// lofts through the saddle-band path (a cone is ruled along its slant), and each lens cap is a trimmed patch
// of the fat cone (a cone is developable, like a cylinder, so the metric-patch mesher handles it).

// ConeConeIntersect builds the exact intersection of a cone crossing a fatter cone (the rod-cone band inside
// the fat plus the two fat-cone lens caps), or ok=false when the configuration is outside the wired
// cone-through-cone case (not exactly two imprint loops, neither cone is the rod both loops encircle, or a
// loop lies beyond the rod cone's caps) so kernel/ops keeps its fallback.
//
// Example — a radius-0.8→1.5 frustum on x crossing a radius-2→4 frustum on z gives a three-face solid:
//
//	thin, _ := brep.SolidCylinderCone(math.P3(-6,0,0), math.P3(6,0,0), 0.8, 1.5, "thin")
//	fat, _  := brep.SolidCylinderCone(math.P3(0,0,-6), math.P3(0,0,6), 2, 4, "fat")
//	res, ok := brep.ConeConeIntersect(thin, fat) // rod-cone band capped by two fat-cone lenses
func ConeConeIntersect(a, b *topo.Body) (*topo.Body, bool) {
	loops, ok := coneConeImprint(a, b)
	if !ok || len(loops) != 2 {
		return nil, false
	}
	rod, fat, rodVMin, rodVMax, ok := rodAndFatCones(a, b, loops)
	if !ok || !loopsSpanCone(rod, rodVMin, rodVMax, loops) {
		return nil, false // a loop beyond a rod-cone cap is a partial penetration, not a full crossing
	}
	lo, hi, ok := assignRimLoops(coneAxis(rod), coneAxis(fat), loops)
	if !ok {
		return nil, false
	}
	return stitchRodWithSaddleCaps(rod, fat, lo, hi), true
}

// rodAndFatCones picks which cone is the rod (the band-bearing one both imprint loops fully encircle) and
// which is the fat cone (the lens-capped one), carrying the rod cone's apex-distance band [vMin, vMax]
// through. ok=false unless both bodies are bare cones and exactly one is the rod the loops encircle.
func rodAndFatCones(a, b *topo.Body, loops []geom.Polyline) (rod, fat geom.Cone, rodVMin, rodVMax float64, ok bool) {
	ca, aMin, aMax, okA := coneSolidParams(facesOfAny(a))
	cb, bMin, bMax, okB := coneSolidParams(facesOfAny(b))
	if !okA || !okB {
		return geom.Cone{}, geom.Cone{}, 0, 0, false
	}
	aWraps, bWraps := allLoopsEncircle(loops, coneAxis(ca)), allLoopsEncircle(loops, coneAxis(cb))
	if aWraps && !bWraps {
		return ca, cb, aMin, aMax, true
	}
	if bWraps && !aWraps {
		return cb, ca, bMin, bMax, true
	}
	return geom.Cone{}, geom.Cone{}, 0, 0, false
}
