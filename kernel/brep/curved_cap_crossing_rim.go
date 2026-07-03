// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-boolean CAP-CROSSING, slice 2 — the RIM-CROSSING CORNER (EPIC Oblikovati/Oblikovati#1724,
// ADR-0046). Slice 1 handled a tool whose exit ellipse lies strictly inside the cap rim. This file handles
// the next sub-family: the exit ellipse CROSSES the rim at exactly two corner points, so the tool exits
// partly through the cap and partly through the wall near the rim. Measured on real traced imprints, that
// case still has a clean closed wall-ENTRY hole lower down (the tool enters the wall once) plus an EXIT
// corner where the wall∩wall SSI is an OPEN chain between the two rim corners, closed on the cap plane by
// an ellipse arc. It decomposes into (a) a wall with the entry hole AND a top-rim notch, (b) the exit cap
// with a mixed rim-arc + ellipse-arc bite, and (c) a tool tunnel whose two loops are the entry rim and the
// [exit-chain ⊕ ellipse-arc] composite, reversed into the cavity. Every configuration outside this exact
// recognizer (tangency, two-lens footprint, two-cap exit, cone tools, no separate entry hole) is DECLINED.

// rimCrossPlan is the recognised slice-2 rim-crossing: both operands as cylinders, the exit cap and its
// exact analytic pieces (the surviving rim arc + the inside ellipse arc, meeting at the two corners), the
// closed wall-entry imprint, and the open wall-exit chain (endpoints snapped to the corners).
type rimCrossPlan struct {
	tgt, tool  ruledOperand
	exitCap    curvedFace
	rimArc     geom.Arc3d
	ellipseArc geom.EllipticalArc
	entry      geom.Polyline // closed wall-entry loop
	exitChain  geom.Polyline // open wall-exit chain p1→p2
}

// RimCrossingCutGeneral is the exported entry kernel/ops routes a rim-crossing cap-crossing subtract
// through. ok=false outside the recognised slice so kernel/ops keeps its CSG fallback.
func RimCrossingCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	plan, ok := classifyRimCross(target, tool, rec)
	if !ok {
		return nil, false
	}
	return buildRimCrossCut(plan)
}

// classifyRimCross recognises the slice-2 rim-crossing, or ok=false for anything outside it. Positive gate:
// both operands full cylinders; the tool exits EXACTLY ONE cap whose exit ellipse crosses that cap's rim at
// EXACTLY TWO corners; and the tool∩wall SSI is EXACTLY ONE closed entry loop plus ONE open exit chain whose
// endpoints are those two corners.
func classifyRimCross(target, tool *topo.Body, rec *diag.Recorder) (rimCrossPlan, bool) {
	toolCyl, _, _, okL := cylinderSolidParams(facesOfAny(tool))
	tgtCyl, _, _, okT := cylinderSolidParams(facesOfAny(target))
	tgtOp, ok1 := cylinderOperand(target)
	toolOp, ok2 := cylinderOperand(tool)
	if !okL || !okT || !ok1 || !ok2 {
		return rimCrossPlan{}, false
	}
	exitCap, ellipse, corners, ok := rimCrossExitCap(target, toolCyl)
	if !ok {
		return rimCrossPlan{}, false
	}
	entry, exitChain, ok := traceEntryAndExit(tgtCyl, toolCyl, target, rec)
	if !ok {
		return rimCrossPlan{}, false
	}
	rimArc, ellArc, ok := cornerArcs(exitCap, ellipse, corners)
	if !ok {
		return rimCrossPlan{}, false
	}
	entryPL, e1 := geom.NewPolyline(entry)
	exitPL, e2 := geom.NewPolyline(snapEndsToCorners(exitChain, corners))
	if e1 != nil || e2 != nil {
		return rimCrossPlan{}, false
	}
	return rimCrossPlan{tgt: tgtOp, tool: toolOp, exitCap: exitCap, rimArc: rimArc,
		ellipseArc: ellArc, entry: entryPL, exitChain: exitPL}, true
}

