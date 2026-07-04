// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-junction assembly (EPIC Oblikovati/Oblikovati#1738, ADR-0048). The exact triple points (curved_
// corner_junction.go) are the shared authority; this file resolves them once and drives the three coupled
// faces — target cylinder wall, rod tunnel wall, bitten notch cap — through the SAME trimByImprint pipeline,
// each split at those points, so curvedStitch welds them watertight. It fires only on the two-triple-point
// TRANSVERSAL crossing that the disjoint gate (disjointFromPrior) declines today; every other config
// (tangency, a higher-order junction, a removed bottom-circle anchor) returns ok=false and keeps the
// observable CSG decline, so the arrangement golden stays byte-identical (ADR-0048 §consequences).

// cornerCutSetup is the resolved corner-junction seam: the operands, the shared triple points and rod∩notch
// arc, and the imprint/prior pre-split at those points that every coupled face consumes.
type cornerCutSetup struct {
	target, tool *topo.Body
	tgtFace      curvedFace
	tgtCyl       geom.Cylinder
	band         coneSideBand_
	toolOp       ruledOperand
	rodCyl       geom.Cylinder
	notchArc     geom.EllipticalArc
	splitPrior   priorTrimLoop
	splitImprint []geom.Curve3
	tgtInside    func(math.Point3) bool
}

// PartialRimCornerCutGeneral builds target − tool for the corner-junction partial-rim case: a second curved
// cut whose SSI imprint CROSSES the surviving first-cut notch boundary (#1738). It resolves the exact triple
// points where the target cylinder, the rod surface and the notch plane meet, pre-splits every coupled face
// there, and stitches a watertight analytic solid. ok=false (→ observable CSG decline) unless the config is a
// clean two-triple-point transversal crossing of a recognisable notched cylinder by a bare rod.
func PartialRimCornerCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	s, ok := setupCornerCut(target, tool, rec)
	if !ok {
		return nil, false
	}
	return curvedStitch(assembleCornerFaces(s)), true
}

// setupCornerCut resolves the seam: it traces the imprint, recognises the notched target and bare rod,
// solves the triple points, runs the transversal gate, and pre-splits the prior and imprint at the points.
func setupCornerCut(target, tool *topo.Body, rec *diag.Recorder) (cornerCutSetup, bool) {
	loops, okL := require2Loops(partialRimImprint(target, tool, rec))
	f, cyl, band, prior, okT := cutCylinderSideFace(target)
	toolOp, okO := ruledOperandOf(tool)
	rodCyl, _, _, okR := cylinderSolidParams(facesOfAny(tool))
	if !okL || !okT || !okO || !okR {
		return cornerCutSetup{}, false
	}
	js := cornerJunctions(prior, rodCyl.Origin, rodCyl.AxisDir.AsVector(), rodCyl.Radius)
	notch, okN := recoverNotchPlane(prior)
	if !okN || !cornerJunctionTransversal(js, prior, cyl, rodCyl) {
		return cornerCutSetup{}, false
	}
	res := geom.ResolutionForBox(target.RangeBox().Union(tool.RangeBox()))
	arc, okA := rodNotchArc(notch, rodCyl, res, js, cyl.Origin, cyl.AxisDir.AsVector(), cyl.Radius)
	if !okA {
		return cornerCutSetup{}, false
	}
	return cornerCutSetup{
		target: target, tool: tool, tgtFace: f, tgtCyl: cyl, band: band, toolOp: toolOp, rodCyl: rodCyl,
		notchArc: arc, splitPrior: splitPriorAtJunctions(prior, js), splitImprint: splitImprintAtJunctions(loops, js),
		tgtInside: cutCylinderSolidMembership(target, cyl, band),
	}, true
}

// cornerAngFloor is the fixed dimensionless angular floor of the tangency-decline gate — a crossing shallower
// than this is not worth trusting even with exact arithmetic. Set slightly high because a false accept ships a
// wrong solid while a false decline only falls back to observable CSG (ADR-0048 §tangency, "bias to decline").
const cornerAngFloor = 0.02

