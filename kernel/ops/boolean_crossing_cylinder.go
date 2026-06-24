// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
)

// Exact crossing-cylinder intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). Wires
// brep.CrossingCylinderIntersect into the boolean: a rod ∩ a fatter cylinder builds straight from the
// traced imprint loops into a watertight analytic solid (rod band + two fat-wall lens caps), keeping the
// exact cylinder surfaces instead of triangle-soup CSG. Anything outside the thin-through-fat case (a
// non-cylinder operand, equal radii, a partial penetration) returns ok=false so booleanGeneral keeps its
// CSG fallback — no regression.

// curvedCrossingIntersect returns the exact intersection of two crossing cylinders, or ok=false to defer.
// Only Intersect maps here, and only a valid closed manifold result is adopted (an inside-out or open
// assembly is rejected so the fallback runs instead).
func curvedCrossingIntersect(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.CrossingCylinderIntersect(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCrossingCut returns target − tool for two crossing cylinders (drilling a fat cylinder with a
// crossing rod), or ok=false to defer. Only Cut maps here, and only a valid closed manifold result is
// adopted.
func curvedCrossingCut(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.CrossingCylinderCut(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}