// rimCrossExitCap finds the single target cap whose exit ellipse crosses that cap's rim at exactly two
// corners; ok=false for zero (interior-exit slice 1 or no contact), a non-2 corner count (tangency or
// two-lens), or more than one such cap (two-cap exit — deferred).
func rimCrossExitCap(target *topo.Body, toolCyl geom.Cylinder) (curvedFace, geom.EllipseFull, []capRimCorner, bool) {
	var exitCap curvedFace
	var ellipse geom.EllipseFull
	var corners []capRimCorner
	found := 0
	for _, cap := range planarCapFaces(target) {
		pl, ok := cap.surface.(geom.Plane)
		if !ok {
			continue
		}
		e, ok := capExitEllipse(toolCyl, pl)
		if !ok {
			continue
		}
		rim, ok := capRimCircle(cap)
		if !ok {
			continue
		}
		cs := capRimCorners(toolCyl, rim)
		if len(cs) != 2 {
			continue
		}
		exitCap, ellipse, corners, found = cap, e, cs, found+1
	}
	return exitCap, ellipse, corners, found == 1
}

// CodeRimCrossTraceTopology marks a rim-crossing wall trace that, AFTER the two-corner cap gate passed, did
// not resolve to the expected one-closed-entry + one-open-exit topology — a genuine near-miss (a tangential
// or grazing wall trace) surfaced so the CSG fallback that follows is observable, not silent (#1724).
const CodeRimCrossTraceTopology diag.Code = "cap-crossing.rim-trace-topology"

// traceEntryAndExit traces the tool∩target-wall SSI over the target band and separates it into exactly one
// CLOSED entry loop and one OPEN exit chain (the rim-crossing signature). ok=false for any other topology
// (no separate entry hole, multiple exit chains) so the recognizer declines; because the two-corner cap gate
// has already passed by the time this runs, an unexpected count is recorded as a tracked defect.
func traceEntryAndExit(tgtCyl, toolCyl geom.Cylinder, target *topo.Body, rec *diag.Recorder) ([]math.Point3, []math.Point3, bool) {
	band := cylinderBand(target)
	tr := geom.TraceSurfaceIntersection(tgtCyl, toolCyl, band)
	var entry, exit [][]math.Point3
	for _, c := range tr.Curves {
		if len(c) < 3 {
			continue
		}
		if c[0].DistanceTo(c[len(c)-1]) < geom.ResolutionForSize(tgtCyl.Radius).Weld() {
			entry = append(entry, c)
		} else {
			exit = append(exit, c)
		}
	}
	if len(entry) != 1 || len(exit) != 1 {
		rec.Recordf(CodeRimCrossTraceTopology, diag.Defect,
			"rim-crossing wall trace gave %d closed + %d open chains, want 1 + 1; declining to CSG", len(entry), len(exit))
		return nil, nil, false
	}
	return entry[0], exit[0], true
}

// cylinderBand returns the target cylinder's axial (u,v) trace window from its own extent.
func cylinderBand(b *topo.Body) geom.SurfaceGrid {
	c, base, h, _ := cylinderSolidParams(facesOfAny(b))
	vLo := float64(c.Origin.VectorTo(base).Dot(c.AxisDir.AsVector()))
	return geom.SurfaceGrid{VMin: vLo, VMax: vLo + h}
}

// snapEndsToCorners returns chain with its two endpoints replaced by the nearest analytic corner point, so
// the wall exit chain, the cap arcs, and the tunnel all share the EXACT corner vertices (the by-value/weld
// discipline of ADR-0046: shared vertices must be bit-identical or the cross-face stitch T-junction-cracks).
func snapEndsToCorners(chain []math.Point3, corners []capRimCorner) []math.Point3 {
	out := append([]math.Point3(nil), chain...)
	out[0] = nearestCorner(out[0], corners)
	out[len(out)-1] = nearestCorner(out[len(out)-1], corners)
	return out
}

