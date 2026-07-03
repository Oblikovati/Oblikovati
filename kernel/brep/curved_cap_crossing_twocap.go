// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Curved-boolean CAP-CROSSING, two-cap exit (EPIC Oblikovati/Oblikovati#1724, ADR-0046). Slices 1 and 2
// handle an oblique tool that enters the target WALL and exits ONE cap. This file handles the sibling the
// slice-1 recognizer explicitly deferred: a steeper oblique tool that enters one planar cap and exits the
// OTHER, staying INSIDE the wall the whole way (an angled through-hole). The result is the target's wall
// kept WHOLE, each cap holed by its own exit ellipse (an inner-loop hole, as in slice 1), and the tool wall
// between the two cap planes — a two-closed-ellipse-rim band — reversed into the cavity as the tunnel. One
// genus-1 solid. Any tool that also breaches the wall is NOT this clean case and declines to the CSG fallback.

// capEllipse pairs a target cap face with the ellipse the tool exits it through.
type capEllipse struct {
	cap     curvedFace
	ellipse geom.EllipseFull
}

// twoCapPlan is the recognised two-cap exit: both operands as cylinders and the two interior-exit caps with
// their exit ellipses.
type twoCapPlan struct {
	tgt, tool ruledOperand
	capA      capEllipse
	capB      capEllipse
}

// TwoCapCrossingCutGeneral is the exported entry kernel/ops routes a two-cap-exit subtract through. ok=false
// outside the recognised slice so kernel/ops keeps its CSG fallback.
func TwoCapCrossingCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	plan, ok := classifyTwoCapCross(target, tool, rec)
	if !ok {
		return nil, false
	}
	return buildTwoCapCross(plan)
}

// classifyTwoCapCross recognises the two-cap exit, or ok=false for anything outside it. Positive gate: both
// operands full cylinders; the tool exits EXACTLY TWO target caps, each through an ellipse strictly inside
// that cap's rim; and the tool does NOT breach the wall (an empty wall∩wall imprint) — a wall breach is a
// cap-crossing (slice 1/2) or a drill, not this clean cap-to-cap tunnel.
func classifyTwoCapCross(target, tool *topo.Body, rec *diag.Recorder) (twoCapPlan, bool) {
	toolCyl, _, _, okL := cylinderSolidParams(facesOfAny(tool))
	tgtOp, ok1 := cylinderOperand(target)
	toolOp, ok2 := cylinderOperand(tool)
	if !okL || !ok1 || !ok2 {
		return twoCapPlan{}, false
	}
	caps, ok := twoInteriorExitCaps(target, toolCyl)
	if !ok {
		return twoCapPlan{}, false
	}
	if _, breached := crossingCylinderImprint(target, tool, rec); breached {
		return twoCapPlan{}, false // the tool also cuts the wall — not the clean cap-to-cap tunnel
	}
	return twoCapPlan{tgt: tgtOp, tool: toolOp, capA: caps[0], capB: caps[1]}, true
}

// twoInteriorExitCaps returns the target caps the tool exits through an ellipse strictly inside that cap's
// rim; ok=false unless EXACTLY TWO qualify (one → slice 1/2's single-cap exit; zero or ≥3 → not this case).
func twoInteriorExitCaps(target *topo.Body, toolCyl geom.Cylinder) ([]capEllipse, bool) {
	var out []capEllipse
	for _, cap := range planarCapFaces(target) {
		e, ok := capExitEllipseInsideRim(cap, toolCyl)
		if !ok {
			continue
		}
		out = append(out, capEllipse{cap: cap, ellipse: e})
	}
	return out, len(out) == 2
}

// buildTwoCapCross assembles target − tool for the recognised two-cap exit: the whole target wall, each cap
// holed by its exit ellipse, and the tool wall between the two cap planes reversed into the cavity (the
// tunnel). The two exit ellipses are fed to the tool split BY VALUE so curvedStitch re-anchors each stored
// edge to its seam vertex (the slice-1 by-value/T-junction discipline); each cap then reuses the tunnel's
// full-swept ellipse rim so the cap↔tunnel edge welds and orients oppositely once the tunnel is reversed.
func buildTwoCapCross(p twoCapPlan) (*topo.Body, bool) {
	toolImprint := []geom.Curve3{p.capA.ellipse, p.capB.ellipse}
	tunnel, okT := p.tool.split(toolImprint, Difference, true, p.tgt.inside)
	if !okT {
		return nil, false
	}
	holeA, okA := capHoleFromTunnel(tunnel, p.capA.cap)
	holeB, okB := capHoleFromTunnel(tunnel, p.capB.cap)
	if !okA || !okB {
		return nil, false
	}
	faces := make([]curvedFace, 0, len(tunnel)+3)
	faces = append(faces, p.tgt.face) // the wall stays whole — the tool never reaches it
	faces = append(faces, capWithInnerLoop(p.capA.cap, holeA))
	faces = append(faces, capWithInnerLoop(p.capB.cap, holeB))
	faces = append(faces, reverseCurvedFaces(tunnel)...)
	return curvedStitch(faces), true
}

// capHoleFromTunnel finds the tunnel's ellipse rim lying on cap's plane and returns it as a full-swept inner
// loop for that cap, so the cap and the tunnel share the identical oriented ellipse edge. ok=false if cap is
// not planar or the tunnel has no rim on its plane.
func capHoleFromTunnel(tunnel []curvedFace, cap curvedFace) (curvedLoop, bool) {
	pl, ok := cap.surface.(geom.Plane)
	if !ok {
		return curvedLoop{}, false
	}
	return fullSweepCapLoop(tunnel, pl)
}
