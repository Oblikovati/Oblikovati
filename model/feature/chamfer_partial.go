// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bytes"
	stdmath "math"

	"oblikovati.org/kernel/topo"
)

// Which face a chamfer's first setback lands on, and how much of the edge it covers
// (Oblikovati#1888).
//
// An asymmetric chamfer — two distances, or a distance and an angle — is only meaningful once you
// say WHICH face the first distance is measured on. Before this the answer was `edge.Faces()[0]`:
// an artefact of how the topology happened to be built, not anything the author chose. On mirrored
// geometry that order can differ between two edges that look identical, so the larger setback
// landed on the wrong face and quietly changed the part. Naming the reference face makes the
// assignment the author's, and deterministic.
//
// A PARTIAL chamfer bevels only a span of the edge. That is the same prism, cut over a shorter run
// — with one wrinkle: the tool normally overhangs both ends so its cut meets the neighbouring faces
// flush, but a partial chamfer's ends sit INSIDE material, where an overhang would bevel a little
// more edge than was asked for. The overhang is therefore taken only at an end that is the edge's
// own end (the same rule a from-to hole's entry follows, #1863).

// chamferRun is how much of an edge one chamfer covers and which face its first setback belongs to.
// The zero value is the whole edge with the setbacks in the edge's own face order — what every
// chamfer authored before #1888 means.
type chamferRun struct {
	start, length float64 // 0,0 ⇒ the whole edge
	reference     []byte  // lineage key of the face d1 is measured on; empty ⇒ the edge's first face
}

// isPartial reports whether the run covers less than the whole edge.
func (r chamferRun) isPartial() bool { return r.length > 0 }

// runOf reads the run off a definition.
func (d *ChamferDefinition) runOf() chamferRun {
	return chamferRun{
		start: callOrZero(d.PartialStart), length: callOrZero(d.PartialLength),
		reference: d.ReferenceFace,
	}
}

// orderedSetbacks puts d1 on the face the author named. Without a reference face the edge's own
// order stands, which is what every chamfer written before #1888 relied on.
func orderedSetbacks(edge *topo.Edge, d1, d2 float64, reference []byte) (float64, float64) {
	if len(reference) == 0 {
		return d1, d2
	}
	faces := edge.Faces()
	if len(faces) == 2 && bytes.Equal(faces[1].ReferenceKey(), reference) {
		return d2, d1 // the reference is the SECOND face, so the first setback belongs to it
	}
	return d1, d2
}

// wedgeSpan is the length of edge the cut tool covers. A full-edge chamfer overhangs both ends so
// the boolean meets the neighbouring faces flush; a partial one overhangs only where it reaches a
// real end of the edge, since an interior end sits in material the chamfer was not asked to touch.
func wedgeSpan(fr edgeFrame, run chamferRun, overhang float64) span {
	if !run.isPartial() {
		return span{near: -overhang, far: fr.length + overhang}
	}
	near := stdmath.Max(0, run.start)
	far := stdmath.Min(fr.length, near+run.length)
	return span{near: endOverhang(near, 0, overhang), far: endOverhang(far, fr.length, overhang)}
}

// endOverhang extends one end of the run past the edge only when it IS the edge's end.
func endOverhang(at, edgeEnd, overhang float64) float64 {
	if stdmath.Abs(at-edgeEnd) > chamferRunTol {
		return at // an interior end: stop exactly where the chamfer was asked to stop
	}
	if edgeEnd == 0 {
		return -overhang
	}
	return edgeEnd + overhang
}

// chamferRunTol is how near a partial chamfer's end must come to the edge's end to count as
// reaching it. Relative to nothing — it is compared against a length in database units, and the
// chamfer's own overhang is the same order — so it is deliberately small and absolute.
const chamferRunTol = 1e-9