func nearestCorner(p math.Point3, corners []capRimCorner) math.Point3 {
	best, bd := p, stdmath.Inf(1)
	for _, c := range corners {
		if d := float64(p.DistanceTo(c.point)); d < bd {
			best, bd = c.point, d
		}
	}
	return best
}

// cornerArcs builds the two exact analytic arcs the cap bite is bounded by: the surviving rim arc (the part
// of the cap rim OUTSIDE the tool) and the inside ellipse arc (the part of the exit ellipse INSIDE the disc),
// both running between the two corners. Each "which arc survives" choice is decided by a midpoint membership
// test, so the construction is orientation-robust. ok=false if the ellipse parameter of a corner is not
// recoverable (a near-tangency the gate should already have excluded).
func cornerArcs(cap curvedFace, ell geom.EllipseFull, corners []capRimCorner) (geom.Arc3d, geom.EllipticalArc, bool) {
	rim, _ := capRimCircle(cap)
	th0, th1 := corners[0].angle, corners[1].angle
	rimArc := rimArcOutsideEllipse(rim, th0, th1, ell)
	f0, n0 := geom.CurveParamAtPoint3(&ell, corners[0].point)
	f1, n1 := geom.CurveParamAtPoint3(&ell, corners[1].point)
	if n0 == geom.NoSolution || n1 == geom.NoSolution {
		return geom.Arc3d{}, geom.EllipticalArc{}, false
	}
	ellArc, ok := ellipseArcInsideDisc(ell, f0, f1, rim)
	if !ok {
		return geom.Arc3d{}, geom.EllipticalArc{}, false
	}
	return rimArc, ellArc, true
}

// rimArcOutsideEllipse returns the rim-circle arc between angles th0 and th1 whose midpoint lies OUTSIDE the
// tool's exit ellipse — the part of the cap rim the cut keeps. It tries the th0→th1 sweep and flips to the
// complementary sweep when that midpoint is inside the ellipse.
func rimArcOutsideEllipse(rim geom.Circle, th0, th1 float64, ell geom.EllipseFull) geom.Arc3d {
	sweep := normalizeSweep(th1 - th0)
	a, _ := geom.NewArc3d(rim.Center, rim.Normal.AsVector(), rim.RefDir.AsVector(), rim.Radius, th0, sweep)
	if insideEllipse(a.PointAt(0.5), ell) {
		a, _ = geom.NewArc3d(rim.Center, rim.Normal.AsVector(), rim.RefDir.AsVector(), rim.Radius, th0, sweep-2*stdmath.Pi)
	}
	return a
}

// ellipseArcInsideDisc returns the ellipse arc between fractions f0 and f1 whose midpoint lies INSIDE the
// cap rim disc — the part of the exit ellipse that actually bounds the cap bite.
func ellipseArcInsideDisc(ell geom.EllipseFull, f0, f1 float64, rim geom.Circle) (geom.EllipticalArc, bool) {
	const twoPi = 2 * stdmath.Pi
	sweep := normalizeSweep(twoPi * (f1 - f0))
	a, err := geom.NewEllipticalArc(ell.Center, ell.Normal.AsVector(), ell.MajorAxis.AsVector(),
		ell.MajorRadius, ell.MinorRadius, twoPi*f0, sweep)
	if err != nil {
		return geom.EllipticalArc{}, false
	}
	if float64(rim.Center.VectorTo(a.PointAt(0.5)).Length()) > rim.Radius {
		a, err = geom.NewEllipticalArc(ell.Center, ell.Normal.AsVector(), ell.MajorAxis.AsVector(),
			ell.MajorRadius, ell.MinorRadius, twoPi*f0, sweep-twoPi)
		if err != nil {
			return geom.EllipticalArc{}, false
		}
	}
	return a, true
}

