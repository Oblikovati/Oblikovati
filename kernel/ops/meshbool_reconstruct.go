// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic-face reconstruction driver (ADR-0054 Layer 2c). It runs the exact
// mesh-arrangement boolean with provenance tags, then rebuilds an ANALYTIC B-rep from
// the tagged result instead of the faceted soupToBody: an untouched face (its result
// facet count equals its input count — co-refinement only splits, and an unsplit face
// is wholly kept or wholly dropped) is passed through with its exact surface and
// original edges; a split face is rebuilt on its exact surface with boundary edges
// reused from the operands' original edges. It DECLINES (returns false) when a split
// face has a boundary run that is a new surface-surface intersection curve rather than
// an original edge — those (near-tangent transversal seams) stay faceted until the SSI
// layer lands, so the change is a strict improvement over the faceted fallback.

// reconstructBoolean computes `a op b` and rebuilds it analytically, or returns false
// to leave the caller on the faceted path.
func reconstructBoolean(a, b *topo.Body, op meshbool.Op, q Quality, feat string) (*topo.Body, bool) {
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox()))
	// Cross-operand vertex-on-edge imprint: sample each operand's edges through the OTHER
	// operand's vertices that lie on them, so the two conform at a shared corner the mesh
	// co-refinement would otherwise miss (ADR-0054 / #2167 chord corner on the rim circle).
	aSoup, aRefs := taggedSoupWithImprints(a, q, 0, crossOperandImprints(a, b, res.Weld()))
	bSoup, bRefs := taggedSoupWithImprints(b, q, len(aRefs), crossOperandImprints(b, a, res.Weld()))
	refs := append(aRefs, bRefs...)
	cat := catalogOriginalEdges(a, b)

	// Fuse coincident-surface tags (cocylindrical walls, coplanar faces) so their shared
	// seam is interior to one face, not a false edge between two (ADR-0054 same-surface merge).
	rep, groupSize := mergeCoincidentTags(refs, res)
	relabelTags(aSoup.Tags, rep)
	relabelTags(bSoup.Tags, rep)
	inputCount := tagCounts(aSoup.Tags, bSoup.Tags)
	result := meshbool.BooleanTagged(aSoup, bSoup, op)
	relabelTags(result.Tags, rep)
	resultCount := tagCounts(result.Tags)

	arr := meshbool.ArrangementTopologyOf(result)
	inputs := make([]brep.ReconInput, 0, len(arr.Faces))
	for _, f := range arr.Faces {
		in, ok := reconstructFace(f, refs, len(aRefs), op, inputCount, resultCount, groupSize, arr.Verts, cat, res)
		if !ok {
			return nil, false
		}
		inputs = append(inputs, in)
	}
	body := brep.ReconstructBooleanBody(inputs)
	if body == nil || !body.IsSolid() || len(body.Faces()) == 0 {
		return nil, false
	}
	return body, true
}

// reconstructFace turns one arrangement face into a ReconInput: a pass-through for an
// untouched survivor, or a rebuilt face for a split one.
func reconstructFace(f meshbool.ArrangementFace, refs []faceSurfaceRef, naRefs int, op meshbool.Op,
	inputCount, resultCount, groupSize map[int]int, verts []meshbool.Point, cat *origEdgeCatalog, res geom.Resolution) (brep.ReconInput, bool) {
	ref := refs[f.Tag]
	fromB := f.Tag >= naRefs
	// Pass a face through only when it is a single original face kept whole. A MERGED group
	// (two coincident faces fused, groupSize>1) is never one original topo.Face, so it must be
	// rebuilt on its shared surface even when every triangle survived.
	if groupSize[f.Tag] <= 1 && resultCount[f.Tag] == inputCount[f.Tag] {
		return brep.ReconInput{PassThrough: ref.face, ForceReversed: op == meshbool.Difference && fromB}, true
	}
	loops, ok := rebuildLoops(f.Loops, ref.surface, refs, verts, cat, res)
	if !ok {
		return brep.ReconInput{}, false
	}
	return brep.ReconInput{
		Surface:  ref.surface,
		Reversed: ref.reversed != (op == meshbool.Difference && fromB),
		Loops:    loops,
		Lineage:  ref.face.Lineage(),
	}, true
}

// rebuildLoops reconstructs a split face's loops. Each run reuses an operand's original
// edge when it traces one (exact, and identical to an untouched neighbour's copy so the
// stitch welds), else it is the analytic surface-surface intersection of this face's
// surface with the run's neighbour surface. It fails only when neither yields a curve.
func rebuildLoops(loops []meshbool.ArrangementLoop, surface geom.Surface, refs []faceSurfaceRef,
	verts []meshbool.Point, cat *origEdgeCatalog, res geom.Resolution) ([]brep.ReconLoop, bool) {
	out := make([]brep.ReconLoop, 0, len(loops))
	for _, l := range loops {
		edges := make([]brep.ReconEdge, 0, len(l.Runs))
		for _, r := range l.Runs {
			e, ok := reconstructRun(r, surface, neighborSurface(refs, r.NeighborTag), verts, cat, res)
			if !ok {
				return nil, false
			}
			edges = append(edges, e)
		}
		out = append(out, brep.ReconLoop{Outer: l.Outer, Edges: edges})
	}
	return out, true
}

// neighborSurface returns the surface of the face on the far side of a run, or nil when
// the neighbour tag is invalid (an open boundary, which a watertight result never has).
func neighborSurface(refs []faceSurfaceRef, tag int) geom.Surface {
	if tag < 0 || tag >= len(refs) {
		return nil
	}
	return refs[tag].surface
}

// tagCounts tallies how many triangles carry each tag across the given tag slices.
func tagCounts(tagSlices ...[]int) map[int]int {
	counts := make(map[int]int)
	for _, tags := range tagSlices {
		for _, t := range tags {
			counts[t]++
		}
	}
	return counts
}

// runPoint rounds one run vertex to a kernel point.
func runPoint(verts []meshbool.Point, run meshbool.ArrangementRun, i int) math.Point3 {
	return verts[run.Verts[i]].Round()
}
