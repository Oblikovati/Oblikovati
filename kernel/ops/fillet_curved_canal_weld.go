// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
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
// declined (empty reason = watertight). W1 SKELETON: it resolves the corner patch ONCE (ADR-C4-2, the
// single source of the boundary curves), gates on the tagged shared rails (canalBoundaryRoles) and the
// per-arm reflected centres (reflectedArmCentres, ADR-C4-3 — the single source W2 consumes), builds the
// corner canal-patch face, and then declines to the do-no-harm floor for the still-missing per-arm arm
// faces + host retrims (W2-W4). So N7 floors exactly as today (corpus unchanged) but with a diagnostic
// canal-specific reason. Example:
//
//	if faces, reason := canalWeldFaces(body, arms, w, loop, res); reason == "" { /* watertight weld */ }
func canalWeldFaces(body *topo.Body, arms []edgeFillet, w cornerWeld, loop RailLoop, res Resolution) ([]filletFace, string) {
	patch, ok := resolveBlend(loop, res)
	if !ok {
		return nil, "canal corner patch declined (resolveBlend honest-reject)"
	}
	if _, err := canalBoundaryRoles(patch); err != nil {
		return nil, fmt.Sprintf("canal boundary roles unavailable: %v", err)
	}
	if _, ok := reflectedArmCentres(w, arms, tangentCornerScale(w, arms), res); !ok {
		return nil, "canal per-arm reflected centres unresolved"
	}
	faces := make([]filletFace, 0, len(body.Faces())+len(arms)+1)
	faces = append(faces, patchToFilletFace(patch, topo.Lineage{}))
	return faces, "canal arm faces not yet assembled (W1 skeleton: corner patch + tagged rails + per-arm centres ready; per-arm faces + host retrims pending W2-W4)"
}

// canalBoundaries is the canal patch's four boundary isocurves tagged by ROLE (ADR-C4-2, the SINGLE
// source of the shared weld rails W2-W4 consume): the two END ARCS (the v=v0/v=v1 cross-section arcs at
// the wall-sharing arms' reflected ball centres — N7: C″ and C) and the two FOOT-LOCI (the u=u0 curve on
// the wall roll host R=50 and the u=u1 curve on the mid arm's s_10 roll host R=5). Every neighbour that
// shares a boundary samples the SAME curve object the SAME way, so assembleBody welds without a crack.
type canalBoundaries struct {
	endArcs [2]geom.Curve3 // v=v0, v=v1 cross-section arcs (@ the two reflected ball centres)
	feet    [2]geom.Curve3 // feet[0] = u=u0 foot-locus on the wall; feet[1] = u=u1 foot-locus on s_10
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
		endArcs: [2]geom.Curve3{sides[0].curve, sides[2].curve}, // v=v0, v=v1 cross-section arcs
		feet:    [2]geom.Curve3{sides[3].curve, sides[1].curve}, // u=u0 (wall), u=u1 (s_10) foot-loci
	}, nil
}