// normalizeSweep folds a raw angle difference into (0, 2π] so an arc constructor gets a positive sweep whose
// complement is sweep−2π.
func normalizeSweep(d float64) float64 {
	for d <= 0 {
		d += 2 * stdmath.Pi
	}
	for d > 2*stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	return d
}

// insideEllipse reports whether p (on the cap plane) lies inside the tool's exit ellipse — dist to the tool
// axis < r, tested via the ellipse's own quadratic form so it needs no tool handle.
func insideEllipse(p math.Point3, ell geom.EllipseFull) bool {
	w := ell.Center.VectorTo(p)
	u := float64(w.Dot(ell.MajorAxis.AsVector())) / ell.MajorRadius
	minor := ell.Normal.Cross(ell.MajorAxis)
	v := float64(w.Dot(minor)) / ell.MinorRadius
	return u*u+v*v < 1
}

// buildRimCrossCut assembles target − tool for the recognised slice-2 rim-crossing: the target wall kept
// OUTSIDE the tool (a wall with the entry hole AND a top-rim notch), the exit cap with the mixed rim-arc +
// ellipse-arc bite, the target's other caps whole, and the tool wall kept INSIDE the target — a tunnel whose
// two loops are the entry rim and the [exit-chain ⊕ ellipse-arc] composite — reversed into the cavity.
func buildRimCrossCut(p rimCrossPlan) (*topo.Body, bool) {
	wallImprint := []geom.Curve3{&p.entry, &p.exitChain}
	wall, okW := p.tgt.split(wallImprint, Difference, false, p.tool.inside)
	// The tunnel's exit loop closes the open wall chain with the cap-plane ellipse arc; feed the SAME arc the
	// cap uses so the tunnel↔cap edge welds (ADR-0046 shared-edge discipline). A partial EllipticalArc already
	// has PointAt(0) at its start corner, so no re-anchoring is needed (unlike slice 1's full ellipse).
	toolImprint := []geom.Curve3{&p.entry, &p.exitChain, &p.ellipseArc}
	tunnel, okT := p.tool.split(toolImprint, Difference, true, p.tgt.inside)
	if !okW || !okT {
		return nil, false
	}
	faces := make([]curvedFace, 0, len(wall)+len(tunnel)+3)
	faces = append(faces, wall...)
	faces = append(faces, mixedArcCap(p.exitCap, &p.rimArc, &p.ellipseArc))
	faces = append(faces, otherCapsWhole(p.tgt.body, p.exitCap, p.tool.inside)...)
	faces = append(faces, reverseCurvedFaces(tunnel)...)
	return curvedStitch(faces), true
}

// mixedArcCap returns the exit cap trimmed to its rim-crossing bite: a single outer loop made of the
// surviving rim arc and the inside ellipse arc, meeting at the two corners. Both arcs reference the SAME
// curve objects fed to the wall/tunnel splits, so the cap↔wall (rim) and cap↔tunnel (ellipse) edges weld.
func mixedArcCap(cap curvedFace, rimArc geom.Curve3, ellArc geom.Curve3) curvedFace {
	// Both arcs are built running corner[0]→corner[1] (each PointAt(0) is corner[0]). The loop walks the
	// ELLIPSE arc corner[0]→corner[1] then the RIM arc back corner[1]→corner[0]: this winding orients the cap
	// bite consistently with the cap's outward (+cap-normal) face, so the shared rim edge (cap↔wall) and
	// ellipse edge (cap↔tunnel) are each traversed OPPOSITELY by their two faces (the manifold orientation
	// contract). The reverse winding [rim 0→1, ellipse 1→0] chains just as well but leaves both edges
	// co-directional with their neighbours — an inconsistent-orientation crack at the two corners (#1724).
	loop := curvedLoop{edges: []loopEdge{
		{curve: ellArc, t0: 0, t1: 1},
		{curve: rimArc, t0: 1, t1: 0},
	}}
	return curvedFace{surface: cap.surface, reversed: cap.reversed, lineage: cap.lineage, loops: []curvedLoop{loop}}
}
