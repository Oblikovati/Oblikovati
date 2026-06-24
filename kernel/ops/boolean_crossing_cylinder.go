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

// curvedConeCylinderIntersect returns the exact intersection of a cone crossing a cylinder (the cone band
// plus two cylinder-wall lens caps), or ok=false to defer. Only Intersect maps here, and only a valid closed
// manifold result is adopted.
func curvedConeCylinderIntersect(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.ConeCylinderIntersect(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedSteinmetzIntersect returns the exact intersection of two equal-radius perpendicular cylinders (the
// Steinmetz bicylinder), or ok=false to defer. Only Intersect maps here, and only a valid closed manifold
// result is adopted. This is the equal-radius case the SSI imprint tracer cannot trace (it pinches), fitted
// analytically as two crossing ellipses instead.
func curvedSteinmetzIntersect(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.EqualRadiusSteinmetzIntersect(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedPartialIntersect returns the exact intersection of a thin rod ending inside a fatter cylinder (the
// rod plug), or ok=false to defer. Only Intersect maps here, and only a valid closed manifold result is
// adopted. This is the partial-penetration case the imprint traces as a single loop (one wall breach).
func curvedPartialIntersect(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.PartialPenetrationIntersect(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedPartialCut returns target − tool when tool is a thin rod ending inside the fatter target (a blind
// hole), or ok=false to defer. Only Cut maps here, and only a valid closed manifold result is adopted.
func curvedPartialCut(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.PartialPenetrationCut(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedPartialJoin returns target ∪ tool for a thin rod ending inside the fatter cylinder (the fat with a
// single rod stub sticking out the entry side), or ok=false to defer. Only Join maps here, and only a valid
// closed manifold result is adopted.
func curvedPartialJoin(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.PartialPenetrationJoin(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedSteinmetzCut returns target − tool for two equal-radius perpendicular cylinders (the target with
// the tool's saddle bite removed), or ok=false to defer. Only Cut maps here, and only a valid closed
// manifold result is adopted.
func curvedSteinmetzCut(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.EqualRadiusSteinmetzCut(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCrossingCut returns target − tool for two crossing cylinders (drilling a fat cylinder with a
// crossing rod, or the two rod stubs of rod − fat), or ok=false to defer. Only Cut maps here, and only a
// valid closed manifold result is adopted.
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

// curvedSteinmetzJoin returns target ∪ tool for two equal-radius perpendicular cylinders (the union of two
// crossing cylinders of the same radius), or ok=false to defer. Only Join maps here, and only a valid closed
// manifold result is adopted.
func curvedSteinmetzJoin(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.EqualRadiusSteinmetzJoin(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCrossingJoin returns target ∪ tool for two crossing cylinders (a fat cylinder side-breached by a
// rod passing through it, leaving a stub each side), or ok=false to defer. Only Join maps here, and only a
// valid closed manifold result is adopted.
func curvedCrossingJoin(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.CrossingCylinderJoin(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}
