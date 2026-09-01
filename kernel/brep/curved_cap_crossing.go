// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-boolean CAP-CROSSING, slice 1 (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The general ruled
// cut/join (curved_general_ruled_cutjoin.go) DECLINES whenever the tool pokes THROUGH a planar cap of the
// target (loopsClearOfCaps) — a partial curved-on-planar contact ADR-0045 left to a follow-up. This file
// builds the narrowest, fully-certifiable sub-family of that: an OBLIQUE CYLINDER tool that enters the
// target's curved wall once (a single in-band imprint hole) and exits exactly ONE planar cap through an
// ELLIPSE section lying STRICTLY INSIDE that cap's rim (the interior-exit case). Measured on real traced
// imprints, that case decomposes cleanly into (a) a wall with a hole, (b) a cap with an elliptical inner-
// loop hole — the oblique generalization of the drill through-hole [P] pattern — and (c) a tool tunnel
// whose two boundary loops are the wall-entry hole and the cap-exit ellipse, built by the SAME two-closed-
// loop band trim a normal crossing uses. Every other cap-crossing configuration (rim-crossing corner,
// line-pair strip, two-cap exit, cone tools, tangency) is DECLINED so the boolean keeps its recorded CSG
// fallback — a partial implementation ships manifold-but-geometrically-wrong solids, so the gate is a
// tight POSITIVE recognizer, never a widened decline.

const (
	// capEllipseCosMin/Max bound |n·d| (n cap-plane normal, d tool axis) to a GENUINE ellipse section: below
	// Min the plane is ~parallel to the axis (a line-pair strip, the deferred rim-crossing family); above Max
	// the tool is ~coaxial with the cap normal (no transversal wall crossing at all).
	capEllipseCosMin = 0.05
	capEllipseCosMax = 0.999
)

// capExitEllipse returns the ellipse tool ∩ capPlane (an oblique cut of a right circular cylinder): centre
// at the tool axis ∩ plane, semi-minor = r along n×d (unforeshortened), semi-major = r/|n·d| along the
// in-plane axis projection. ok=false when |n·d| is out of [capEllipseCosMin, capEllipseCosMax] — a line-pair
// or a near-circular/degenerate section slice 1 does not build.
func capExitEllipse(tool geom.Cylinder, capPlane geom.Plane) (geom.EllipseFull, bool) {
	n, err := math.UnitVector3FromVector(capPlane.Normal())
	if err != nil {
		return geom.EllipseFull{}, false
	}
	nv := n.AsVector()
	d := tool.AxisDir.AsVector()
	nd := float64(nv.Dot(d))
	absnd := stdmath.Abs(nd)
	if absnd < capEllipseCosMin || absnd > capEllipseCosMax {
		return geom.EllipseFull{}, false
	}
	// centre = tool axis ∩ cap plane: A + t·d with t = n·(Q0 − A)/(n·d), A = tool.Origin, Q0 = capPlane.Origin.
	a := tool.Origin
	t := float64(a.VectorTo(capPlane.Origin).Dot(nv)) / nd
	centre := a.TranslateBy(d.Scale(math.Scalar(t)))
	major := d.Sub(nv.Scale(math.Scalar(nd))) // projection of the axis onto the plane
	e, err := geom.NewEllipseFull(centre, nv, major, tool.Radius/absnd, tool.Radius)
	if err != nil {
		return geom.EllipseFull{}, false
	}
	return e, true
}

// ellipseInsideRim reports whether every point of e sits strictly inside a rim circle of radius R centred at
// rimCentre (both coplanar on the cap) by at least margin — the slice-1 interior-exit gate. Any point on or
// outside the rim means the ellipse crosses it (the deferred rim-crossing corner), so slice 1 declines.
func ellipseInsideRim(e geom.EllipseFull, rimCentre math.Point3, r, margin float64) bool {
	const n = 128 // dense enough to catch a shallow rim graze the coarse arrangement would miss
	for i := range n {
		p := e.PointAt(float64(i) / float64(n))
		if float64(rimCentre.VectorTo(p).Length()) > r-margin {
			return false
		}
	}
	return true
}

// capRimCircle recovers a planar cap's boundary rim circle (centre + radius) from its outer loop.
func capRimCircle(cap curvedFace) (geom.Circle, bool) {
	for _, lp := range cap.loops {
		for _, e := range lp.edges {
			if c, ok := e.curve.(geom.Circle); ok {
				return c, true
			}
		}
	}
	return geom.Circle{}, false
}

// capCrossPlan is the recognised slice-1 cap-crossing: both operands as ruled cylinders, the single target
// cap the tool exits (with its exact exit ellipse), and the lone in-band wall-entry imprint loop.
type capCrossPlan struct {
	tgt, tool ruledOperand
	exitCap   curvedFace
	ellipse   geom.EllipseFull
	entry     []geom.Curve3
}

