// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// Two kept faces of one boolean result that lie on ONE surface and share a boundary are ONE face
// (ADR-0061 stage 5). A boss whose wall is cocylindrical with its host's, a bore that continues a
// bore, two coaxial rods joined end to end: the classification keeps a band from each operand, and
// the curve where they meet bounds nothing — the surface is smooth across it. Leaving it there
// invents an edge the model does not have, which a user can select and a fillet would try to round.
//
// The merge is COMBINATORIAL. The pair is decided by geom.SurfacesCoincide, the same "one surface"
// answer the radial sew uses, and the shared boundary by the edges the two walk together; no radius,
// axis or azimuth is compared here, and no coordinate moves. What the merged face is, is settled by
// following each loop and CROSSING at every dissolved edge into the other face's loop — a rule that
// needs no geometry at a vertex, which matters because the vertex where two cocylindrical walls meet
// can carry four seam ends at one point and a "turn left" rule would have to pick among them.
//
// ADR-0061 recorded an earlier reading of this: "their common boundary is part of the cylinder's rim,
// not a whole edge of it". That was wrong. The boundary IS whole edges — TWO of them, because the
// receiving wall's own seam ruling splits the run it shares, and the splice this replaced merged only
// when exactly ONE edge was shared. Dissolving a SET of edges is what the configuration needs, plus
// the rule below for the seam the set orphans.

// mergeCoincidentFaces joins result faces on ONE surface until no two of them share a boundary, and
// reports every pair it could not join.
//
// The declines are reported from the LAST scan only. Merging one pair restarts the scan, so a pair
// that refuses would otherwise be reported again on every pass, and a diagnostic repeated N times says
// nothing the first one did not.
//
// Example:
//
//	faces = mergeCoincidentFaces(faces, rec) // two coaxial cylinder bands become one wall
func mergeCoincidentFaces(faces []curvedFace, rec *diag.Recorder) []curvedFace {
	for {
		next, declined, merged := mergeFirstPair(faces)
		if !merged {
			reportDeclines(rec, declined)
			return faces
		}
		faces = next
	}
}

// reportDeclines names every pair that shares a boundary the merge could not dissolve.
func reportDeclines(rec *diag.Recorder, declined []declinedMerge) {
	for _, d := range declined {
		recordMergeDecline(rec, d.on, d.why)
	}
}

// declinedMerge is one pair that IS one face and could not be made one, kept until the scan finishes.
type declinedMerge struct {
	on  curvedFace
	why mergeDecline
}

// mergeFirstPair merges the first mergeable pair in index order — never in map or pointer order, so
// the result is the same body on every run — and returns the declines it passed on the way.
func mergeFirstPair(faces []curvedFace) ([]curvedFace, []declinedMerge, bool) {
	var declined []declinedMerge
	for i := range faces {
		j, joined, seen, ok := firstMergeableWith(faces, i)
		declined = append(declined, seen...)
		if !ok {
			continue
		}
		faces[i] = joined
		return append(faces[:j], faces[j+1:]...), nil, true
	}
	return faces, declined, false
}

// firstMergeableWith returns the lowest-indexed face after i that merges with it, and every reportable
// refusal it met before that.
func firstMergeableWith(faces []curvedFace, i int) (int, curvedFace, []declinedMerge, bool) {
	var declined []declinedMerge
	for j := i + 1; j < len(faces); j++ {
		joined, why := mergePairOnOneSurface(faces[i], faces[j])
		if why == mergeJoined {
			return j, joined, declined, true
		}
		if why.reportable() {
			declined = append(declined, declinedMerge{on: faces[i], why: why})
		}
	}
	return 0, curvedFace{}, declined, false
}

// mergePairOnOneSurface merges two faces across every edge they share, and names its reason when it
// does not. A pair that is not a candidate at all — different sense, different surface — gives the
// ordinary unshared reason, which is the only one that is never reported.
func mergePairOnOneSurface(a, b curvedFace) (curvedFace, mergeDecline) {
	if a.reversed != b.reversed || !onOneSurface(a, b) {
		return curvedFace{}, declineUnshared
	}
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	loops, why := dissolveSharedEdges(a, b, res)
	if why != mergeJoined {
		return curvedFace{}, why
	}
	return chartedMerge(a, b, loops)
}

// mergeOnSharedBoundary is mergePairOnOneSurface with its refusal reported at once — the single-pair
// entry the merge's own rows drive, where there is no later scan to report from.
func mergeOnSharedBoundary(a, b curvedFace, rec *diag.Recorder) (curvedFace, bool) {
	merged, why := mergePairOnOneSurface(a, b)
	recordMergeDecline(rec, a, why)
	return merged, why == mergeJoined
}

// onOneSurface is the "same surface" decision the radial sew already makes (ADR-0058): surface
// identity, not a tolerance on radii or axes.
func onOneSurface(a, b curvedFace) bool {
	return geom.SurfacesCoincide(a.surface, b.surface, geom.ResolutionForBox(faceLoopBox(a)))
}

// chartedMerge assembles the merged face: a's identity, the fused loops, both parents' reference keys,
// and the chart those loops determine — the union of the two trims in the covering space, taken on the
// branch loopToUV unwraps the FIRST loop onto (ADR-0063). A chart the loops do not determine is a
// named decline, not a guess.
//
// The complement flag is the one datum the fused loops CANNOT carry — it is precisely what ADR-0063
// says a face's rings do not determine — so it is taken from the parents, and a pair that disagrees on
// it is refused rather than given a's answer.
func chartedMerge(a, b curvedFace, loops []curvedLoop) (curvedFace, mergeDecline) {
	if a.outerless != b.outerless {
		return curvedFace{}, declineMixedComplement
	}
	out := a
	out.loops = loops
	out.chart = nil // the merged trim is neither operand's chart; the fused loops determine it
	out.aliasKeys = mergedAliasKeys(a, b)
	uPer, vPer := surfacePeriodic(out.surface)
	chart, ok := faceChart(out, uPer, vPer)
	if !ok {
		return curvedFace{}, declineUndecidedChart
	}
	out.chart = chart
	return out, mergeJoined
}

// mergedAliasKeys is every reference key that resolved to either parent, so a pick on either survives
// the merge (ADR-0043/ADR-0057). a's own key stays the merged face's lineage.
func mergedAliasKeys(a, b curvedFace) [][]byte {
	out := append([][]byte(nil), a.aliasKeys...)
	if len(b.lineage.Key()) > 0 {
		out = append(out, b.lineage.Key())
	}
	return append(out, b.aliasKeys...)
}
