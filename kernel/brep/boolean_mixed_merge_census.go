// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
)

// What the merge saw, so the seam-slit test is measured rather than assumed (Oblikovati#3521).
//
// isReverseTwin reads an edge's IDENTITY — the topo edge it walks — because a curve VALUE comparison
// panics on a value Polyline and because two edges carrying equal curves are still two edges. That
// makes every SYNTHESIZED loop edge nobody's twin, and the (u,v) arrangement re-emits its whole
// boundary, so a face that came through it names no topo edge anywhere. The narrowing is therefore
// correct and silent at once: a boundary that reaches the merge entirely source-less can never lose a
// slit, and nothing said so. These two Info notes are the positive markers that separate "no slit was
// there" from "the test could not have seen one", the same job CodeAssembleEdgeCatalog does for the
// fillet edge catalog.

// CodeMergeEdgeSources reports how many boundary edges reached the cocylindrical merge and how many of
// them name no source edge — the population the seam-slit test can and cannot pair.
const CodeMergeEdgeSources diag.Code = "boolean.merge-edge-sources"

// CodeSeamSlitDropped marks a COMMITTED merge that dropped a seam its dissolve orphaned: the face's own
// two traversals of one edge became adjacent, and a curve walked up and straight back down bounds
// nothing. It is the positive marker that the removal fired in a real boolean.
const CodeSeamSlitDropped diag.Code = "boolean.seam-slit-dropped"

// recordEdgeSourceCensus counts the boundary edges reaching the merge and how many carry no source
// edge. It is emitted on EVERY call, the all-sourced and the empty one included: a census that speaks
// only when it has something to say cannot tell a zero from a merge that never ran.
func recordEdgeSourceCensus(rec *diag.Recorder, faces []curvedFace) {
	if rec == nil {
		return // not for safety — Recordf is nil-safe — but to skip the walk when nobody is listening
	}
	edges, sourceless := edgeSourceCounts(faces)
	rec.Recordf(CodeMergeEdgeSources, diag.Info,
		"%d of the %d boundary edges on the %d faces reaching the cocylindrical merge name no source "+
			"edge; a source-less edge is nobody's seam twin", sourceless, edges, len(faces))
}

// edgeSourceCounts is how many boundary edges reach the merge, and how many of them name no topo edge.
func edgeSourceCounts(faces []curvedFace) (edges, sourceless int) {
	for _, f := range faces {
		edges += countLoopEdges(f)
		sourceless += sourcelessEdgesOf(f)
	}
	return edges, sourceless
}

// sourcelessEdgesOf counts a face's SYNTHESIZED boundary edges — the ones re-emitted by an arrangement
// rather than resolved from a topo edge use, which isReverseTwin can never pair.
func sourcelessEdgesOf(f curvedFace) int {
	n := 0
	for _, l := range f.loops {
		n += sourcelessEdgesInLoop(l)
	}
	return n
}

// sourcelessEdgesInLoop counts one loop's source-less edges.
func sourcelessEdgesInLoop(l curvedLoop) int {
	n := 0
	for _, e := range l.edges {
		if e.source == nil {
			n++
		}
	}
	return n
}

// recordSeamSlitDrop names the seam a committed merge orphaned. It fires only after chartedMerge
// accepted the pair, so the count is what SHIPPED: a pair that dissolves on every rescan and then
// refuses cannot report the same drop once per pass, the repetition mergeCoincidentFaces already
// avoids for its declines.
func recordSeamSlitDrop(rec *diag.Recorder, a curvedFace, dropped int) {
	if dropped == 0 {
		return
	}
	// The corpus row asserts the COUNT through this exact wording ("dropped N orphaned seam edge"),
	// because diag.Diagnostic carries no structured payload; reword it and re-word the row with it.
	rec.Recordf(CodeSeamSlitDropped, diag.Info,
		"the merged %T face dropped %d orphaned seam edge(s): the dissolve left the face's own two "+
			"traversals of one edge adjacent, and a curve walked straight back bounds nothing",
		a.surface, dropped)
}