// classifyCapCross recognises the slice-1 interior-exit cap-crossing, or ok=false for anything outside it
// (so the caller keeps the recorded CSG fallback). Positive gate: both operands full cylinders; the tool
// exits EXACTLY ONE target cap through an ellipse strictly inside that cap's rim; and the wall∩wall imprint
// is EXACTLY ONE closed in-band loop (the single wall-entry hole).
func classifyCapCross(target, tool *topo.Body, rec *diag.Recorder) (capCrossPlan, bool) {
	toolCyl, _, _, okL := cylinderSolidParams(facesOfAny(tool))
	tgtOp, ok1 := cylinderOperand(target)
	toolOp, ok2 := cylinderOperand(tool)
	if !okL || !ok1 || !ok2 {
		return capCrossPlan{}, false
	}
	exitCap, ellipse, ok := findInteriorExitCap(target, toolCyl)
	if !ok {
		return capCrossPlan{}, false
	}
	entry, ok := crossingCylinderImprint(target, tool, rec)
	if !ok || len(entry) != 1 {
		return capCrossPlan{}, false
	}
	// The single imprint loop must be a genuine WALL hole strictly between the target's caps; if it itself
	// reached a cap the crossing is not the clean interior-exit slice 1 handles.
	if !loopsClearOfCaps(tgtOp.newUV(Difference, false, toolOp.inside), entry) {
		return capCrossPlan{}, false
	}
	return capCrossPlan{tgt: tgtOp, tool: toolOp, exitCap: exitCap, ellipse: ellipse, entry: entry}, true
}

// findInteriorExitCap returns the single target cap the tool exits through an ellipse strictly inside that
// cap's rim (the slice-1 interior-exit gate), with that exit ellipse. ok=false unless EXACTLY ONE cap
// qualifies — zero (no cap exit) or two (a two-cap exit, the deferred family) both decline.
func findInteriorExitCap(target *topo.Body, toolCyl geom.Cylinder) (curvedFace, geom.EllipseFull, bool) {
	var exitCap curvedFace
	var ellipse geom.EllipseFull
	found := 0
	for _, cap := range planarCapFaces(target) {
		e, ok := capExitEllipseInsideRim(cap, toolCyl)
		if !ok {
			continue
		}
		exitCap, ellipse, found = cap, e, found+1
	}
	return exitCap, ellipse, found == 1
}

// capExitEllipseInsideRim returns the tool∩cap ellipse when cap is planar AND that ellipse lies strictly
// inside cap's rim circle; ok=false otherwise (not the interior-exit case for this cap).
func capExitEllipseInsideRim(cap curvedFace, toolCyl geom.Cylinder) (geom.EllipseFull, bool) {
	pl, ok := cap.surface.(geom.Plane)
	if !ok {
		return geom.EllipseFull{}, false
	}
	e, ok := capExitEllipse(toolCyl, pl)
	if !ok {
		return geom.EllipseFull{}, false
	}
	rim, ok := capRimCircle(cap)
	if !ok {
		return geom.EllipseFull{}, false
	}
	if !ellipseInsideRim(e, rim.Center, rim.Radius, geom.ResolutionForSize(rim.Radius).Plane()) {
		return geom.EllipseFull{}, false
	}
	return e, true
}

// CapCrossingCutGeneral is the exported entry kernel/ops routes a cap-crossing subtract through: the
// interior-exit slice of the general cap-crossing pipeline (#1724). ok=false outside the recognised slice
// so kernel/ops keeps its CSG fallback.
func CapCrossingCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	plan, ok := classifyCapCross(target, tool, rec)
	if !ok {
		return nil, false
	}
	return buildCapCrossCut(plan)
}

