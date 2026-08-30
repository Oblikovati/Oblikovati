// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Self-intersection detection (M07 PBI-084, Oblikovati/Oblikovati#300): pairs
// of faces of one body that pass through each other. Topologically adjacent
// faces (sharing an edge or vertex) legitimately touch along that boundary and
// are excluded; what remains crossing is real interpenetration — the classic
// outcome of a bad import or an over-folded offset.
//
// This file is the BROAD PHASE and reporting: it tessellates each face, prunes
// pairs by bounding box + a per-face BVH, excludes shared topology, and reports
// a witness point. The per-triangle Möller narrow phase lives in
// self_intersect_triangle.go; its coplanar branch in self_intersect_coplanar.go.

// SelfIntersection is one interpenetrating, non-adjacent face pair, with a
// witness point on the crossing.
type SelfIntersection struct {
	FaceA, FaceB *topo.Face
	Witness      math.Point3
}

// SelfIntersections tessellates each face at q and reports every non-adjacent
// face pair whose triangles cross. The check is mesh-accurate: a crossing no
// deeper than the two faces' own faceting error is not reported, because the
// mesh carries no evidence that the true surfaces cross at all — which matches
// what every downstream consumer (booleans, mass properties, export) sees anyway.
// Two planar faces are meshed exactly and so are held to a crossing of zero.
//
// Example: if hits := ops.SelfIntersections(body, ops.DefaultQuality()); len(hits) > 0 { reject(body) }
func SelfIntersections(b *topo.Body, q Quality) []SelfIntersection {
	faces := b.Faces()
	tris := make([][][3]math.Point3, len(faces))
	boxes := make([]math.Box, len(faces))
	bvhs := make([]*triBVH, len(faces)) // built lazily the first time a face is the queried-against side
	for i, f := range faces {
		tris[i] = meshTriangles(TessellateFace(f, q))
		boxes[i] = f.RangeBox()
	}
	var out []SelfIntersection
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			if !boxes[i].Intersects(boxes[j]) {
				continue
			}
			// Don't skip the whole pair when the faces merely touch (#1321): test triangle pairs and
			// discard only crossings that land ON the shared boundary (the legitimate edge/vertex
			// contact). A crossing AWAY from the shared topology is a real interpenetration.
			shared := sharedFaceBoundary(faces[i], faces[j])
			if bvhs[j] == nil {
				bvhs[j] = newTriBVH(tris[j])
			}
			// Two meshes may each stray from their true surface, so they can appear to cross by up
			// to the SUM of their deviations with no real interpenetration behind it — which is what
			// made every tangent blend (fillet/sphere/cone weld) report a false self-intersection
			// (#2077). Planar faces contribute nothing, so plane-on-plane stays exact.
			allow := faceMeshDeviation(faces[i], q) + faceMeshDeviation(faces[j], q)
			if p, hit := meshCrossesOffBoundary(tris[i], bvhs[j], shared, q.tol(), allow); hit {
				out = append(out, SelfIntersection{FaceA: faces[i], FaceB: faces[j], Witness: p})
			}
		}
	}
	return out
}

// sharedFaceBoundary returns the geometry two faces legitimately share: their common edges as
// segments and any common vertex as a degenerate (point) segment. Contact within tol of any of these
// is the faces meeting along their shared topology, not an interpenetration.
func sharedFaceBoundary(a, b *topo.Face) [][2]math.Point3 {
	edgesB := map[*topo.Edge]bool{}
	vertsB := map[*topo.Vertex]bool{}
	for _, e := range b.Edges() {
		edgesB[e] = true
		vertsB[e.StartVertex()], vertsB[e.EndVertex()] = true, true
	}
	var shared [][2]math.Point3
	seenV := map[*topo.Vertex]bool{}
	for _, e := range a.Edges() {
		if edgesB[e] {
			shared = append(shared, [2]math.Point3{e.StartVertex().Point(), e.EndVertex().Point()})
		}
		for _, v := range [2]*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if vertsB[v] && !seenV[v] {
				seenV[v] = true
				shared = append(shared, [2]math.Point3{v.Point(), v.Point()})
			}
		}
	}
	return shared
}

// meshCrossesOffBoundary tests each triangle of mesh A against only the B triangles its box overlaps
// (found through the B-side BVH, not an all-pairs scan, #1411) and returns the first crossing whose
// witness lies farther than tol from the shared boundary — the first real interpenetration. The exact
// Möller test and the boundary filter are unchanged, so detection is identical to the old scan; only
// the candidates it runs on are pruned. Crossings on the shared boundary are legitimate contact.
func meshCrossesOffBoundary(aTris [][3]math.Point3, bBVH *triBVH, shared [][2]math.Point3, tol, allow float64) (math.Point3, bool) {
	var witness math.Point3
	found := false
	for _, t1 := range aTris {
		box := math.BoxFromPoints(t1[0], t1[1], t1[2])
		bBVH.query(box, func(j int) bool {
			p, kind := triangleCrossing(t1, bBVH.tris[j], allow)
			if kind == crossNone || onSharedBoundary(p, shared, tol) {
				return false
			}
			witness, found = p, true
			return true // stop the BVH walk at the first real crossing
		})
		if found {
			return witness, true
		}
	}
	return math.Point3{}, false
}

// faceMeshDeviation is how far a face's TESSELLATION may stray from its true surface. A planar
// face is meshed exactly, so it gets no allowance at all; a curved face's chords sit up to the
// chord tolerance away from the surface.
func faceMeshDeviation(f *topo.Face, q Quality) float64 {
	if _, planar := f.Geometry().(geom.Plane); planar {
		return 0
	}
	return q.tol()
}

// onSharedBoundary reports whether p lies within tol of any shared boundary segment/point.
func onSharedBoundary(p math.Point3, shared [][2]math.Point3, tol float64) bool {
	for _, s := range shared {
		if pointToSegment(p, s[0], s[1]) <= tol {
			return true
		}
	}
	return false
}

func meshTriangle(m *Mesh, i int) [3]math.Point3 {
	return [3]math.Point3{
		m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]],
	}
}
