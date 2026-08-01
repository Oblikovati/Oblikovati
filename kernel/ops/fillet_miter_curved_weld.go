// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-face miter body weld (families B/C). Two equal-radius rolling-ball arms — one exact torus,
// one exact cylinder — sharing a face MUTUALLY TRIM along the torus∩cylinder seam (fillet_miter_curved.go).
// This assembles the watertight solid: the two trimmed arm faces (each bounded by its two host contact
// rails, the shared seam, and its far cross-section trim), the receded shared face (bitten by BOTH arms'
// shared rails meeting at the seam top), the receded outer faces and far caps, and every untouched face
// carried through. Any decline returns a do-no-harm reason (the caller keeps the clean floor, never a
// partial body) — so a far end that caps against a curved face (P5's cylinder caps, an oblique/canal
// runout out of this analytic scope) floors honestly here.

// curvedMiterOf returns the curved-contact miter corner (families B/C) whose two arms are exactly the
// filleted edges, or nil when this corner set is not a 2-arm curved miter (the caller keeps the
// trihedral / single-arm paths). It fires only when there are exactly two picked edges and a solved
// curved miter joins them at their shared vertex.
func curvedMiterOf(fils []edgeFillet, miters map[uint64]*cornerMiter) *cornerMiter {
	if len(fils) != 2 {
		return nil
	}
	for _, m := range miters {
		if m == nil || m.curved == nil {
			continue
		}
		if miterJoinsEdges(m, fils[0].edge, fils[1].edge) {
			return m
		}
	}
	return nil
}

// miterJoinsEdges reports whether the curved miter's two arm edges are exactly e0 and e1.
func miterJoinsEdges(m *cornerMiter, e0, e1 *topo.Edge) bool {
	a, b := m.curved.torEdge, m.curved.cylEdge
	return (a == e0 && b == e1) || (a == e1 && b == e0)
}

// miterArmSide is one curved-miter arm's assembled boundary data: its exact surface, the outer host,
// the two host contact rails (shared-face rail farShared→sTop, outer-face rail farOuter→sBot), and the
// far cross-section runout (trim + capping face) terminating the non-miter end.
type miterArmSide struct {
	surface geom.Surface
	edge    *topo.Edge
	outer   *topo.Face
	railSh  endSeg // shared-face contact rail: farShared → sTop
	railOut endSeg // outer-face contact rail: farOuter → sBot (or → the cap junction, see transit)
	run     armRunout
	// transit is non-nil when sBot lies on the OTHER arm's outer host rather than on this arm's
	// own outer face (P5's cylinder arm, sBot on the top cap): the rail then ends at the arm
	// tube's junction with that host's plane, and transit — the tube∩plane cross-section arc —
	// carries the arm boundary from the junction to sBot. nil on every pure two-host miter.
	transit *endSeg
}

