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

// CodeCocylindricalMergeUndecided marks two faces on one surface whose shared boundary dissolved but
// whose merged parametric trim its fused loops do not determine (ADR-0063 refuses to guess a side).
// The pair is left unmerged — two faces where one belongs — and this says so rather than shipping a
// chart nothing verified.
const CodeCocylindricalMergeUndecided diag.Code = "boolean.cocylindrical-merge-undecided"

// mergeCoincidentFaces joins result faces on ONE surface until no two of them share a boundary.
//
// Example:
//
//	faces = mergeCoincidentFaces(faces, rec) // two coaxial cylinder bands become one wall
func mergeCoincidentFaces(faces []curvedFace, rec *diag.Recorder) []curvedFace {
	for {
		next, merged := mergeFirstPair(faces, rec)
		if !merged {
			return faces
		}
		faces = next
	}
}

// mergeFirstPair merges the first mergeable pair in index order — never in map or pointer order, so
// the result is the same body on every run — and reports whether it found one.
func mergeFirstPair(faces []curvedFace, rec *diag.Recorder) ([]curvedFace, bool) {
	for i := range faces {
		j, joined, ok := firstMergeableWith(faces, i, rec)
		if !ok {
			continue
		}
		faces[i] = joined
		return append(faces[:j], faces[j+1:]...), true
	}
	return faces, false
}

// firstMergeableWith returns the lowest-indexed face after i that merges with it.
func firstMergeableWith(faces []curvedFace, i int, rec *diag.Recorder) (int, curvedFace, bool) {
	for j := i + 1; j < len(faces); j++ {
		if joined, ok := mergeOnSharedBoundary(faces[i], faces[j], rec); ok {
			return j, joined, true
		}
	}
	return 0, curvedFace{}, false
}

// mergeOnSharedBoundary merges two faces across every edge they share. ok=false when they are not on
// one surface, differ in sense, share no edge, or the fused loops do not determine a chart.
func mergeOnSharedBoundary(a, b curvedFace, rec *diag.Recorder) (curvedFace, bool) {
	if a.reversed != b.reversed || !onOneSurface(a, b) {
		return curvedFace{}, false
	}
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	loops, ok := dissolveSharedEdges(a, b, res)
	if !ok {
		return curvedFace{}, false
	}
	return chartedMerge(a, b, loops, rec)
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
func chartedMerge(a, b curvedFace, loops []curvedLoop, rec *diag.Recorder) (curvedFace, bool) {
	out := a
	out.loops = loops
	out.chart = nil // the merged trim is neither operand's chart; the fused loops determine it
	out.aliasKeys = mergedAliasKeys(a, b)
	uPer, vPer := surfacePeriodic(out.surface)
	chart, ok := faceChart(out, uPer, vPer)
	if !ok {
		rec.Recordf(CodeCocylindricalMergeUndecided, diag.Defect,
			"two %T faces on one surface share a boundary, but the %d loops it fuses them into do not "+
				"determine a trim in (u,v); left as two faces", a.surface, len(loops))
		return curvedFace{}, false
	}
	out.chart = chart
	return out, true
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
