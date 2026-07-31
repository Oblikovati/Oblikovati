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
	// Cluster-B routing prefix (merge-train seam: lift into assembleCurvedArmBody once the router
	// file unfreezes): an op whose EVERY fillet carries the cyl∧cyl seam payload is claimed here —
	// before the single-pick gate, because a multi-seam op (bfuseblend/B4) is legal — and the full
	// pick group is stashed on fils[0]'s payload for the sequential weld, since the router hands
	// singleArmRunoutBody only fils[0]. Nothing else sets that payload type, so every existing
	// runout keeps its classifier byte-identically.
	if band, ok := fils[0].armSurface.(*cylCylSeamBand); ok {
		band.group = cylCylSeamGroupOf(fils)
		return band.group != nil
	}
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
	// A REFLEX (>180°) rim is admitted too: its receded cone/sphere/cylinder host keeps its true major
	// sector via reflexContactRail (subArcMajor) and, for a sphere host, orientRunoutSphereHost. The
	// blanket reflex reject is gone — the two admission conditions below are the real gate: BOTH ends are
	// clean trihedral single-plane caps (cappingFaceAtFarVertex), and a reflex rim whose major rail cannot
	// close floors honestly inside singleArmRunoutBody (armRunoutRail returns false → curvedArmUnweldedError).
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