// curvedMiterBody welds the 2-arm curved miter into a watertight solid, or returns a do-no-harm reason.
// It builds each arm's rails + far runout, then the arm faces, the twice-bitten shared face, and the
// receded outer/cap hosts, and assembles + certifies via the caller — TWO do-no-harm floors beyond
// assembleBody's own edge-use/Euler checks: miterWeldMeshDefect (a bridged loop self-crosses/retraces)
// and curvedMiterSeamOffSurface (a boundary edge strays off the face it bounds — fillet_miter_seam_offsurface.go).
func curvedMiterBody(body *topo.Body, m *cornerMiter, res Resolution) (*topo.Body, string) {
	c := m.curved
	r := c.radius()
	torSide, reason := buildMiterArmSide(m, c.armA(), c.torEdge, r, res)
	if reason != "" {
		return nil, reason
	}
	cylSide, reason := buildMiterArmSide(m, c.armB(), c.cylEdge, r, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := curvedMiterFaces(body, m, torSide, cylSide, r, res)
	if reason != "" {
		return nil, reason
	}
	built := assembleBody(faces)
	if reason := miterWeldMeshDefect(built); reason != "" {
		return nil, reason
	}
	if reason := curvedMiterSeamOffSurface(built, r); reason != "" {
		return nil, reason
	}
	return built, ""
}

// miterWeldMeshDefect is the miter weld's own do-no-harm floor against a defect the B-rep-level checks
// (edge-use counts, Euler characteristic — weldCurvedArmOrFloor's Validate call) cannot see. It ROOT-
// CAUSED to a shared-boundary identity bug (A1 investigation, simple/W3, W4): an arm's far-runout foot
// (armRunoutFoot, a closed-form ball-tangent projection) and the SAME physical corner's pre-existing
// topology (the picked edge's own far vertex / the host's own boundary loop) are two INDEPENDENTLY
// computed representations of the same point — exact GIVEN their own inputs, but the inputs themselves
// (a face's stored axis parameters vs. a vertex's stored coordinates) come from the SAME STEP file yet
// need only agree to the file's own construction precision, not to float noise (measured on
// simple/W4.step: the boss cylinder's axis sits at z=0.9999, CARTESIAN_POINT #133/#249, while the
// corner vertex it must weld against sits at z=1 exactly, #86 — a genuine ~1e-4 gap baked into the
// SOURCE geometry). Every downstream construction step (armBallCenter, armRunoutFoot, the torus contact
// arc) is closed-form and exact GIVEN that gap, so it projects through to a foot ~2e-5 off the original
// loop it is genuinely incident to — just past the tight weld-scale tol (res.Weld()·r, calibrated for
// float noise) but comfortably inside snapTol (res.Sew(), this project's OWN "close a gap between
// independently-derived geometry" tolerance). sharedRailAnchor / miterTrimChain / chainedHostRetrim now
// reconcile the foot against the original loop at snapTol — the SAME "shared boundary, one canonical
// value" identity this project's discretizeEdge invariant already enforces elsewhere — so W3/W4 never
// build a spurious bridge/chain past the near-coincident point in the first place. This function remains
// as the do-no-harm floor for any FUTURE case whose foot is genuinely off-loop (a real defect, not a
// Sew-scale artifact of independently-stored geometry): the SAME two detectors the corpus's own
// watertightness gate uses (SelfCrossingFaceLoops / RetracingFaceLoops) at PropertyQuality.
func miterWeldMeshDefect(b *topo.Body) string {
	if len(SelfCrossingFaceLoops(b, PropertyQuality())) > 0 {
		return "curved miter: assembled weld self-crosses at a bridged shared/capping loop (a pre-existing near-boundary residual, not a fresh capability gap)"
	}
	if len(RetracingFaceLoops(b, PropertyQuality())) > 0 {
		return "curved miter: assembled weld retraces at a bridged shared/capping loop (a pre-existing near-boundary residual, not a fresh capability gap)"
	}
	return ""
}

// buildMiterArmSide solves one arm's two host contact rails and its far cross-section runout: the far
// feet from the arm ball centre at the non-miter vertex, the shared/outer rails from those feet to the
// seam endpoints (sTop/sBot), and the far termination through the general far-runout engine (which
// floors honestly on an oblique/curved cap — P5's cylinder-capped far ends). reason names any decline.
func buildMiterArmSide(m *cornerMiter, arm geom.Surface, edge *topo.Edge, r float64, res Resolution) (miterArmSide, string) {
	outer := otherFace(edge, m.shared)
	if outer == nil {
		return miterArmSide{}, fmt.Sprintf("curved miter: edge %d has no outer face opposite the shared face", edge.ID())
	}
	tol := res.Weld() * r
	far := farVertexNotVid(edge, m.vertex.ID())
	ball, ok := armBallCenter(arm, far)
	if !ok {
		return miterArmSide{}, fmt.Sprintf("curved miter: arm spine undefined at far vertex of edge %d", edge.ID())
	}
	farShared, okS := armRunoutFoot(m.shared, ball, r, tol)
	farOuter, okO := armRunoutFoot(outer, ball, r, tol)
	if !okS || !okO {
		return miterArmSide{}, fmt.Sprintf("curved miter: far foot not tangent on a host of edge %d (shared=%v outer=%v)", edge.ID(), okS, okO)
	}
	return assembleMiterArmSide(m, arm, edge, outer, farShared, farOuter, r, res)
}

// assembleMiterArmSide builds the two contact rails (far foot → seam endpoint) and the far runout for
// one arm, given its far feet. Split from buildMiterArmSide to stay within funlen.
func assembleMiterArmSide(m *cornerMiter, arm geom.Surface, edge *topo.Edge, outer *topo.Face, farShared, farOuter math.Point3, r float64, res Resolution) (miterArmSide, string) {
	sTop, sBot := m.seam[0], m.seam[len(m.seam)-1]
	tol := res.Weld() * r
	railSh, okA := armRunoutRail(m.shared, edge, arm, farShared, sTop, res)
	railOut, transit, okB := miterOuterRail(m, arm, edge, outer, farOuter, sBot, r, res)
	if !okA || !okB {
		return miterArmSide{}, fmt.Sprintf("curved miter: a host contact rail could not be built on edge %d (shared=%v outer=%v)", edge.ID(), okA, okB)
	}
	ef := edgeFillet{a: m.shared, b: outer, edge: edge, armSurface: arm}
	filed := map[uint64]bool{m.curved.torEdge.ID(): true, m.curved.cylEdge.ID(): true}
	_, _, run, ok, reason := armFarRunout(ef, cornerWeld{center: m.vertex.Point(), radius: r}, railSh, railOut, filed, res)
	if !ok {
		return miterArmSide{}, fmt.Sprintf("curved miter: far runout declined on edge %d: %s", edge.ID(), reason)
	}
	railOut, ok = reconcileOuterRailWithTrim(railOut, run, edge, arm, outer, tol, res)
	if !ok {
		return miterArmSide{}, fmt.Sprintf("curved miter: outer rail does not meet the far trim on edge %d", edge.ID())
	}
	return miterArmSide{surface: arm, edge: edge, outer: outer, railSh: railSh, railOut: railOut, run: run, transit: transit}, ""
}

// miterOuterRail builds one arm's outer-host contact rail. When the seam endpoint sBot lies ON the
// arm's own outer host (every pure two-host miter — byte-identical legacy path) the rail runs
// farOuter→sBot exactly as before. When sBot lies on the OTHER arm's outer plane instead (P5's
// cylinder arm: sBot on the top cap), the rail can only reach the arm tube's junction with that
// plane; the returned transit arc (tube ∩ plane) carries the boundary junction→sBot from there.
func miterOuterRail(m *cornerMiter, arm geom.Surface, edge *topo.Edge, outer *topo.Face, farOuter, sBot math.Point3, r float64, res Resolution) (endSeg, *endSeg, bool) {
	tol := res.Weld() * r
	if distanceToSurface(outer, sBot) <= tol {
		rail, ok := armRunoutRail(outer, edge, arm, farOuter, sBot, res)
		return rail, nil, ok // legacy: sBot is on this arm's own outer host
	}
	cylArm, isCyl := arm.(geom.Cylinder)
	capFace := otherFace(otherMiterEdge(m, edge), m.shared)
	if !isCyl || capFace == nil {
		return endSeg{}, nil, false
	}
	capPl, isPl := capFace.Geometry().(geom.Plane)
	if !isPl {
		return endSeg{}, nil, false // sBot off both outer hosts — outside the analytic transit scope
	}
	spineAtCap, junction, ok := cylArmCapJunction(outer, cylArm, capPl, r, tol)
	if !ok {
		return endSeg{}, nil, false
	}
	transit, ok := tubeCapTransitArc(spineAtCap, r, junction, sBot, tol)
	if !ok {
		return endSeg{}, nil, false
	}
	rail, ok := armRunoutRail(outer, edge, arm, farOuter, junction, res)
	return rail, &transit, ok
}

// otherMiterEdge is the miter's OTHER picked edge (the sibling arm's edge).
func otherMiterEdge(m *cornerMiter, edge *topo.Edge) *topo.Edge {
	if m.curved.torEdge == edge {
		return m.curved.cylEdge
	}
	return m.curved.torEdge
}

// reconcileOuterRailWithTrim rebuilds the outer rail to end at the far trim's actual foot on the
// outer host when the two disagree — the curved-capping case (P5's torus arm, capped by the pocket
// wall): the trim = arm ∩ capping-cylinder ends at the true triple point on the outer host, while
// the rail was seeded from the far BALL's tangency foot, a point r-ish away that is no vertex of
// the final solid. A plane-capped (perpendicular) runout has the two coincide, so this is a no-op
// there and every prior green is byte-identical.
func reconcileOuterRailWithTrim(railOut endSeg, run armRunout, edge *topo.Edge, arm geom.Surface, outer *topo.Face, tol float64, res Resolution) (endSeg, bool) {
	trimFoot := trimEndNearerSurface(run.trim, outer)
	if float64(trimFoot.DistanceTo(railOut.from)) <= tol || distanceToSurface(outer, trimFoot) > tol {
		return railOut, true // coincident (legacy) or the trim never lands on the outer host
	}
	return armRunoutRail(outer, edge, arm, trimFoot, railOut.to, res)
}

// curvedMiterFaces (result-face assembly), curvedMiterHostFaces, the passthrough/shared soundness
// guards, miterHostRetrim/miterHostBiteChains/miterTrimChain, and miterArmFace all live in
// fillet_miter_curved_hostfaces.go — split out to keep this file under the 500-line/one-responsibility
// rule (this file owns the arm-side solve + the shared-face two-rail retrim).

// seamEndSegs turns the seam polyline into straight chord segments sTop→sBot.
func seamEndSegs(seam []math.Point3) []endSeg {
	segs := make([]endSeg, 0, len(seam)-1)
	for i := 0; i+1 < len(seam); i++ {
		segs = append(segs, endSeg{from: seam[i], to: seam[i+1]})
	}
	return segs
}

// sharedMiterRetrim recedes the shared face: it removes the corner span carrying the miter vertex and
// splices in BOTH arms' shared-face contact rails, which meet at the seam top sTop — so the shared face
// recedes to the two rails joined at their common tangent point. On a CURVED shared face (P5's cylinder)
// an arm's far shared foot can be an INTERIOR fresh cut the fillet itself makes (mid-wall, not on any
// original edge); sharedMiterLoop then bridges it to the arm's far vertex with one cap-bridge arc so the
// far path can anchor on the original loop. The all-on-loop case stays byte-identical (no bridge). snapTol
// is the SEPARATE, coarser reconciliation tolerance (see sharedRailAnchor) — never tol, which stays the
// tight per-corner weld scale for the loop-selection test (hostBittenLoop) this function also runs.
func sharedMiterRetrim(m *cornerMiter, tor, cyl miterArmSide, tol, snapTol float64) (filletFace, bool) {
	shared, v := m.shared, m.vertex.Point()
	bitten := hostBittenLoop(shared, v, tol)
	outer := outerHostLoop(shared)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	retrim, ok := sharedMiterLoop(m, tor, cyl, segsFromLoop(bitten), tol, snapTol)
	if !ok {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(shared, bitten, outer, retrim)
	return filletFace{surface: shared.Geometry(), loops: loops, parent: shared.Lineage()}, true
}

// sharedMiterLoop builds the receded shared face's boundary loop: both arms' shared rails (farShared_tor
// →sTop→farShared_cyl), each arm's cap-bridge to its far vertex when its far foot is an off-loop fresh cut
// (P5's cyl side; empty on the on-loop torus side → byte-identical), and the surviving original-loop far
// path between the two anchors that avoids the miter vertex. false on any anchor / far-path decline.
// snapTol is the coarser reconciliation tolerance (see sharedRailAnchor) for the on-loop TEST and the
// far-path lookup of whichever anchor that test accepted — both must agree on the same tolerance for the
// same points, or a foot sharedRailAnchor accepts as on-loop becomes unlocatable here and the corner
// spuriously declines (simple/W3, W4: torAnchor sits 2e-5 off the segment it is genuinely incident to).
// tol stays the tight per-corner weld scale for the bridge's own co-latitude/exactness certificate.
func sharedMiterLoop(m *cornerMiter, tor, cyl miterArmSide, segs []endSeg, tol, snapTol float64) (filletLoop, bool) {
	vid, v := m.vertex.ID(), m.vertex.Point()
	torAnchor, torBridge, okT := sharedRailAnchor(m.shared, tor, vid, segs, tol, snapTol)
	cylAnchor, cylBridge, okC := sharedRailAnchor(m.shared, cyl, vid, segs, tol, snapTol)
	if !okT || !okC {
		return filletLoop{}, false
	}
	far, ok := farPathSegs(segs, cylAnchor, torAnchor, v, snapTol)
	if !ok {
		return filletLoop{}, false
	}
	return loopFromSegs(sharedRetrimSegs(tor, cyl, cylBridge, far, torBridge)), true
}

// sharedRetrimSegs concatenates the receded shared loop in traversal order: railTor (farShared_tor→sTop),
// railCyl reversed (sTop→farShared_cyl), the cyl arm's cap-bridge (→its far vertex, empty when on-loop),
// the surviving far path (→the tor anchor), and the tor arm's cap-bridge reversed (→farShared_tor, empty
// when on-loop). With both bridges empty this is byte-identical to the prior [railTor, railCyl⁻¹, far] splice.
func sharedRetrimSegs(tor, cyl miterArmSide, cylBridge, far, torBridge []endSeg) []endSeg {
	segs := append([]endSeg{tor.railSh}, reverseEndSegs([]endSeg{cyl.railSh})...)
	segs = append(segs, cylBridge...)
	segs = append(segs, far...)
	return append(segs, reverseEndSegs(torBridge)...)
}

// sharedRailAnchor resolves where one arm's shared-face contact rail re-enters the host's original loop,
// plus any cap-bridge segment needed to reach it. When the rail's far foot already lies on the original
// loop (a vertex or interior to an edge — the torus side of P5 and every planar host), the anchor IS the
// foot and there is no bridge (byte-identical). When the foot is an interior fresh cut the fillet itself
// makes mid-face on the shared face (P5's CURVED cyl side, mid-wall at (48.148,0.034,60); W4's PLANAR
// shared side, simple/W4's boss-notch corner — both foot and farVtx are guaranteed ON the shared face by
// construction: foot is the rail's own contact point, farVtx is a vertex of the arm's own edge, which is
// incident to the shared face), the far path cannot begin there; the anchor is the arm's FAR VERTEX (an
// original-loop vertex = shared ∩ the arm's far capping plane) and sharedFaceBridgeSeg bridges
// foot→farVertex confined to the shared face's own surface. false only when the foot is off-loop AND no
// exact bridge closes (an unsupported shared-face type or a degenerate span) — the do-no-harm floor.
//
// snapTol (not tol) gates the on-loop test. WHY THE SPLIT: a picked edge's own analytic host geometry
// (a face's stored surface parameters) and its own topology vertices are two INDEPENDENT pieces of a
// STEP-imported B-rep — they need only agree to the file's own construction precision, not to float
// noise. simple/W3, W4's fixture is measured proof: the boss cylinder's axis is stored at z=0.9999
// (CCV_1_c12gsf.rle / simple/W4.step, CARTESIAN_POINT #133/#249) while the corner vertex it must weld
// against sits at z=1 exactly (#86) — a genuine ~1e-4 gap baked into the SOURCE file, not a computation
// bug (every step from that cylinder to the torus arm's ball-tangent foot is closed-form and exact GIVEN
// that input). Projected through the arm's ball-tangent construction this ~1e-4 host/vertex disagreement
// lands the foot ~2e-5 off the ORIGINAL edge it is genuinely incident to (armBallCenter's radial term
// picks up the axis's own z-offset) — just past tol (res.Weld()·r, calibrated for float-noise
// coincidence, ~1e-9 relative) but comfortably inside snapTol (res.Sew(), THIS project's existing,
// purpose-built "close a gap between independently-derived geometry" tolerance — see resolution.go; the
// same job the Sew op already does, reused rather than a new fudge constant). Below snapTol, pointOnLoop
// correctly recognizes the foot as interior to the SAME segment the far path already walks, so no bridge
// is built at all (byte-identical to every other on-loop case) — above it (P4/P5's genuinely mid-face
// cuts, ~50 units from any vertex) the bridge still fires exactly as before. Un-fixed, the bridge foot→
// farVtx instead ran ALONGSIDE that near-coincident original segment, and the torus arm's own contact arc
// (which must reach the SAME foot) crossed back through it — miterWeldMeshDefect's self-cross/retrace,
// not a defect tol could ever safely paper over (tol stays tight for the bridge's own exactness
// certificate, sharedFaceBridgeSeg/capBridgeArc, which is a different question: IS this bridge curve
// exact, not IS this foot already part of the original loop).
func sharedRailAnchor(shared *topo.Face, side miterArmSide, vid uint64, segs []endSeg, tol, snapTol float64) (math.Point3, []endSeg, bool) {
	foot := side.railSh.from
	if pointOnLoop(segs, foot, snapTol) {
		return foot, nil, true // already on the original loop (within the model's own construction gap) — no bridge
	}
	farVtx := farVertexNotVid(side.edge, vid)
	bridge, ok := sharedFaceBridgeSeg(shared.Geometry(), foot, farVtx, tol)
	if !ok {
		return math.Point3{}, nil, false
	}
	return farVtx, []endSeg{bridge}, true
}

// pointOnLoop reports whether p lies on the loop — coincident with a vertex OR interior to an edge — using
// the SAME insertSplits/indexOfSegFrom mechanism farPathSegs anchors on, so a foot farPathSegs would accept
// is never mis-routed through a spurious cap-bridge (and an interior fresh-cut foot is always caught).
func pointOnLoop(segs []endSeg, p math.Point3, tol float64) bool {
	return indexOfSegFrom(insertSplits(segs, []math.Point3{p}, tol), p, tol) >= 0
}

// sharedFaceBridgeSeg builds sharedRailAnchor's cap-bridge on WHATEVER surface the shared face carries: a
// straight segment when it is a Plane (any two points known to lie in a plane are joined by a segment that
// lies in it too — foot is the rail's own shared-face contact point and farVtx is a vertex of the arm's
// own edge, which is incident to the shared face, so both ARE in-plane by construction; simple/W4's boss-
// notch corner is the first case that exercises this branch), or capBridgeArc's exact latitude Arc3d when
// it is a Cylinder (P5's curved shared wall). ok=false for any other shared-face type — the do-no-harm
// floor capBridgeArc already applies to the cylinder case, extended rather than loosened.
func sharedFaceBridgeSeg(shared geom.Surface, foot, farVtx math.Point3, tol float64) (endSeg, bool) {
	switch shared.(type) {
	case geom.Plane:
		return endSeg{from: foot, to: farVtx}, true
	case geom.Cylinder:
		return capBridgeArc(shared, foot, farVtx, tol)
	}
	return endSeg{}, false
}

// capBridgeArc builds the cap-bridge: the single exact Arc3d on the shared cylinder from an arm's interior
// far foot to its far vertex, both on the same latitude circle (shared ∩ the arm's far capping plane). It
// is a genuine MINOR arc through the circle midpoint (never a chord) so the shared-cyl↔cap weld stays
// watertight. ok=false when the shared face is not a cylinder, the two points are not co-latitude on its
// axis, or either is not at the cylinder radius — a curved shared face this bridge does not model floors.
func capBridgeArc(shared geom.Surface, foot, farVtx math.Point3, tol float64) (endSeg, bool) {
	cyl, ok := shared.(geom.Cylinder)
	if !ok {
		return endSeg{}, false
	}
	center := projectOntoAxis(foot, cyl.Origin, cyl.AxisDir)
	if !coLatitudeOnCyl(cyl, center, foot, farVtx, tol) {
		return endSeg{}, false
	}
	mid := arcMidBetween(center, cyl.Radius, foot, farVtx)
	arc, err := geom.Arc3dByThreePoints(foot, mid, farVtx)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: foot, to: farVtx, curve: arc, mid: mid, arc: true}, true
}

// coLatitudeOnCyl reports whether foot and farVtx lie on the SAME latitude circle of cyl — both at the
// cylinder radius from center (the axis point at foot's height) and farVtx projecting to that same center
// — so a single horizontal Arc3d bridges them. Guards capBridgeArc against a non-planar/oblique far cap.
func coLatitudeOnCyl(cyl geom.Cylinder, center, foot, farVtx math.Point3, tol float64) bool {
	if float64(center.DistanceTo(projectOntoAxis(farVtx, cyl.Origin, cyl.AxisDir))) > tol {
		return false
	}
	return stdmath.Abs(float64(foot.DistanceTo(center))-cyl.Radius) <= tol &&
		stdmath.Abs(float64(farVtx.DistanceTo(center))-cyl.Radius) <= tol
}
