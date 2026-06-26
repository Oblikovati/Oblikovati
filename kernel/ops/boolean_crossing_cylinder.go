// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
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
func curvedCrossingIntersect(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.CrossingCylinderIntersect(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeCylinderIntersect returns the exact intersection of a cone crossing a cylinder (the cone band
// plus two cylinder-wall lens caps), or ok=false to defer. Only Intersect maps here, and only a valid closed
// manifold result is adopted.
func curvedConeCylinderIntersect(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.ConeCylinderIntersect(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeConeIntersect returns the exact intersection of a cone crossing a fatter cone (the rod-cone band
// plus two fat-cone lens caps), or ok=false to defer. Only Intersect maps here, and only a valid closed
// manifold result is adopted.
func curvedConeConeIntersect(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.ConeConeIntersect(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedSteinmetzIntersect returns the exact intersection of two equal-radius perpendicular cylinders (the
// Steinmetz bicylinder), or ok=false to defer. Only Intersect maps here, and only a valid closed manifold
// result is adopted. The SSI tracer now traces this equal-radius pinch (#1404), but the four-lobe bicylinder
// solid is still built from exact analytic ellipse edges here; unifying it onto the traced loops is #1403.
func curvedSteinmetzIntersect(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
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
func curvedPartialIntersect(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	res, ok := brep.PartialPenetrationIntersect(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedPartialCut returns target − tool when tool is a thin rod ending inside the fatter target (a blind
// hole), or ok=false to defer. Only Cut maps here, and only a valid closed manifold result is adopted.
func curvedPartialCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.PartialPenetrationCut(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedPartialJoin returns target ∪ tool for a thin rod ending inside the fatter cylinder (the fat with a
// single rod stub sticking out the entry side), or ok=false to defer. Only Join maps here, and only a valid
// closed manifold result is adopted.
func curvedPartialJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.PartialPenetrationJoin(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCylindricalHoleCut returns target − tool when the tool is a straight cylinder drilling a clean
// through-hole in an all-planar target (a drilled plate), or ok=false to defer. The result is an EXACT
// curved B-rep — the two pierced faces gain a circular hole and a single geom.Cylinder face forms the
// hole wall — so the most common curved cut no longer degrades to triangle-soup CSG (M2 Phase 3,
// Oblikovati/Oblikovati#1336, reverse of the #1334 cylinder − box case). Only Cut maps here; a tool that
// is not a single full cylinder, or a hole that clips a face or does not pass clean through, returns
// ok=false so the CSG fallback still runs.
func curvedCylindricalHoleCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.DrillThroughHole(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCoaxialJoin returns target ∪ tool when both are coaxial equal-radius cylinders overlapping or
// abutting along the axis — one cylinder spanning their merged extent — or ok=false to defer. Their side
// faces are coincident (the curved analogue of a coplanar overlap), which the CSG fallback faceted; the
// exact union keeps the analytic cylinder. Only Join maps here, and only a valid closed manifold result is
// adopted.
func curvedCoaxialJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.CoaxialCylinderUnion(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedCylinderBossJoin returns target ∪ tool when tool is a cylinder seated flush and perpendicular on
// one planar face of target, protruding outward (a boss / spigot), or ok=false to defer. The boss base
// disk is coplanar with the seat face — the canonical coplanar overlap the CSG fallback faceted; the exact
// union keeps the analytic cylinder wall. Only Join maps here, and only a valid closed manifold result is
// adopted.
func curvedCylinderBossJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.JoinCylindricalBoss(target, tool)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeCylinderCut returns target − tool for a cone crossing a cylinder (drilling the fat cylinder with
// the cone, or the two cone stubs of cone − fat), or ok=false to defer. Only Cut maps here, and only a valid
// closed manifold result is adopted.
func curvedConeCylinderCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.ConeCylinderCut(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeCylinderJoin returns target ∪ tool for a cone crossing a cylinder (a fat cylinder side-breached
// by a cone passing through it, leaving a tapered stub each side), or ok=false to defer. Only Join maps here,
// and only a valid closed manifold result is adopted.
func curvedConeCylinderJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.ConeCylinderJoin(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeConeCut returns target − tool for a cone crossing a fatter cone (drilling the fat cone with the
// rod cone, or the two rod-cone stubs of cone − fat), or ok=false to defer. Only Cut maps here, and only a
// valid closed manifold result is adopted.
func curvedConeConeCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.ConeConeCut(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedConeConeJoin returns target ∪ tool for a cone crossing a fatter cone (the fat cone side-breached by a
// rod cone passing through it, leaving a tapered stub each side), or ok=false to defer. Only Join maps here,
// and only a valid closed manifold result is adopted.
func curvedConeConeJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.ConeConeJoin(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedSteinmetzCut returns target − tool for two equal-radius perpendicular cylinders (the target with
// the tool's saddle bite removed), or ok=false to defer. Only Cut maps here, and only a valid closed
// manifold result is adopted.
func curvedSteinmetzCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
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
func curvedCrossingCut(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Cut {
		return nil, false
	}
	res, ok := brep.CrossingCylinderCut(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}

// curvedSteinmetzJoin returns target ∪ tool for two equal-radius perpendicular cylinders (the union of two
// crossing cylinders of the same radius), or ok=false to defer. Only Join maps here, and only a valid closed
// manifold result is adopted.
func curvedSteinmetzJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
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
func curvedCrossingJoin(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if op != Join {
		return nil, false
	}
	res, ok := brep.CrossingCylinderJoin(target, tool, rec)
	if !ok || !validBooleanSolid(res) {
		return nil, false
	}
	return res, true
}