// buildCapCrossCut assembles target − tool for the recognised slice-1 cap-crossing (#1724): the target
// wall kept OUTSIDE the tool (a wall with the entry hole), the exit cap holed by the exit ellipse (an
// inner-loop hole), the target's other caps whole, and the tool wall kept INSIDE the target — a tube
// between the entry hole and the exit ellipse — reversed into the cavity. One genus-1 tunnel solid (χ=0).
func buildCapCrossCut(p capCrossPlan) (*topo.Body, bool) {
	wallImprint := p.entry
	wall, okW := p.tgt.split(wallImprint, Difference, false, p.tool.inside)
	// The tool tunnel's two boundary loops are the wall-entry hole and the cap-exit ellipse. Feed the ellipse
	// by VALUE, not &p.ellipse: curvedStitch's edgeCurveFor switches on the CONCRETE curve type (case
	// geom.EllipseFull) to re-anchor the stored edge to an EllipticalArc whose PointAt(0) is the seam vertex.
	// A *pointer* misses that switch, so the edge keeps the raw full ellipse (PointAt(0)=major-axis vertex)
	// while its seam vertex sits at PointAt(t0) — a mismatch that makes discretizeEdge (snaps to the vertex)
	// and TessellateEdge (walks the raw domain) sample the ellipse differently, T-junction-cracking the cap
	// against the tunnel rim (#1724).
	toolImprint := make([]geom.Curve3, 0, len(wallImprint)+1)
	toolImprint = append(toolImprint, wallImprint...)
	toolImprint = append(toolImprint, p.ellipse)
	tunnel, okT := p.tool.split(toolImprint, Difference, true, p.tgt.inside)
	if !okW || !okT {
		return nil, false
	}
	// Normalize the tunnel's cap-plane ellipse edge from the arrangement's degenerate t0==t1 (zero-sweep,
	// whose direction is ambiguous and whose midpoint collapses onto its vertex) to a FULL forward sweep
	// (t0, t0+1). That makes it (a) a genuine CLOSED rim the two-rim band mesher accepts and (b) an edge with
	// a real midpoint, so the cap hole — reusing the identical loopEdge — welds with it and, after the tunnel
	// is reversed into the cavity, curvedStitch orients the two incident faces oppositely (#1724).
	capPlane, _ := p.exitCap.surface.(geom.Plane)
	holeLoop, ok := fullSweepCapLoop(tunnel, capPlane)
	if !ok {
		return nil, false
	}
	faces := make([]curvedFace, 0, len(wall)+len(tunnel)+3)
	faces = append(faces, wall...)
	faces = append(faces, capWithInnerLoop(p.exitCap, holeLoop))
	faces = append(faces, otherCapsWhole(p.tgt.body, p.exitCap, p.tool.inside)...)
	faces = append(faces, reverseCurvedFaces(tunnel)...)
	return curvedStitch(faces), true
}

// fullSweepCapLoop finds the tunnel's single-edge loop on the exit cap plane (its ellipse rim), rewrites
// that edge IN PLACE to a full forward sweep (t1 = t0+1) so it is a closed rim with a real midpoint, and
// returns a copy of the loop for the cap hole. Both the tunnel and the cap then reference the identical
// oriented ellipse edge, so it welds and orients consistently. ok=false if no cap-plane loop is found.
func fullSweepCapLoop(tunnel []curvedFace, capPlane geom.Plane) (curvedLoop, bool) {
	nrm := capPlane.Normal()
	tol := geom.ResolutionForSize(1).Plane()
	for fi := range tunnel {
		for li := range tunnel[fi].loops {
			edges := tunnel[fi].loops[li].edges
			if len(edges) == 1 && onPlane(edges[0].start(), capPlane.Origin, nrm, tol) {
				edges[0].t1 = edges[0].t0 + 1.0 // full forward sweep: a closed rim with a real midpoint
				return curvedLoop{edges: []loopEdge{edges[0]}}, true
			}
		}
	}
	return curvedLoop{}, false
}

// onPlane reports whether p lies on the plane through origin with the given normal, within tol.
func onPlane(p, origin math.Point3, normal math.Vector3, tol float64) bool {
	return stdmath.Abs(float64(origin.VectorTo(p).Dot(normal))) < tol
}

// capWithHole returns the exit cap with holeLoop added as an inner-loop hole (the disc minus the tool
// footprint — the interior-exit case, so the ellipse is strictly inside the rim). holeLoop is the tunnel's
// own cap-plane loop, so the shared ellipse edge welds; curvedStitch orients the two incident faces.
func capWithInnerLoop(cap curvedFace, holeLoop curvedLoop) curvedFace {
	loops := make([]curvedLoop, 0, len(cap.loops)+1)
	loops = append(loops, cap.loops...)
	loops = append(loops, holeLoop)
	return curvedFace{surface: cap.surface, reversed: cap.reversed, lineage: cap.lineage, loops: loops}
}

// otherCapsWhole returns the target's caps OTHER than the exit cap that lie outside the tool — kept whole
// (the entry-side cap the tool never reaches). The exit cap is emitted separately by capWithEllipseHole.
func otherCapsWhole(b *topo.Body, exit curvedFace, toolInside func(math.Point3) bool) []curvedFace {
	exitPl, _ := exit.surface.(geom.Plane)
	var out []curvedFace
	for _, cap := range planarCapFaces(b) {
		pl, ok := cap.surface.(geom.Plane)
		if !ok || samePoint(pl.Origin, exitPl.Origin, geom.ResolutionForSize(1)) {
			continue
		}
		if !toolInside(pl.Origin) {
			out = append(out, cap)
		}
	}
	return out
}
