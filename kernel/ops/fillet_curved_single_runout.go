// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Single-arm curved RUNOUT weld (curved-runout-r1-brief.md, curved-runout-forensic.md). A SINGLE convex
// Plane∧{Cylinder|Cone} pick whose exact analytic arm (an exact torus or cylinder) terminates at TWO
// trihedral vertices, each closed by ONE non-host capping plane, is NOT a trihedral corner — it is the
// curved cousin of the planar V3/V5 runout. The 3-arm trihedral weld (fillet_curved_weld.go) floors it at
// "needs 3 arms"; this file dispatches such a pick to a corner-free BOTH-ENDS assembly instead: the arm
// face plus its two receded hosts plus the two bitten caps, welded watertight, with NO corner sphere and
// NO setback great-arc. Both ends terminate through the EXISTING far-runout engine (armFarRunout), which
// reads only w.center (to pick the far end) and w.radius (r) — so it is called once per end with a minimal
// synthesized cornerWeld whose centre is the OTHER end's vertex. Any decline returns the do-no-harm floor.

// isSingleArmRunout is assembleCurvedArmBody's dispatch classifier: true iff there is exactly ONE pick
// carrying ONE curved cylinder/torus arm whose BOTH end vertices are clean trihedral single-plane caps
// (cappingFaceAtFarVertex admits each). It is FALSE for every multi-pick and every trihedral case (a
// trihedral corner has ≥2 picked edges meeting a vertex, so cappingFaceAtFarVertex's second-filleted-edge
// guard declines), so the 3-arm weld path — where every prior curved green lives — is never reached by
// this branch. Gated to cylinder/torus arms (armBallCenter's domain); a canal BSpline arm stays flooring.
func isSingleArmRunout(fils []edgeFillet) bool {
	if len(fils) != 1 {
		return false // a runout is a single pick; a trihedral corner or any multi-pick is not
	}
	curved := curvedArmsOf(fils)
	if len(curved) != 1 {
		return false
	}
	ef := curved[0]
	if ef.edge == nil || !isRunoutArmKind(ef.armSurface) {
		return false // no wired edge, or a canal/BSpline arm this construction does not model
	}
	if isReflexArcEdge(ef.edge) {
		return false // a >180° curved rim: its receded cone/sphere host is a REFLEX sector whose periodic
		// tessellation (D9-class, subArcMajor territory) is beyond R1's convex-sector tracers — keep flooring
	}
	filletedEdges := map[uint64]bool{ef.edge.ID(): true}
	for _, v := range ef.edge.Vertices() {
		if _, ok, _ := cappingFaceAtFarVertex(v, ef, filletedEdges); !ok {
			return false // an end is not a clean trihedral single-plane cap — keep flooring (do-no-harm)
		}
	}
	return true
}

// isRunoutArmKind reports whether the arm surface is an exact cylinder or torus — the two kinds whose
// rolling-ball spine armBallCenter resolves and whose host contact rail this construction builds. A canal
// (geom.BSplineSurface) arm is excluded, so a single Cone∧Plane RULING pick keeps flooring byte-identically.
func isRunoutArmKind(arm geom.Surface) bool {
	switch arm.(type) {
	case geom.Cylinder, geom.Torus:
		return true
	default:
		return false
	}
}

// isReflexArcEdge reports whether the picked edge is a curved rim sweeping MORE than half a turn (a
// reflex arc, |sweep| > π). Such a rim's receded curved host (cone/sphere) is a reflex angular sector,
// whose periodic-surface tessellation needs the >180° major-span handling (subArcMajor / D9) that R2/R3
// own — beyond R1's convex-sector cylinder-arm / torus-on-cone tracers. A straight (line) edge is never
// reflex, so a cylinder-arm pick (B6) always passes; only a torus-arm arc pick can trip this.
func isReflexArcEdge(e *topo.Edge) bool {
	arc, ok := e.Geometry().(geom.Arc3d)
	return ok && stdmath.Abs(arc.SweepAngle) > stdmath.Pi
}

