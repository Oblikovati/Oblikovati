// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Canal-aware arm-weld (M6' C4, architecture: .superpowers/sdd/canal-armweld-architecture.md). A
// tangent-degenerate valence-4 corner (the N7 family) has NO single concurrent corner ball: its three
// offset spines do not concur, so each arm rolls on its OWN reflected centre and the corner is a
// rolling-ball CANAL whose four boundaries weld to four distinct neighbours. The single-ball weld
// (curvedWeldFaces/armRailBundle, B3 byte-locked) is the wrong geometric model for it. This file is the
// SIBLING assembler behind a pure dispatch (ADR-C4-1): assembleCurvedArmBody routes here on
// loop.Canal != nil and the single-ball path is never edited, so B3 stays byte-identical BY CONSTRUCTION.
//
// W1 lands the seam only: the dispatch, the tagged boundary-isocurve accessor (the single source of the
// shared weld rails, ADR-C4-2), and a canalWeldFaces SKELETON that builds the corner face + declines to
// the do-no-harm floor for the still-missing arm faces (so N7 floors exactly as today, corpus unchanged).
// W2/W3/W4 fill in the per-arm-centre arm faces, the canal-aware host retrims, and the final assembly.

// canalArmBody is the canal-weld DISPATCH (ADR-C4-1), inserted as a pure prefix in assembleCurvedArmBody.
// It builds the corner ball weld from the blend sphere at vid (BEFORE the single-ball solveCurvedArmCorner,
// which since ADR-C4-4's torusStation axial guard now HONEST-REJECTS the N7 off-plane centre and would
// otherwise floor N7 before any dispatch), extracts the corner RailLoop, and — only when that loop is the
// tangent-degenerate valence-4 canal (loop.Canal != nil) — hands the weld to canalWeldFaces and reports
// took=true (with the welded body, or the do-no-harm floor reason). Every concurrent-spine corner (B3 +
// green corpus) extracts Canal==nil (the octant branch needs w.arms this preliminary weld lacks) and
// returns took=false, so the untouched single-ball path below is reached byte-identically.
func canalArmBody(body *topo.Body, arms []edgeFillet, blends map[uint64]*cornerBlend, vid uint64, res Resolution) (*topo.Body, string, bool) {
	cb, ok := blends[vid]
	if !ok {
		return nil, "", false // no corner ball solved here — leave it to the single-ball solve to diagnose
	}
	w := cornerWeld{center: cb.sphere.Center, radius: cb.sphere.Radius}
	// extractTangentDegenerateCorner is the CANAL-specific extractor (the sole setter of loop.Canal); it
	// declines gracefully to false for a concurrent octant. Calling the general extractCurvedCorner here
	// would fall through to its octant branch, which needs w.arms this preliminary weld deliberately
	// lacks (they are the single-ball solve's product) and would panic in chainSetbackArcs on the empty set.
	loop, ok := extractTangentDegenerateCorner(w, arms, res)
	if !ok || loop.Canal == nil {
		return nil, "", false // concurrent-spine (octant) corner — the single-ball path owns it
	}
	faces, reason := canalWeldFaces(body, arms, w, loop, res)
	if reason != "" {
		return nil, reason, true // honest decline → the clean do-no-harm floor (never a partial body)
	}
	return assembleBody(faces), "", true
}

// canalWeldFaces assembles the canal corner into a watertight solid's faces, or names WHY the weld
// declined (empty reason = watertight). It resolves the corner patch ONCE (ADR-C4-2, the single source
// of the boundary curves), tags the four shared rails (canalBoundaryRoles) and the per-arm reflected
// centres (reflectedArmCentres, ADR-C4-3) — computed ONCE here and threaded into the arm-face builder,
// never recomputed — then builds the corner canal-patch face + the three per-arm-centre arm faces (W2).
// It still floors to the do-no-harm floor for the missing canal host retrims + far-runout (W3-W4), so
// N7 stays declined (corpus unchanged) but with a diagnostic canal-specific reason. Example:
//
//	if faces, reason := canalWeldFaces(body, arms, w, loop, res); reason == "" { /* watertight weld */ }
func canalWeldFaces(body *topo.Body, arms []edgeFillet, w cornerWeld, loop RailLoop, res Resolution) ([]filletFace, string) {
	patch, boundaries, centres, scale, reason := canalWeldContext(loop, w, arms, res)
	if reason != "" {
		return nil, reason
	}
	armFaces, reason := canalArmFaces(arms, w, boundaries, centres, scale, res)
	if reason != "" {
		return nil, reason
	}
	bundles, ok := canalArmBundles(arms, w, centres, scale, res)
	if !ok {
		return nil, "canal arm far cross-sections unresolved (far-runout bundle)"
	}
	hostFaces, reason := canalHostFaces(body, w, boundaries, bundles, loop.Canal.Rolls, res)
	if reason != "" {
		return nil, reason
	}
	faces := assembleCanalFaces(body, patch, armFaces, hostFaces)
	return faces, "canal final weld not yet assembled (W3: corner patch + per-arm-centre arm faces + host retrims/far-runout ready; whole-body assembly + Σ verification pending W4)"
}

