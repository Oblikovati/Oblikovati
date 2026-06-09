// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cross-face conformance repair (M25 PBI-330). A trimmed cylinder/cone wall meshed by the best-fit-plane
// ear-clip (boundaryPatchMesh) can ABSORB a near-collinear point of a rim arc — a cyl/cone rim is an
// iso-curve, nearly straight once projected — so it emits one fewer segment than its planar neighbour,
// which keeps the arc curved and emits all of them. The two faces then disagree on the shared edge and
// leave an unpaired (free) edge: a visible crack. This pass detects those free edges and re-meshes ONLY
// the cyl/cone faces touching them (and their cyl/cone neighbours) with a BOUNDARY-ONLY constrained
// Delaunay in metric (u,v), which keeps every boundary segment as a constraint (so it conforms) and
// never folds (metric (u,v) is developable). A watertight body has no free edges, so it is left
// completely untouched — no regression on bodies the ear-clip already meshes correctly. It deliberately
// does NOT re-mesh planes: re-triangulating a planar multi-hole face cascades new mismatches, so a crack
// where the PLANE is the absorber is left to a future pass.
func conformCylConeFaces(faces []*topo.Face, idx map[*topo.Face]int, fm []*Mesh, q Quality) {
	free := freeSegments(fm)
	if len(free) == 0 {
		return // watertight body: nothing to repair (and the hot path skips the weld below)
	}
	toFix := map[int]bool{}
	for i, f := range faces {
		if !meshTouchesFree(fm[i], free) {
			continue
		}
		if isCylOrCone(f.Geometry()) {
			toFix[i] = true
		}
		// The face MISSING the segment is the topo neighbour across the shared edge; re-mesh it too.
		for _, e := range f.Edges() {
			for _, nf := range e.Faces() {
				if j, ok := idx[nf]; ok && j != i && isCylOrCone(nf.Geometry()) {
					toFix[j] = true
				}
			}
		}
	}
	for j := range toFix {
		if m := conformingCylConeMesh(faces[j], q); m != nil {
			fm[j] = m
		}
	}
}

func isCylOrCone(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.Cone:
		return true
	}
	return false
}

// conformingCylConeMesh re-meshes a non-rectangular cyl/cone trim with a boundary-only metric-(u,v)
// constrained Delaunay, the conforming alternative to the plane ear-clip. nil if not applicable (the
// trim is a full periodic band / apex cap / iso-rectangle handled watertight by other meshers, or its
// (u,v) degenerates).
func conformingCylConeMesh(f *topo.Face, q Quality) *Mesh {
	s := f.Geometry()
	outer3D := faceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if len(outer3D) < 3 {
		return nil
	}
	outerUV, holesUV, ok := toUVLoops(s, outer3D, holes3D)
	if !ok {
		return nil
	}
	if _, _, isRect := isoRectangleGrid(outerUV); isRect && len(holesUV) == 0 {
		return nil // an iso-rectangle is already watertight via structuredGridMesh
	}
	su, sv := metricScale(s)
	b := newPatchBuilder(s, su, sv)
	loops := [][]int{b.addLoop(outer3D, outerUV)}
	for i := range holes3D {
		loops = append(loops, b.addLoop(holes3D[i], holesUV[i]))
	}
	tris := constrainedDelaunay(b.scaled, loops)
	if len(tris) == 0 {
		return nil
	}
	return patchMeshFrom(b.pos, b.nrm, tris)
}

// segKey is the collision-free, order-independent key of a welded segment: the quantized (1 µm grid,
// matching freeEdgeCount) coordinates of both endpoints, the lexicographically smaller one first.
type segKey [6]int64

// freeSegments returns the welded segments that exactly ONE triangle uses across all face meshes — the
// cross-face cracks (an interior manifold edge is used by two).
func freeSegments(fm []*Mesh) map[segKey]bool {
	deg := map[segKey]int{}
	for _, m := range fm {
		for t := 0; t+2 < len(m.Indices); t += 3 {
			for k := 0; k < 3; k++ {
				deg[weldSeg(m.Positions[m.Indices[t+k]], m.Positions[m.Indices[t+(k+1)%3]])]++
			}
		}
	}
	free := map[segKey]bool{}
	for e, n := range deg {
		if n == 1 {
			free[e] = true
		}
	}
	return free
}

// meshTouchesFree reports whether any edge of m is one of the free (unpaired) segments.
func meshTouchesFree(m *Mesh, free map[segKey]bool) bool {
	for t := 0; t+2 < len(m.Indices); t += 3 {
		for k := 0; k < 3; k++ {
			if free[weldSeg(m.Positions[m.Indices[t+k]], m.Positions[m.Indices[t+(k+1)%3]])] {
				return true
			}
		}
	}
	return false
}

func weldSeg(a, b math.Point3) segKey {
	ka, kb := quantCoord(a), quantCoord(b)
	if ka[0] > kb[0] || (ka[0] == kb[0] && (ka[1] > kb[1] || (ka[1] == kb[1] && ka[2] > kb[2]))) {
		ka, kb = kb, ka
	}
	return segKey{ka[0], ka[1], ka[2], kb[0], kb[1], kb[2]}
}

func quantCoord(p math.Point3) [3]int64 {
	v := math.Point3{}.VectorTo(p)
	return [3]int64{
		int64(float64(v.Dot(math.Vector3{X: 1})) * 1e6),
		int64(float64(v.Dot(math.Vector3{Y: 1})) * 1e6),
		int64(float64(v.Dot(math.Vector3{Z: 1})) * 1e6),
	}
}
