// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic-face reconstruction driver (ADR-0056 Layer 2c). It runs the exact
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
func reconstructBoolean(a, b *topo.Body, op meshbool.Op, q Quality) (*topo.Body, bool) {
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox()))
	// Cross-operand vertex-on-edge imprint: sample each operand's edges through the OTHER
	// operand's vertices that lie on them, so the two conform at a shared corner the mesh
	// co-refinement would otherwise miss (ADR-0056 / #2167 chord corner on the rim circle).
	aSoup, aRefs := taggedSoupWithImprints(a, q, 0, crossOperandImprints(a, b, res.Weld()))
	bSoup, bRefs := taggedSoupWithImprints(b, q, len(aRefs), crossOperandImprints(b, a, res.Weld()))
	refs := make([]faceSurfaceRef, 0, len(aRefs)+len(bRefs))
	refs = append(append(refs, aRefs...), bRefs...)

	// Fuse coincident-surface tags (cocylindrical walls, coplanar faces) so their shared
	// seam is interior to one face, not a false edge between two (ADR-0056 same-surface merge).
	rep, groupSize := mergeCoincidentTags(refs, res)
	relabelTags(aSoup.Tags, rep)
	relabelTags(bSoup.Tags, rep)
	inputCount := tagCounts(aSoup.Tags, bSoup.Tags)
	result := meshbool.BooleanTagged(aSoup, bSoup, op)
	result = weldResultSoup(result, res.Weld())
	relabelTags(result.Tags, rep)

	arr := meshbool.ArrangementTopologyOf(result)
	inputs, ok := reconstructFaces(arr, refs, len(aRefs), op, inputCount, tagCounts(result.Tags),
		groupSize, catalogOriginalEdges(a, b), res)
	if !ok {
		return nil, false
	}
	body := brep.ReconstructBooleanBody(inputs)
	if body == nil || len(body.Faces()) == 0 || !Validate(body).ValidSolid() {
		// Reconstruction can assemble a solid that is closed+manifold yet Euler-inadmissible
		// (an oblique conic exit that crosses an operand CORNER splits its ellipse across two
		// faces, doubling a loop → χ too large). Decline those to the exact faceted fallback;
		// a CLEAN oblique cut (an ellipse bounding one hole per face) validates and rebuilds
		// analytically (ADR-0056 Layer 4).
		return nil, false
	}
	return body, true
}

// reconstructFaces turns each arrangement face into a ReconInput, failing fast when any face
// cannot be rebuilt (a run matching neither an original edge nor an analytic SSI).
func reconstructFaces(arr meshbool.ArrangementTopology, refs []faceSurfaceRef, naRefs int, op meshbool.Op,
	inputCount, resultCount, groupSize map[int]int, cat *origEdgeCatalog, res geom.Resolution) ([]brep.ReconInput, bool) {
	inputs := make([]brep.ReconInput, 0, len(arr.Faces))
	for _, f := range arr.Faces {
		in, ok := reconstructFace(f, refs, naRefs, op, inputCount, resultCount, groupSize, arr.Verts, cat, res)
		if !ok {
			return nil, false
		}
		inputs = append(inputs, in)
	}
	return inputs, true
}

// reconstructFace turns one arrangement face into a ReconInput: a pass-through for an
// untouched survivor, or a rebuilt face for a split one.
func reconstructFace(f meshbool.ArrangementFace, refs []faceSurfaceRef, naRefs int, op meshbool.Op,
	inputCount, resultCount, groupSize map[int]int, verts []meshbool.Point, cat *origEdgeCatalog, res geom.Resolution) (brep.ReconInput, bool) {
	ref := refs[f.Tag]
	fromB := f.Tag >= naRefs
	// Pass a face through only when it is a single original CURVED face kept whole — pass-through
	// exists to preserve a periodic seam loop (a full cylinder wall) that cannot be recovered from
	// facets. A PLANAR face has no such seam and is always rebuilt from its arrangement boundary,
	// because the "kept-whole" signal (result facet count == input count) is not reliable for it:
	// weldResultSoup drops the co-refinement's degenerate slivers, so a face whose overlap WAS cut
	// away can collapse back to its original count (a trimmed cap's minor segment happening to hold
	// as many facets as the whole disc) and would otherwise pass through as the untrimmed face. A
	// MERGED group (groupSize>1) is never one original topo.Face and is likewise always rebuilt.
	_, planar := ref.surface.(geom.Plane)
	if !planar && groupSize[f.Tag] <= 1 && resultCount[f.Tag] == inputCount[f.Tag] {
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

// weldResultSoup collapses result vertices that co-refinement placed within tol of each other to
// a single representative, then drops the degenerate triangles that then appear. The exact mesh
// boolean welds each operand INTERNALLY, but never the two operands to EACH OTHER: where they share
// a boundary sampled by two paths (e.g. a rim reached by a canonical cap edge on one side and a
// cylinder surface evaluation on the other), it inserts one operand's vertex a sub-tolerance step
// from the other's, leaving a near-degenerate sliver that fragments the arrangement. Welding the
// combined output — the same size-relative hygiene the IN adapter applies to each operand — removes
// it. The tolerance lives here in the ops layer; the meshbool core stays exact (ADR-0056).
func weldResultSoup(soup meshbool.TaggedSoup, tol float64) meshbool.TaggedSoup {
	w := newVertexWelder(tol)
	out := meshbool.TaggedSoup{}
	for i := range soup.Tris {
		t := soup.Tris[i]
		tri := [3]meshbool.Point{w.canon(t[0]), w.canon(t[1]), w.canon(t[2])}
		if tri[0].Equal(tri[1]) || tri[1].Equal(tri[2]) || tri[2].Equal(tri[0]) {
			continue // welded to a degenerate sliver
		}
		out.Tris = append(out.Tris, tri)
		out.Tags = append(out.Tags, soup.Tags[i])
	}
	return out
}

// vertexWelder maps points within tol of an earlier one to that earlier representative, via a
// spatial hash keyed on the rounded position (27-cell neighbour search catches a pair straddling
// a cell border).
type vertexWelder struct {
	tol     float64
	buckets map[[3]int64][]weldedVertex
}

type weldedVertex struct {
	rat  meshbool.Point
	flat math.Point3
}

func newVertexWelder(tol float64) *vertexWelder {
	return &vertexWelder{tol: tol, buckets: map[[3]int64][]weldedVertex{}}
}

// canon returns the representative p welds to (an earlier vertex within tol), or records and
// returns p when none exists.
func (w *vertexWelder) canon(p meshbool.Point) meshbool.Point {
	flat := p.Round()
	c := w.cellOf(flat)
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			for dz := int64(-1); dz <= 1; dz++ {
				for _, e := range w.buckets[[3]int64{c[0] + dx, c[1] + dy, c[2] + dz}] {
					if e.flat.DistanceTo(flat) <= w.tol {
						return e.rat
					}
				}
			}
		}
	}
	w.buckets[c] = append(w.buckets[c], weldedVertex{p, flat})
	return p
}

func (w *vertexWelder) cellOf(p math.Point3) [3]int64 {
	return [3]int64{int64(p.X / w.tol), int64(p.Y / w.tol), int64(p.Z / w.tol)}
}
