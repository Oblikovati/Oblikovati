// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Curved-boolean CAP-CROSSING with a CONE tool (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The cylinder-tool
// cap-crossing (curved_cap_crossing.go) enters the target wall once and exits ONE cap through an ellipse
// strictly inside the rim. A conical tool (a countersink/tapered pin) makes the SAME arrangement — its only
// differences are analytic: the wall entry is the cone∩cylinder imprint (coneCylinderImprint, not the
// cylinder∩cylinder one) and the exit section is the cone∩plane ellipse (geom.IntersectSurfacesAnalytic, not
// the r/|n·d| cylinder ellipse). Because capCrossPlan and buildCapCrossCut are operand-generic over
// ruledOperand, the whole assembly (wall with entry hole + holed exit cap + reversed cone tunnel) is REUSED;
// only the recognizer changes. Non-ellipse exit sections (a parabola/hyperbola — the tool grazes the cap) and
// rim-crossing cone exits are out of this slice and decline to the CSG fallback.

// ConeCapCrossingCutGeneral is the exported entry kernel/ops routes a cone-tool cap-crossing subtract
// through. ok=false outside the recognised interior-exit slice so kernel/ops keeps its CSG fallback.
func ConeCapCrossingCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	plan, ok := classifyConeCapCross(target, tool, rec)
	if !ok {
		return nil, false
	}
	return buildCapCrossCut(plan)
}

// classifyConeCapCross recognises a cone tool making an interior-exit cap-crossing on a cylinder target, or
// ok=false otherwise. Positive gate: target a cylinder, tool a cone/frustum; the tool exits EXACTLY ONE cap
// through an ellipse strictly inside that cap's rim; and the cone∩wall imprint is EXACTLY ONE closed in-band
// loop (the single wall-entry hole, clear of the caps).
func classifyConeCapCross(target, tool *topo.Body, rec *diag.Recorder) (capCrossPlan, bool) {
	tgtOp, ok1 := cylinderOperand(target)
	toolOp, ok2 := coneOperand(tool)
	if !ok1 || !ok2 {
		return capCrossPlan{}, false
	}
	cone, ok := toolOp.surface.(geom.Cone)
	if !ok {
		return capCrossPlan{}, false
	}
	exitCap, ellipse, ok := coneInteriorExitCap(target, cone)
	if !ok {
		return capCrossPlan{}, false
	}
	traced, ok := coneCylinderImprint(target, tool, rec)
	if !ok {
		return capCrossPlan{}, false
	}
	// A long frustum's wall meets the target cylinder's INFINITE-surface trace both where it really enters the
	// finite wall AND where it would exit past the cap (a loop lying entirely BEYOND the exit cap plane). Keep
	// only the loops strictly inside the target's cap band; a loop that STRADDLES a cap level is a rim-crossing
	// (out of this interior-exit slice) and declines. See wallEntryLoops (#1724).
	entry, ok := wallEntryLoops(tgtOp.newUV(Difference, false, toolOp.inside), traced)
	if !ok || len(entry) != 1 {
		return capCrossPlan{}, false
	}
	return capCrossPlan{tgt: tgtOp, tool: toolOp, exitCap: exitCap, ellipse: ellipse, entry: entry}, true
}

// wallEntryLoops partitions traced imprint loops against the target side's cap band: it KEEPS the loops that
// sit strictly between the two cap levels (the genuine wall-entry holes) and DROPS loops lying entirely
// beyond a cap (phantom crossings of the target's infinite ruled surface past its finite wall — a long tool
// that reaches past the exit cap). ok=false if any loop STRADDLES a cap level, because a loop reaching a cap
// is a rim-crossing/cap-reaching breach this interior-exit slice does not build (it keeps its CSG fallback).
func wallEntryLoops(side ruledUV, loops []geom.Polyline) ([]geom.Polyline, bool) {
	margin := geom.ResolutionForSize(side.band.vMax - side.band.vMin).Plane()
	lo, hi := side.band.vMin+margin, side.band.vMax-margin
	var kept []geom.Polyline
	for _, lp := range loops {
		below, above := 0, 0
		for _, p := range lp.Vertices {
			v := float64(side.base.VectorTo(p).Dot(side.axis))
			switch {
			case v < lo:
				below++
			case v > hi:
				above++
			}
		}
		switch {
		case below == 0 && above == 0:
			kept = append(kept, lp) // strictly in-band: a real wall-entry hole
		case below == len(lp.Vertices) || above == len(lp.Vertices):
			continue // entirely beyond one cap: a phantom past the finite wall — drop it
		default:
			return nil, false // straddles a cap level: a rim-crossing this slice does not build
		}
	}
	return kept, true
}

// coneInteriorExitCap returns the single target cap the cone exits through an ellipse strictly inside that
// cap's rim, with that ellipse. ok=false unless EXACTLY ONE cap qualifies (zero → no cap exit; two → a
// two-cap exit, deferred).
func coneInteriorExitCap(target *topo.Body, cone geom.Cone) (curvedFace, geom.EllipseFull, bool) {
	var exitCap curvedFace
	var ellipse geom.EllipseFull
	found := 0
	for _, cap := range planarCapFaces(target) {
		e, ok := coneCapEllipseInsideRim(cap, cone)
		if !ok {
			continue
		}
		exitCap, ellipse, found = cap, e, found+1
	}
	return exitCap, ellipse, found == 1
}

// coneCapEllipseInsideRim returns the cone∩cap ellipse when cap is planar, the section is a bounded ELLIPSE,
// and that ellipse lies strictly inside cap's rim circle; ok=false otherwise (a non-planar cap, an open
// parabola/hyperbola section, or an ellipse reaching the rim — the latter a rim-crossing case, deferred).
func coneCapEllipseInsideRim(cap curvedFace, cone geom.Cone) (geom.EllipseFull, bool) {
	pl, ok := cap.surface.(geom.Plane)
	if !ok {
		return geom.EllipseFull{}, false
	}
	rim, ok := capRimCircle(cap)
	if !ok {
		return geom.EllipseFull{}, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(cone, pl, geom.ResolutionForSize(rim.Radius))
	if !handled || len(curves) != 1 {
		return geom.EllipseFull{}, false
	}
	ell, ok := curves[0].(geom.EllipseFull)
	if !ok {
		return geom.EllipseFull{}, false // an open section (parabola/hyperbola): not an interior cap exit
	}
	if !ellipseInsideRim(ell, rim.Center, rim.Radius, geom.ResolutionForSize(rim.Radius).Plane()) {
		return geom.EllipseFull{}, false
	}
	return ell, true
}