// isReflexArcEdge reports whether the picked edge is a curved rim sweeping MORE than half a turn (a reflex
// arc, |sweep| > π). Its receded curved host (cone/sphere/cylinder) is a reflex angular sector kept MAJOR by
// reflexContactRail (subArcMajor). armRunoutRail uses this to FLOOR honestly when a reflex rim's major rail
// cannot close — a straight or convex rim would instead take the byte-identical three-point minor rail. A
// straight (line) edge is never reflex, so a cylinder-arm pick (B6) is never tripped; only a torus-arm arc can.
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
	// Cluster-B routing prefix (pairs with isSingleArmRunout's payload claim): a cyl∧cyl seam
	// payload takes the sequential closed-band weld; every other arm proceeds unchanged.
	if band, ok := ef.armSurface.(*cylCylSeamBand); ok {
		return cylCylSeamGroupBody(body, band, res)
	}
	arm := ef.armSurface
	r, ok := armTubeRadius(arm)
	if !ok {
		return nil, fmt.Sprintf("single-arm runout: arm surface %T is neither an exact cylinder nor torus (no rolling-ball radius)", arm)
	}
	railA, railB, run0, run1, reason := singleRunoutRailsAndTrims(ef, arm, r, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := singleRunoutFaces(body, ef, arm, railA, railB, run0, run1, r, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(orientRunoutSphereHost(body, faces)), ""
}

// singleRunoutRailsAndTrims builds the two host contact rails at the PERPENDICULAR rolling-ball feet,
// terminates BOTH ends through the far-runout engine (yielding each end's cross-section trim + capping
// identity), then re-terminates the rails onto any OBLIQUE end's analytic feet (R3, obliqueRetermRails) so
// an oblique end's rail terminus coincides with its trim foot (D4/E3) while perpendicular ends stay
// byte-identical. Returns rails whose outer ends match the trims' feet — the shared-edge identity the
// host retrim needs to close. Any decline (naming the offending values) floors honestly to the caller.
func singleRunoutRailsAndTrims(ef edgeFillet, arm geom.Surface, r float64, res Resolution) (endSeg, endSeg, armRunout, armRunout, string) {
	feet, reason := singleRunoutFeet(ef, arm, r, res)
	if reason != "" {
		return endSeg{}, endSeg{}, armRunout{}, armRunout{}, reason
	}
	railA, okA := armRunoutRail(ef.a, ef.edge, arm, feet.a0, feet.a1, res)
	railB, okB := armRunoutRail(ef.b, ef.edge, arm, feet.b0, feet.b1, res)
	if !okA || !okB {
		return endSeg{}, endSeg{}, armRunout{}, armRunout{}, fmt.Sprintf("single-arm runout: a host contact rail could not be built (ef.a ok=%v, ef.b ok=%v)", okA, okB)
	}
	run0, run1, reason := singleRunoutTrims(ef, railA, railB, r, res)
	if reason != "" {
		return endSeg{}, endSeg{}, armRunout{}, armRunout{}, reason
	}
	railA, railB, reason = obliqueRetermRails(railA, railB, run0, run1, res.Weld()*r)
	if reason != "" {
		return endSeg{}, endSeg{}, armRunout{}, armRunout{}, reason
	}
	return railA, railB, run0, run1, ""
}

// orientRunoutSphereHost seeds a sphere-HOST single-runout shell (D8) from its host sphere, wound so the
// sphere-patch mesher fills the MATERIAL zone rather than the complement. A reflex runout's host sphere is a
// >hemisphere 270° zone whose retrimmed boundary is a WIDE arc loop — so orientForSphereHost's compact
// byte-identity gate skips it — but the material-zone reseed is winding-robust once the rail/rim arcs are
// sampled (filletLoopWindRing). A cone/cylinder-host runout (C5/B6/C9/C1/M7) carries no sphere face, so this
// is a no-op there (byte-identical). nil body ⇒ no original zone to anchor: do-no-harm pass-through.
func orientRunoutSphereHost(body *topo.Body, faces []filletFace) []filletFace {
	if body == nil {
		return faces
	}
	for i := range faces {
		if _, ok := faces[i].surface.(geom.Sphere); !ok || len(faces[i].loops) == 0 || len(faces[i].loops[0].pts) < 3 {
			continue
		}
		return seedSphereHostSense(body, faces, i, filletLoopWindRing(faces[i].loops[0]))
	}
	return faces
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
// a circular arc of the host contact circle (torusContactCircle — the SAME circle the corner retrim uses).
// A CONVEX (<180°) rim's arc is re-fit through the two feet and the contact-circle midpoint (an exact minor
// Arc3d, never a chord); a REFLEX (>180°) rim (C5/D8) keeps the MAJOR contact band via reflexContactRail so
// its receded cone/sphere host holds the true 270° sector, not the minor complement a 3-point re-fit snaps to.
func armRunoutRail(host *topo.Face, picked *topo.Edge, arm geom.Surface, foot0, foot1 math.Point3, res Resolution) (endSeg, bool) {
	switch a := arm.(type) {
	case geom.Cylinder:
		return endSeg{from: foot0, to: foot1}, true // straight ruling on a wall/plane
	case geom.Torus:
		center, radius, ok := torusContactCircle(host.Geometry(), a, res)
		if !ok {
			center, radius, ok = coneBoreRunoutContactCircle(host.Geometry(), a, res)
		}
		if !ok {
			return endSeg{}, false // this host carries no torus contact circle
		}
		if seg, ok := reflexContactRail(picked, center, radius, foot0, foot1); ok {
			return seg, true // >180° rim: MAJOR contact band (not the minor complement)
		}
		if picked != nil && isReflexArcEdge(picked) {
			return endSeg{}, false // reflex rim whose major rail did not close — floor honestly (never the minor snap)
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

// coneBoreRunoutContactCircle is the single-arm runout's SCOPED fallback for I1's concave-BORE cone arm
// (coneArmFilletConcave, fillet_conearm.go): coneContactCircle (fillet_curved_retrim.go, shared with the
// trihedral corner weld) validates only the CONVEX-external internal tangency (h·sinα−R_s·cosα=+r) and
// must stay that way — TestConvexContactCircleRejectsConcaveTorus pins it as a do-no-harm firewall,
// because the tangency equation alone cannot distinguish I1's edge-CONVEX bore torus from S2's
// genuinely edge-CONCAVE cove torus (concaveConeArmSurface), which needs to keep going through its own
// closed-rim engine. Scoping the external-tangency (h·sinα−R_s·cosα=−r) reading to THIS runout-only
// helper is safe because a genuinely armConcave cone arm never reaches singleArmRunoutBody today (the
// only concave-arm dispatch that can, fillet_arm_concave.go's concaveCurvedArmFillet, handles Cylinder
// hosts only; S2/S5's concave cone/sphere arms are CLOSED rims, routed by isConcaveClosedRimArm before
// isSingleArmRunout is ever tried) — ok=false for any non-Cone host or a genuinely non-coaxial/non-tangent
// torus, so every existing (non-cone, non-bore) runout stays byte-identical.
func coneBoreRunoutContactCircle(host geom.Surface, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	co, isCone := host.(geom.Cone)
	if !isCone || !co.AxisDir.IsParallelTo(tor.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false
	}
	a := co.AxisDir.AsVector()
	h := float64(co.Apex.VectorTo(tor.Center).Dot(a))
	band := res.Weld() * (tor.MajorRadius + tor.MinorRadius)
	if float64(co.Apex.TranslateBy(a.Scale(h)).DistanceTo(tor.Center)) > band {
		return math.Point3{}, 0, false // torus centre off the cone axis — not coaxial
	}
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	if stdmath.Abs(h*sinA-tor.MajorRadius*cosA+tor.MinorRadius) > band {
		return math.Point3{}, 0, false // tube not externally tangent to the cone (h·sinα − R_s·cosα ≠ −r)
	}
	star := h*cosA + tor.MajorRadius*sinA
	return co.Apex.TranslateBy(a.Scale(star * cosA)), star * sinA, true
}

// reflexContactRail builds the torus arm's host contact rail as the MAJOR sub-arc of the contact circle
// (centre/radius from torusContactCircle) when the picked rim is REFLEX (>180°). The rail must span the whole
// picked edge's azimuth (C5/D8's 270°), so a three-point re-fit — ill-conditioned past a semicircle, silently
// snapping to the minor complement (the N7 whole-curve-sub-span lesson) — is wrong there. It reuses subArcMajor:
// it builds a contact-circle PARENT starting at foot0 and sweeping the picked edge's OWN signed sweep (so the
// contact circle winds the rim's rotational sense and its endpoints stay distinct — a full 2π parent would make
// arcFrac alias foot0 to the [0,1] window's far end), then takes that parent's from→to major sub-span. ok=false
// for a straight or convex (<180°) picked edge (the caller keeps the byte-identical three-point rail) and when
// the feet do not subtend a major span on the contact circle.
func reflexContactRail(picked *topo.Edge, center math.Point3, radius float64, foot0, foot1 math.Point3) (endSeg, bool) {
	if picked == nil {
		return endSeg{}, false
	}
	arc, ok := picked.Geometry().(geom.Arc3d)
	if !ok || stdmath.Abs(arc.SweepAngle) <= stdmath.Pi {
		return endSeg{}, false // straight or convex rim: the minor three-point rail is correct (byte-identical)
	}
	parent, err := geom.NewArc3d(center, arc.Normal.AsVector(), center.VectorTo(foot0), radius, 0, arc.SweepAngle)
	if err != nil {
		return endSeg{}, false
	}
	sub, mid, ok := subArcMajor(parent, foot0, foot1)
	if !ok {
		return endSeg{}, false // feet do not subtend a major span on the contact circle
	}
	return endSeg{from: foot0, to: foot1, curve: sub, mid: mid, arc: true}, true
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
		ff, ok := runoutHostRetrim(f, ef, bite, avoid[f], tol)
		if !ok {
			// P4-class capped end: a bite foot is a mid-face fresh cut — bridge it back to the
			// picked edge's end vertex along wall ∩ cap and splice the chain (do-no-harm on decline).
			ff, ok = bridgedRunoutHostFace(f, ef, bite, tol)
		}
		if !ok {
			return nil, fmt.Sprintf("single-arm runout: host %T retrim declined (bite %v→%v)", f.Geometry(), bite.from, bite.to)
		}
		out = append(out, ff)
	}
	return out, ""
}

// runoutHostRetrim dispatches one bitten host's retrim: a CONCAVE arm host (ef.a/ef.b on an armConcave
// fillet — N3/M4/N9) GROWS to the contact rail via concaveArmHostRetrim (feet on the rim-edge
// extensions), while every convex host and every cap keeps the byte-identical recede-and-splice
// singleRunoutHostFace. Gating on ef.armConcave keeps the convex single-arm runout greens bit-identical.
func runoutHostRetrim(f *topo.Face, ef edgeFillet, bite endSeg, avoid math.Point3, tol float64) (filletFace, bool) {
	if !ef.armConcave {
		return singleRunoutHostFace(f, bite, avoid, tol)
	}
	if f == ef.a || f == ef.b {
		return concaveArmHostRetrim(f, bite, ef.edge, tol) // arm host GROWS to the contact rail
	}
	return concaveCapRetrim(f, bite, avoid, tol) // end cap GAINS the fill wedge (variant b)
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

// singleRunoutHostFace re-clips one bitten host: it retrims the loop the bite actually consumes (the loop
// carrying the removed feature vertex — the OUTER boundary for a simple host, but M7's flush-cut cap is bitten
// on its INNER footprint loop), keeping that loop's span from the bite's far foot back to its near foot that
// AVOIDS the removed vertex, then closing with the bite (the contact rail / far cross-section arc). Every OTHER
// loop (e.g. M7 cap's unrelated outer box boundary) is carried through unchanged. Loop ROLES are preserved —
// the OUTER loop stays at index 0 (assembleBody keys outer-ness on that) so the retrimmed inner footprint
// stays a hole, not a phantom outer. Declines when the bitten loop is too small or a bite foot is off it.
func singleRunoutHostFace(host *topo.Face, bite endSeg, avoid math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(host, avoid, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false // malformed host (no loops) — do-no-harm
	}
	retrim, ok := retrimBittenLoop(bitten, bite, avoid, tol)
	if !ok {
		return filletFace{}, false // a bite foot is not on the bitten loop, or the far path cannot close
	}
	loops := hostLoopsWithRetrim(host, bitten, outer, retrim)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// retrimBittenLoop closes the retrimmed bitten loop: the surviving far-path span (bite's far foot back to its
// near foot, avoiding the removed vertex) plus the bite (contact rail / far cross-section arc). Declines when
// the loop is too small or a foot is off it. loopFromSegs drops source ids (coordinate weld) as before.
func retrimBittenLoop(bitten *topo.Loop, bite endSeg, avoid math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	// ≥2, not ≥3: a two-arc "lens" loop (two intersecting-cylinder caps, e.g. blend/simple/O8's
	// top cap) is a legitimate bitten wire — farPathSegs splits it at the bite feet and the far
	// path closes exactly as on a many-seg loop. Only a single-seg (whole-circle) wire, which the
	// split machinery cannot anchor on, stays declined.
	if len(segs) < 2 {
		return filletLoop{}, false
	}
	far, ok := farPathSegs(segs, bite.to, bite.from, avoid, tol)
	if !ok {
		return filletLoop{}, false
	}
	return loopFromSegs(append([]endSeg{bite}, far...)), true
}

// hostLoopsWithRetrim rebuilds the host's loop set with the bitten loop replaced by its retrim and every
// other loop carried through unchanged, EMITTING THE OUTER LOOP FIRST (index 0) because assembleBody marks
// loops[0] as the outer boundary. On a single-loop host (every prior single-arm green) this is just the
// retrimmed outer loop — byte-identical to the previous [retrim]+no-inner emission.
func hostLoopsWithRetrim(host *topo.Face, bitten, outer *topo.Loop, retrim filletLoop) []filletLoop {
	out := []filletLoop{loopForRole(outer, bitten, retrim)}
	for _, l := range host.Loops() {
		if l != outer {
			out = append(out, loopForRole(l, bitten, retrim))
		}
	}
	return out
}

// loopForRole yields the retrim for the bitten loop and a COORDINATE-welded (loopFromSegs, id-0) pass-through
// for any other loop — matching passthroughFace, so M7 cap's surviving outer box boundary welds to the id-0
// pass-through box walls that share it (an id-carrying loop would not merge onto an id-0 neighbour, splitting
// the shared edge).
func loopForRole(l, bitten *topo.Loop, retrim filletLoop) filletLoop {
	if l == bitten {
		return retrim
	}
	return loopFromSegs(segsFromLoop(l))
}

// hostBittenLoop selects the host loop the fillet bite actually consumes — the loop carrying the picked
// feature vertex the bite removes. It PREFERS the outer loop whenever that carries the vertex (every prior
// single-arm green — B6/C9/C1/B7 and M7's three non-cap hosts — so their retrim stays byte-identical); it
// drops to an inner loop only when the vertex is NOT on the outer boundary. That is M7's flush-cut cap (plane
// x=60 through the cylinder axis): the picked-edge vertex lives on the cap's INNER footprint-hole loop, so
// that hole (not the untouched outer box square) is the wire that recedes along the bite. nil for a loopless host.
func hostBittenLoop(host *topo.Face, avoid math.Point3, tol float64) *topo.Loop {
	outer := outerHostLoop(host)
	if outer != nil && loopHasVertex(outer, avoid, tol) {
		return outer
	}
	for _, l := range host.Loops() {
		if loopHasVertex(l, avoid, tol) {
			return l
		}
	}
	return outer
}

// loopHasVertex reports whether p coincides (within the model-relative tol) with one of the loop's vertices.
func loopHasVertex(l *topo.Loop, p math.Point3, tol float64) bool {
	for _, u := range l.EdgeUses() {
		if float64(useFromVertex(u).Point().DistanceTo(p)) <= tol {
			return true
		}
	}
	return false
}