// cornerJunctionTransversal is the scale-invariant tangency gate (ADR-0048): it accepts ONLY a clean pair of
// transversal triple points. It rejects any count other than two (a tangency degenerates to a single double
// root; a higher-order junction gives more), and rejects a pair where either the two SURFACES graze
// (‖n_target × n_rod‖→0, a singular SSI) or the two boundary CURVES graze (sinθ_metric→0, the corner cusp).
// The angle is the genuine 3D tangent-plane angle, which on a developable cylinder already equals the metric
// angle in I=diag(R²,1) — scale-invariant with no R weighting. The threshold is ε_effective=max(ε_ang,
// τ/h_local): a fixed angular floor OR the backward-error resolution ratio, whichever is larger.
func cornerJunctionTransversal(js []cornerJunction, prior priorTrimLoop, tgt, rod geom.Cylinder) bool {
	if len(js) != 2 {
		return false
	}
	hLocal := float64(js[0].point.DistanceTo(js[1].point))
	tau := geom.ResolutionForSize(hLocal + tgt.Radius).Stitch()
	epsEff := cornerAngFloor
	if hLocal > 0 {
		epsEff = stdmath.Max(cornerAngFloor, tau/hLocal)
	}
	for _, j := range js {
		surfSurf, curveCurve := junctionDegeneracy(j, prior, tgt.Origin, tgt.AxisDir.AsVector(), rod.Origin, rod.AxisDir.AsVector())
		if surfSurf < cornerAngFloor || curveCurve < epsEff {
			return false
		}
	}
	return true
}

// assembleCornerFaces builds the result's face set: the target's breached wall + its caps (the notch cap
// bitten by the rod, the others whole) + the rod's tunnel wall and any blind cap reversed into the cavity —
// the corner-junction analogue of cutFaces, differing only in that three faces are split at the seam.
func assembleCornerFaces(s cornerCutSetup) []curvedFace {
	faces := append([]curvedFace{}, cornerTargetWall(s)...)
	faces = append(faces, cornerCaps(s)...)
	faces = append(faces, reverseCurvedFaces(cornerToolTunnel(s))...)
	return append(faces, reverseCurvedFaces(capsInside(s.tool, s.tgtInside))...)
}

// cornerTargetWall trims the target cylinder side by the pre-split imprint, composing the pre-split prior
// notch as constraint edges — the same cutCylinderUV path the disjoint case uses, now fed shared triple-point
// vertices so its imprint↔prior junctions weld exactly.
func cornerTargetWall(s cornerCutSetup) []curvedFace {
	c := newCutCylinderUVSolid(s.tgtCyl, s.band, s.splitPrior, Difference, false, s.toolOp.inside)
	c.placeSeams(s.splitImprint)
	faces, _, _ := trimByImprint(&c, s.tgtFace, s.tgtCyl, s.splitImprint, cutCylinderMaterial(&c))
	return faces
}

// cornerToolTunnel trims the rod side by its own imprint PLUS the rod∩notch arc, so the tunnel wall is cut by
// the notch at the same triple points (the rod's membership already drops cells above the notch; it only
// lacked the arrangement edge to follow — ADR-0048 §tool-side weld).
func cornerToolTunnel(s cornerCutSetup) []curvedFace {
	imprint := append(append([]geom.Curve3{}, s.splitImprint...), s.notchArc)
	faces, _ := s.toolOp.split(imprint, Difference, true, s.tgtInside)
	return faces
}

// cornerCaps returns the target's outward caps with the rod-crossed notch cap replaced by its bitten form and
// every other cap kept whole — biteNotchCap returns ok only for a cap the rod actually crosses.
func cornerCaps(s cornerCutSetup) []curvedFace {
	var out []curvedFace
	for _, cap := range capsOutside(s.target, s.toolOp.inside) {
		if bitten, ok := biteNotchCap(cap, s.notchArc, s.rodCyl.Origin, s.rodCyl.AxisDir.AsVector(), s.rodCyl.Radius); ok {
			out = append(out, bitten)
		} else {
			out = append(out, cap)
		}
	}
	return out
}