// canalWeldContext resolves the corner patch ONCE (ADR-C4-2, the single source of the boundary curves)
// and derives the tagged boundary isocurves, the per-arm reflected centres, and the model scale — the
// shared inputs the arm-face + host-retrim builders thread. Non-empty reason names WHY it declined.
func canalWeldContext(loop RailLoop, w cornerWeld, arms []edgeFillet, res Resolution) (CornerBlendPatch, canalBoundaries, []math.Point3, float64, string) {
	patch, ok := resolveBlend(loop, res)
	if !ok {
		return CornerBlendPatch{}, canalBoundaries{}, nil, 0, "canal corner patch declined (resolveBlend honest-reject)"
	}
	boundaries, err := canalBoundaryRoles(patch)
	if err != nil {
		return CornerBlendPatch{}, canalBoundaries{}, nil, 0, fmt.Sprintf("canal boundary roles unavailable: %v", err)
	}
	scale := tangentCornerScale(w, arms)
	centres, ok := reflectedArmCentres(w, arms, scale, res)
	if !ok {
		return CornerBlendPatch{}, canalBoundaries{}, nil, 0, "canal per-arm reflected centres unresolved"
	}
	return patch, boundaries, centres, scale, ""
}

// assembleCanalFaces gathers the canal weld's faces in assembly order: the corner canal-patch face, the
// three per-arm-centre arm faces, then the retrimmed/far-runout host faces (empty on a face-less body).
func assembleCanalFaces(body *topo.Body, patch CornerBlendPatch, armFaces, hostFaces []filletFace) []filletFace {
	faces := make([]filletFace, 0, len(body.Faces())+len(armFaces)+1)
	faces = append(faces, patchToFilletFace(patch, topo.Lineage{}))
	faces = append(faces, armFaces...)
	return append(faces, hostFaces...)
}

// canalArmBundle is one arm's per-reflected-centre weld rails the canal HOST retrims collect (W3b): the
// far cross-section arc (the rail shared with the far-runout hosts, canalFarOrPassthrough → farArcsBiting)
// PLUS the two host contact rails, each tagged with the host face it lies on. canalHostBite COLLECTS an
// arm's already-built host rail from here (shared-edge identity for free) instead of rebuilding it at a
// shared w.center — which is exactly why the single-ball retrimCornerHost is not reusable: its rails come
// from one shared centre, but the canal arms have DIFFERENT reflected centres (architecture §"Assembly
// decision"). rails[k] lies on hosts[k]; rails[0] is on arm.a, rails[1] on arm.b (canalArmHostRails' order).
type canalArmBundle struct {
	far   endSeg
	rails [2]endSeg
	hosts [2]*topo.Face
}

// canalArmBundles builds each arm's per-reflected-centre rail bundle (far arc + the two host rails) at its
// reflected centre, reusing the W2 machinery (solveArmSetback / canalArmHostRails / farCrossSectionArc) so
// every rail is byte-identical to the one the arm FACE closes on (shared-edge identity). Declines (false)
// if any arm's rails cannot be built at its reflected centre.
func canalArmBundles(arms []edgeFillet, w cornerWeld, centres []math.Point3, scale float64, res Resolution) ([]canalArmBundle, bool) {
	bundles := make([]canalArmBundle, len(arms))
	for i := range arms {
		b, ok := canalArmBundle1(arms[i], centres[i], w, scale, res)
		if !ok {
			return nil, false
		}
		bundles[i] = b
	}
	return bundles, true
}