// armTubeRadius is the rolling-ball radius r of a cylinder or torus arm — the cylinder's radius or the
// torus's minor (tube) radius. ok=false for any other surface (the caller floors).
func armTubeRadius(arm geom.Surface) (float64, bool) {
	switch s := arm.(type) {
	case geom.Cylinder:
		return s.Radius, true
	case geom.Torus:
		return s.MinorRadius, true
	default:
		return 0, false
	}
}

// singleArmRunoutBody welds a single-arm curved runout into a watertight solid: the trimmed arm face, the
// two receded host faces, and the two bitten cap faces, with NO corner sphere/setback. It builds the two
// host contact rails (spanning cap-to-cap), terminates BOTH ends through armFarRunout (which yields the
// far cross-section trim AND the capping-face identity per end), closes the both-ends arm loop, retrims the
// four bitten hosts by splicing each rail/trim into its loop, and assembles + certifies via the caller. An
// EMPTY reason means the returned body is the weld; a non-empty reason names the exact obstruction (with
// offending values) and the body is nil — the caller keeps the clean do-no-harm floor (never a partial body).
func singleArmRunoutBody(body *topo.Body, ef edgeFillet, res Resolution) (*topo.Body, string) {
	arm := ef.armSurface
	r, ok := armTubeRadius(arm)
	if !ok {
		return nil, fmt.Sprintf("single-arm runout: arm surface %T is neither an exact cylinder nor torus (no rolling-ball radius)", arm)
	}
	feet, reason := singleRunoutFeet(ef, arm, r, res)
	if reason != "" {
		return nil, reason
	}
	railA, okA := armRunoutRail(ef.a, arm, feet.a0, feet.a1, res)
	railB, okB := armRunoutRail(ef.b, arm, feet.b0, feet.b1, res)
	if !okA || !okB {
		return nil, fmt.Sprintf("single-arm runout: a host contact rail could not be built (ef.a ok=%v, ef.b ok=%v)", okA, okB)
	}
	run0, run1, reason := singleRunoutTrims(ef, railA, railB, r, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := singleRunoutFaces(body, ef, arm, railA, railB, run0, run1, r, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(faces), ""
}

// runoutFeet holds the arm's four contact feet: on host ef.a at cap0/cap1 (a0/a1) and host ef.b (b0/b1).
// Each foot is the rolling ball's tangent point on that host at that cap — the endpoint shared by the host
// contact rail and the far cross-section trim, so both sides weld byte-identically by construction.
type runoutFeet struct {
	a0, a1, b0, b1 math.Point3
}

// singleRunoutFeet resolves the arm's four contact feet from the spine ball-centres at the two caps (the
// arm spine ∩ each cap = armBallCenter at that far vertex) and the tangent foot of each ball on each host.
// Declines (naming which) when a spine is undefined at a cap or a ball is not internally tangent (≈ r) to a host.
func singleRunoutFeet(ef edgeFillet, arm geom.Surface, r float64, res Resolution) (runoutFeet, string) {
	m0, ok0 := armBallCenter(arm, ef.edge.StartVertex().Point())
	m1, ok1 := armBallCenter(arm, ef.edge.EndVertex().Point())
	if !ok0 || !ok1 {
		return runoutFeet{}, fmt.Sprintf("single-arm runout: arm spine undefined at a far vertex (start ok=%v, end ok=%v)", ok0, ok1)
	}
	tol := res.Weld() * r
	a0, okA0 := armRunoutFoot(ef.a, m0, r, tol)
	a1, okA1 := armRunoutFoot(ef.a, m1, r, tol)
	b0, okB0 := armRunoutFoot(ef.b, m0, r, tol)
	b1, okB1 := armRunoutFoot(ef.b, m1, r, tol)
	if !okA0 || !okA1 || !okB0 || !okB1 {
		return runoutFeet{}, fmt.Sprintf("single-arm runout: an arm ball is not internally tangent (radius %g) to its host "+
			"(a0=%v a1=%v b0=%v b1=%v)", r, okA0, okA1, okB0, okB1)
	}
	return runoutFeet{a0: a0, a1: a1, b0: b0, b1: b1}, ""
}

// armRunoutFoot is the rolling ball's tangent point on host — the foot of the perpendicular from the ball
// centre onto the host surface — accepted only when the ball is internally tangent there (foot distance ≈ r,
// within the model-relative tol). Rejects a host the ball does not actually touch at radius r (do-no-harm).
func armRunoutFoot(host *topo.Face, ballCenter math.Point3, r, tol float64) (math.Point3, bool) {
	_, _, foot := geom.ClosestPointOnSurface(host.Geometry(), ballCenter)
	if stdmath.Abs(float64(foot.DistanceTo(ballCenter))-r) > tol {
		return math.Point3{}, false // ball not tangent to this host at radius r — not a contact foot
	}
	return foot, true
}

// armRunoutRail builds one arm's host contact rail spanning both caps, oriented foot0→foot1: a CYLINDER arm
// (spine a line) contacts a wall/plane along a straight ruling; a TORUS arm (spine a circle) contacts along
// a circular arc of the host contact circle (torusContactCircle — the SAME circle the corner retrim uses),
// re-fit through the two feet and the contact-circle midpoint so it is an exact Arc3d, never a chord.
func armRunoutRail(host *topo.Face, arm geom.Surface, foot0, foot1 math.Point3, res Resolution) (endSeg, bool) {
	switch a := arm.(type) {
	case geom.Cylinder:
		return endSeg{from: foot0, to: foot1}, true // straight ruling on a wall/plane
	case geom.Torus:
		center, radius, ok := torusContactCircle(host.Geometry(), a, res)
		if !ok {
			return endSeg{}, false // this host carries no torus contact circle
		}
		mid := arcMidBetween(center, radius, foot0, foot1)
		arc, err := geom.Arc3dByThreePoints(foot0, mid, foot1)
		if err != nil {
			return endSeg{}, false
		}
		return endSeg{from: foot0, to: foot1, curve: arc, mid: mid, arc: true}, true
	default:
		return endSeg{}, false
	}
}

// singleRunoutTrims terminates BOTH ends through the general far-runout engine (armFarRunout). Each call
// synthesizes a minimal cornerWeld whose centre is the OTHER end's vertex, so farEndVertex returns the
// intended cap: run0 terminates the start end (centre = end vertex), run1 the end end (centre = start
// vertex). The engine reads ONLY w.center and w.radius, so no other cornerWeld field is needed; it yields
// each end's cross-section trim AND its capping-face identity (for the host bite routing) and its admission
// gate. Declines with the engine's exact reason on any far-vertex reject.
func singleRunoutTrims(ef edgeFillet, railA, railB endSeg, r float64, res Resolution) (armRunout, armRunout, string) {
	filletedEdges := map[uint64]bool{ef.edge.ID(): true}
	_, _, run0, ok0, why0 := armFarRunout(ef, cornerWeld{center: ef.edge.EndVertex().Point(), radius: r}, railA, railB, filletedEdges, res)
	if !ok0 {
		return armRunout{}, armRunout{}, why0
	}
	railAr := reverseEndSegs([]endSeg{railA})[0]
	railBr := reverseEndSegs([]endSeg{railB})[0]
	_, _, run1, ok1, why1 := armFarRunout(ef, cornerWeld{center: ef.edge.StartVertex().Point(), radius: r}, railAr, railBr, filletedEdges, res)
	if !ok1 {
		return armRunout{}, armRunout{}, why1
	}
	return run0, run1, ""
}

// singleRunoutFaces builds every result face: the trimmed arm face bounded by the both-ends loop [railA,
// trim@cap1, railB(rev), trim@cap0(rev)], plus the retrimmed hosts. Each of the FOUR bitten faces (the two
// arm hosts ef.a/ef.b, and the two caps run0.capping/run1.capping) has its own rail/trim spliced into its
// loop (farRunoutFace); every other body face passes through verbatim. A retrim decline floors honestly.
func singleRunoutFaces(body *topo.Body, ef edgeFillet, arm geom.Surface, railA, railB endSeg, run0, run1 armRunout, r float64, res Resolution) ([]filletFace, string) {
	loop := append([]endSeg{railA, run1.trim}, reverseEndSegs([]endSeg{railB})...)
	loop = append(loop, reverseEndSegs([]endSeg{run0.trim})...)
	armFace := filletFace{surface: arm, loops: []filletLoop{loopFromSegs(loop)}, parent: filletEdgeProvenance(ef.edge)}
	hosts, reason := singleRunoutHostFaces(body, ef, railA, railB, run0, run1, r, res)
	if reason != "" {
		return nil, reason
	}
	return append(hosts, armFace), ""
}

// singleRunoutHostFaces retrims the four bitten hosts and carries every other face through unchanged. Each
// bitten host keeps the boundary span AWAY from the sharp feature the fillet consumed and splices in the
// bite: an arm host (ef.a/ef.b) recedes along its contact rail, dropping the span carrying the PICKED EDGE
// (identified by either of its end vertices, which lie in that span); a cap drops the small corner carrying
// the FAR VERTEX its far cross-section arc bites. The removed span is chosen by which vertex it contains
// (farPathSegs), NOT by area — a single-arm runout removes ~half of an arm host, so the far-runout "smaller
// corner" heuristic would splice the wrong side. Each bite is the SAME curve object the arm face carries, so
// the two sides weld watertight.
func singleRunoutHostFaces(body *topo.Body, ef edgeFillet, railA, railB endSeg, run0, run1 armRunout, r float64, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * r
	v0, v1 := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	bites := map[*topo.Face]endSeg{ef.a: railA, ef.b: railB, run0.capping: run0.trim, run1.capping: run1.trim}
	// The vertex the removed span carries: the picked edge (either endpoint) for an arm host; the far vertex
	// the arc bites for a cap. run0 terminates the START end (its capping bites v0), run1 the END end (v1).
	avoid := map[*topo.Face]math.Point3{ef.a: v0, ef.b: v0, run0.capping: v0, run1.capping: v1}
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		bite, bitten := bites[f]
		if !bitten {
			out = append(out, passthroughFace(f)) // untouched by the runout — verbatim (coordinate-welded)
			continue
		}
		ff, ok := singleRunoutHostFace(f, bite, avoid[f], tol)
		if !ok {
			return nil, fmt.Sprintf("single-arm runout: host %T retrim declined (bite %v→%v)", f.Geometry(), bite.from, bite.to)
		}
		out = append(out, ff)
	}
	return out, ""
}

// passthroughFace carries a face the runout does not touch through UNCHANGED, but with COORDINATE-welded
// (op-generated, id-0) loop points rather than transformFace's source-id-carrying loops. The retrimmed
// hosts weld by coordinate (loopFromSegs drops source ids), and the point-welder never merges an id-carrying
// point onto an id-0 one — so a source-id pass-through face would NOT weld to its retrimmed neighbour and
// split the shared edge (the B6 other-radial face open-shell). Provenance (the face's own lineage) is kept.
func passthroughFace(f *topo.Face) filletFace {
	loops := make([]filletLoop, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		loops = append(loops, loopFromSegs(segsFromLoop(l)))
	}
	return filletFace{surface: f.Geometry(), loops: loops, parent: f.Lineage()}
}

// singleRunoutHostFace re-clips one bitten host: it keeps the original-loop boundary span from the bite's
// far foot back to its near foot that AVOIDS the removed feature vertex, then closes it with the bite itself
// (the contact rail / far cross-section arc). Any inner (hole) loop is carried through unchanged. Declines
// when the host has too few edges or a bite foot does not lie on its loop (farPathSegs) — do-no-harm.
func singleRunoutHostFace(host *topo.Face, bite endSeg, avoid math.Point3, tol float64) (filletFace, bool) {
	segs := originalHostSegs(host)
	if len(segs) < 3 {
		return filletFace{}, false
	}
	far, ok := farPathSegs(segs, bite.to, bite.from, avoid, tol)
	if !ok {
		return filletFace{}, false // a bite foot is not on this host's loop, or the far path cannot close
	}
	loop := append([]endSeg{bite}, far...)
	loops := append([]filletLoop{loopFromSegs(loop)}, innerHostLoops(host)...)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}
