// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// filletResultFaces builds the faces of the filleted body — every original face transformed for the
// fillets touching it (A/B corners pulled to tangent points, simple end corners replaced by an arc or
// chord fan), one cylinder face per constant filleted edge (or a ruling strip per variable one), and
// one sphere patch per corner blend — and reports whether either enabled local rebuild (the mid-span
// obstacle notch, ADR-4, or the double-interference runout tiling, ADR-5) handled any edge.
// enableObstacles/enableRunout independently gate the two rebuilds so a caller can assemble any of the
// four compositions (baseline, obstacle-only, runout-only, both) and let each clear the do-no-harm bar
// on its own — a failing one must never veto a passing other (M2 whole-branch review, systemic minor).
func filletResultFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, enableObstacles, enableRunout bool) ([]filletFace, bool) {
	maps, caps := filletBuildMaps(body, fils)
	replace, extra, handled := map[uint64]filletFace{}, map[uint64][]filletFace{}, map[uint64]bool{}
	if enableObstacles || enableRunout {
		replace, extra, handled = collectRebuildFaces(body, fils, ResolutionForBody(body), maps, caps, enableObstacles, enableRunout)
	}
	// The far-end multi-face trim (fillet_farend_chain.go) rebuilds every host its contact chain touches,
	// from that host's ORIGINAL ring — so it must run after the obstacle/runout rebuilds have claimed
	// theirs, and it declines outright on a collision rather than half-applying (an unclosed shell).
	for id, ff := range commitFarEndSplits(body, fils, replace, handled) {
		replace[id] = ff
	}
	out := transformedBodyFaces(body, maps, replace)
	out = append(out, filletBlendFaces(fils, caps, handled, extra)...)
	for _, cb := range blends {
		out = append(out, spherePatchFace(cb))
	}
	return out, len(handled) > 0
}

// filletBuildMaps builds the per-face substitution/insert/spread maps and end-corner cap pieces every
// rebuild composition shares — independent of which local rebuild(s), if any, are enabled.
func filletBuildMaps(body *topo.Body, fils []edgeFillet) (filletRebuildMaps, map[uint64][]cornerPiece) {
	abSubst, endCorner, edgeInserts := filletMaps(fils)
	fans, fanV := classifyEndCorners(fils)
	spreads, caps := buildSpreadMaps(fans, body)
	pruneEndCorners(endCorner, fanV) // a fan vertex is rounded by the spread arm alone, never as a trihedral end
	return filletRebuildMaps{abSubst: abSubst, endCorner: endCorner, edgeInserts: edgeInserts,
		insertCurves: map[*topo.Face]map[uint64][]geom.Curve3{}, spreads: spreads}, caps
}

// collectRebuildFaces runs the ENABLED local fillet rebuild(s) — the mid-span obstacle notch (ADR-4),
// the band∩obstacle imprint walk (fillet_band_imprint.go) and/or the double-interference runout tiling
// (ADR-5) — and merges their face substitutions, extra faces and handled-edge sets into one lookup.
// Each later path skips any edge an earlier one already owns, so an edge is rebuilt by exactly one.
func collectRebuildFaces(body *topo.Body, fils []edgeFillet, res Resolution, maps filletRebuildMaps,
	caps map[uint64][]cornerPiece, enableObstacles, enableRunout bool) (
	map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	replace, extra, handled := map[uint64]filletFace{}, map[uint64][]filletFace{}, map[uint64]bool{}
	if enableObstacles {
		replace, extra, handled = collectObstacles(body, fils, res, maps)
		replace, extra, handled = collectBandImprints(body, fils, maps, caps, replace, extra, handled)
	}
	if !enableRunout {
		return replace, extra, handled
	}
	return mergeRunoutFaces(body, fils, res, maps, replace, extra, handled)
}

// mergeRunoutFaces runs collectRunouts and folds its result into the (possibly still-empty)
// obstacle-path maps, so an edge already owned by the obstacle rebuild is never double-handled.
func mergeRunoutFaces(body *topo.Body, fils []edgeFillet, res Resolution, maps filletRebuildMaps,
	replace map[uint64]filletFace, extra map[uint64][]filletFace, handled map[uint64]bool) (
	map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	rnReplace, rnExtra, rnHandled := collectRunouts(body, fils, res, handled, maps)
	for id, f := range rnReplace {
		replace[id] = f
	}
	for id, fs := range rnExtra {
		extra[id] = fs
	}
	for id := range rnHandled {
		handled[id] = true
	}
	return replace, extra, handled
}

// transformedBodyFaces transforms every original body face for the fillets touching it, except that a
// face the mid-span obstacle rebuild replaced (ADR-4) is substituted by its notched / split-wall face.
func transformedBodyFaces(body *topo.Body, maps filletRebuildMaps, obReplace map[uint64]filletFace) []filletFace {
	scale := ResolutionForBody(body).Size() // model scale for the subs-branch survivor-arc-carry gate (I3)
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		if notched, ok := obReplace[f.ID()]; ok {
			out = append(out, notched) // host notch / split obstacle wall replaces the default transform
			continue
		}
		out = append(out, transformFace(f, maps.forFace(f, scale)))
	}
	return out
}

// filletBlendFaces builds the blend face(s) of each filleted edge: an obstacle edge (ADR-4) contributes
// its pre-built two wings + corner-blend patch; a variable edge a ruled/strip blend; a constant edge one
// cylinder face. A non-obstacle edge takes the same path byte-for-byte as before ADR-4.
func filletBlendFaces(fils []edgeFillet, caps map[uint64][]cornerPiece, obHandled map[uint64]bool, obExtra map[uint64][]filletFace) []filletFace {
	var out []filletFace
	for _, ef := range fils {
		switch {
		case obHandled[ef.edge.ID()]:
			out = append(out, obExtra[ef.edge.ID()]...) // two wings + the corner-blend patch
		case ef.varying && ef.exact:
			out = append(out, ruledBlendFaces(ef)...)
		case ef.varying:
			out = append(out, rulingStripFaces(ef)...)
		default:
			out = append(out, cylinderFace(ef, caps))
		}
	}
	return out
}