// canalArmBundle1 solves ONE arm at its reflected centre (a one-arm local cornerWeld) and collects its far
// cross-section arc + the two host contact rails, tagged with the host faces they lie on. Declines (false)
// if the setback, the host rails, or the far arc cannot be built at this centre.
func canalArmBundle1(arm edgeFillet, centre math.Point3, w cornerWeld, scale float64, res Resolution) (canalArmBundle, bool) {
	set, ok := solveArmSetback(arm, centre, w.radius, scale, res)
	if !ok {
		return canalArmBundle{}, false
	}
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, ok := canalArmHostRails(arm, set, wi, res)
	if !ok {
		return canalArmBundle{}, false
	}
	far, ok := farCrossSectionArc(set.arm, w.radius, h0.from, h1.from)
	if !ok {
		return canalArmBundle{}, false
	}
	return canalArmBundle{far: far, rails: [2]endSeg{h0, h1}, hosts: [2]*topo.Face{arm.a, arm.b}}, true
}

// canalBoundaries is the canal patch's four boundary isocurves tagged by ROLE (ADR-C4-2, the SINGLE
// source of the shared weld rails W2-W4 consume): the two END ARCS (the v=v0/v=v1 cross-section arcs at
// the wall-sharing arms' reflected ball centres — N7: C″ and C) and the two FOOT-LOCI (the u=u0 curve on
// the wall roll host R=50 and the u=u1 curve on the mid arm's s_10 roll host R=5). Every neighbour that
// shares a boundary samples the SAME curve object the SAME way, so assembleBody welds without a crack.
type canalBoundaries struct {
	endArcs [2]geom.Curve3 // v=v0, v=v1 cross-section arcs (@ the two reflected ball centres)
	feet    [2]geom.Curve3 // feet[0] = u=u0 foot-locus on the wall; feet[1] = u=u1 foot-locus on s_10
	// endArcsRev/feetRev carry each boundary's canalPatchLoops sampling direction (the canalRingSide.rev
	// flag) so a neighbour arm face samples the SHARED curve the SAME way the corner patch does — the
	// point-for-point identity that welds them watertight (ADR-C4-2). Parallel to endArcs/feet by index.
	endArcsRev [2]bool
	feetRev    [2]bool
}

// canalBoundaryRoles tags the canal patch's four boundary isocurves by role, reusing canalBoundaryIsocurves
// (the single extractor in corner_provider_canal.go) so the curves are the SAME objects the patch loops
// sample (ADR-C4-2). The role is fixed by iso-DIRECTION, a structural property of the canal loft
// parametrisation (u runs AROUND the rolling-ball cross-section, v runs ALONG the spine): a v-boundary
// (fixed v) is a cross-section END ARC, a u-boundary (fixed u) is a FOOT-LOCUS on a roll host.
// canalBoundaryIsocurves returns the sides in closed-ring order [v0, u1, v1, u0], so endArcs = {v0, v1}
// and feet = {u0 (wall), u1 (s_10)}. Errors (carrying the offending surface type / underlying error) when
// the patch surface is not the canal BSpline or the isocurve extraction declines.
func canalBoundaryRoles(patch CornerBlendPatch) (canalBoundaries, error) {
	surf, ok := patch.Surface.(geom.BSplineSurface)
	if !ok {
		return canalBoundaries{}, fmt.Errorf("canalBoundaryRoles: patch surface is %T, want geom.BSplineSurface (a canal patch)", patch.Surface)
	}
	sides, err := canalBoundaryIsocurves(surf)
	if err != nil {
		return canalBoundaries{}, fmt.Errorf("canalBoundaryRoles: %w", err)
	}
	if len(sides) != 4 {
		return canalBoundaries{}, fmt.Errorf("canalBoundaryRoles: got %d boundary isocurves, want 4 (canal patch)", len(sides))
	}
	return canalBoundaries{
		endArcs:    [2]geom.Curve3{sides[0].curve, sides[2].curve}, // v=v0, v=v1 cross-section arcs
		endArcsRev: [2]bool{sides[0].rev, sides[2].rev},
		feet:       [2]geom.Curve3{sides[3].curve, sides[1].curve}, // u=u0 (wall), u=u1 (s_10) foot-loci
		feetRev:    [2]bool{sides[3].rev, sides[1].rev},
	}, nil
}
