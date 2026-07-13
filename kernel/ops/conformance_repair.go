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
	if allBareTriangleFaces(faces) {
		return // a pure faceted triangle soup (e.g. an imported STL body) has no near-collinear
		// boundary point any mesher could drop, so no face can crack — skip the O(edges) free-segment
		// scan and per-face re-mesh that would each be a no-op (#1766).
	}
	w := meshSegWelder(fm...)
	free := freeSegments(fm, w)
	if len(free) == 0 {
		return // watertight body: nothing to repair (and the hot path skips the weld below)
	}
	for j := range facesToFix(faces, idx, fm, free, w) {
		if m := conformingMesh(faces[j], q); m != nil {
			fm[j] = m
		}
	}
}

// conformingMesh re-meshes an absorber face with the appropriate boundary-faithful mesher: a
// cyl/cone via the metric-(u,v) CDT, a plane via the projected-plane CDT. Both keep every boundary
// segment, so the face conforms to its neighbour; nil for anything else (left as meshed).
func conformingMesh(f *topo.Face, q Quality) *Mesh {
	switch f.Geometry().(type) {
	case geom.Cylinder, geom.EllipticalCylinder, geom.Cone:
		if isPeriodicTwoRimBand(f) {
			return nil // a periodic two-rim band: the saddle loft already meshed it exactly, and its
			// full-wrap outer loop degenerates the (u,v) CDT (constrainedDelaunay can spin) — keep the loft
		}
		return conformingCylConeMesh(f, q)
	case geom.Plane:
		return conformingPlaneMesh(f, q)
	}
	return nil
}

// facesToFix is the set of face indices to re-mesh: every cyl/cone/plane face whose mesh touches a
// free (unwelded) segment, plus its cyl/cone/plane topo neighbours across the shared edge — the face
// that ABSORBED the segment (dropped a near-collinear shared-edge point its neighbour kept).
func facesToFix(faces []*topo.Face, idx map[*topo.Face]int, fm []*Mesh, free map[segKey]bool, w segWelder) map[int]bool {
	toFix := map[int]bool{}
	for i, f := range faces {
		if !meshTouchesFree(fm[i], free, w) {
			continue
		}
		if conformable(f.Geometry()) {
			toFix[i] = true
		}
		addConformableNeighbours(f, i, idx, toFix)
	}
	return toFix
}

// addConformableNeighbours marks face i's cyl/cone/plane neighbours (across a shared edge) for
// re-meshing — the plane neighbour is the absorber when a planar cap drops a curved rim point.
func addConformableNeighbours(f *topo.Face, i int, idx map[*topo.Face]int, toFix map[int]bool) {
	for _, e := range f.Edges() {
		for _, nf := range e.Faces() {
			if j, ok := idx[nf]; ok && j != i && conformable(nf.Geometry()) {
				toFix[j] = true
			}
		}
	}
}

// conformable reports whether a face's surface has a boundary-faithful conforming re-mesher
// (conformingMesh): cylinders, cones, and planes.
func conformable(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.EllipticalCylinder, geom.Cone, geom.Plane:
		return true
	}
	return false
}

// allBareTriangleFaces reports whether every face is a straight-edged planar triangle with no hole
// loop — a pure faceted triangle soup (e.g. an imported STL body). Such a body has no near-collinear
// boundary point any mesher could drop, so the conformance-repair pass can never change it and the
// caller skips it. Topology + curve/surface kind only, no discretization (#1766).
func allBareTriangleFaces(faces []*topo.Face) bool {
	if len(faces) == 0 {
		return false
	}
	for _, f := range faces {
		if !isBareTriangleFace(f) {
			return false
		}
	}
	return true
}

// isBareTriangleFace reports whether f is a planar face bounded by exactly three straight (line) edges
// and no hole loops, so its discretized boundary is exactly its three corners: a straight edge with no
// healing snap samples to two points (subdivide stops immediately on a zero-curvature run), and three
// such edges close to three points. A snapped edge is excluded — it can discretize to a polyline.
func isBareTriangleFace(f *topo.Face) bool {
	if _, planar := f.Geometry().(geom.Plane); !planar {
		return false
	}
	loops := f.Loops()
	if len(loops) != 1 || !loops[0].IsOuter() {
		return false
	}
	uses := loops[0].EdgeUses()
	if len(uses) != 3 {
		return false
	}
	for _, u := range uses {
		e := u.Edge()
		if e.SnappedCurve() != nil {
			return false
		}
		switch e.Geometry().(type) {
		case geom.Line, geom.LineSegment: // straight: samples to its two endpoints
		default:
			return false
		}
	}
	return true
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

// conformingPlaneMesh re-meshes a planar absorber with the projected-plane constrained Delaunay,
// which keeps EVERY boundary segment (so the face reproduces a near-collinear shared-edge point a
// curved neighbour kept, instead of the area-gated planarTris dropping it — the plane-absorber crack
// on screw caps, #1073). It feeds constrainedDelaunay the SAME projected coordinates the plane's
// neighbours discretize from, so it conforms rather than cascading; and because every one of the
// plane's OWN boundary segments stays a constraint, re-meshing it cannot crack its other neighbours.
// nil when the trim has overlapping holes (its own union mesher conforms). The former
// 256-vertex CDT budget is retired with planarTris' (#1610): #1409's corridor-walk
// segment insertion removed the quadratic constraint recovery the budget guarded against.
func conformingPlaneMesh(f *topo.Face, q Quality) *Mesh {
	normal := f.Geometry().NormalAt(0, 0)
	flat := planeProjector(normal)
	outer3D := faceOuterBoundary(f, q)
	if len(outer3D) < 3 {
		return nil
	}
	holes3D := faceHoleBoundaries(f, q)
	if len(outer3D) == 3 && len(holes3D) == 0 {
		return nil // an already-triangular boundary: planarCDT of three points reproduces the same
		// single triangle the initial mesher built, so re-meshing is a no-op — keep it (#1766).
	}
	outer2D := project2D(outer3D, flat)
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		holes2D[i] = project2D(h, flat)
	}
	if holesOverlap(holes2D) {
		return nil
	}
	if !simpleLoop2D(outer2D) {
		return nil // a self-intersecting boundary: the boundary-faithful CDT collapses on it (it
		// once shrank a correct 8475 face to 675 — issue: fillet builds a self-intersecting loop
		// when a feature protrudes into the removed strip). The robust earclip initial mesh has the
		// right area; conformance must never REPLACE a good mesh with a collapsed one, so keep it.
	}
	tris := planarCDT(outer2D, holes2D)
	if len(tris) == 0 {
		return nil
	}
	return planarMeshFromTris(outer3D, holes3D, tris, normal)
}

// simpleLoop2D reports whether the closed polygon pts has no two non-adjacent edges properly
// crossing — i.e. it is a simple polygon the boundary-faithful CDT can triangulate. Adjacent edges
// (sharing a vertex) are skipped; segmentsCross uses a strict sign test so shared endpoints and
// collinear touches do not count. O(n^2), fine for a face boundary's vertex count.
func simpleLoop2D(pts []math.Point2) bool {
	n := len(pts)
	if n < 4 {
		return true
	}
	for i := 0; i < n; i++ {
		a, b := xy(pts[i]), xy(pts[(i+1)%n])
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // edges n-1→0 and 0→1 are adjacent (share vertex 0)
			}
			if segmentsCross(a, b, xy(pts[j]), xy(pts[(j+1)%n])) {
				return false
			}
		}
	}
	return true
}

// planarMeshFromTris builds a planar face mesh: the outer-then-holes 3D vertex buffer (the order
// planarTris/planarCDT index into) carrying the face normal, triangulated by tris.
func planarMeshFromTris(outer3D []math.Point3, holes3D [][]math.Point3, tris [][3]int, normal math.Vector3) *Mesh {
	m := &Mesh{}
	for _, p := range outer3D {
		m.addVertex(p, normal)
	}
	for _, h := range holes3D {
		for _, p := range h {
			m.addVertex(p, normal)
		}
	}
	for _, t := range tris {
		m.addTriangle(t[0], t[1], t[2])
	}
	return m
}

// segKey is the collision-free, order-independent key of a welded segment: the quantized (1 µm grid,
// matching freeEdgeCount) coordinates of both endpoints, the lexicographically smaller one first.
type segKey [6]int64

// freeSegments returns the welded segments that exactly ONE triangle uses across all face meshes — the
// cross-face cracks (an interior manifold edge is used by two).
func freeSegments(fm []*Mesh, w segWelder) map[segKey]bool {
	deg := map[segKey]int{}
	for _, m := range fm {
		for t := 0; t+2 < len(m.Indices); t += 3 {
			for k := 0; k < 3; k++ {
				deg[w.seg(m.Positions[m.Indices[t+k]], m.Positions[m.Indices[t+(k+1)%3]])]++
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
func meshTouchesFree(m *Mesh, free map[segKey]bool, w segWelder) bool {
	for t := 0; t+2 < len(m.Indices); t += 3 {
		for k := 0; k < 3; k++ {
			if free[w.seg(m.Positions[m.Indices[t+k]], m.Positions[m.Indices[t+(k+1)%3]])] {
				return true
			}
		}
	}
	return false
}

// segWelder quantizes mesh positions onto a model-relative grid (Resolution.Weld) for
// edge pairing. Shared boundary points from adjacent faces are BIT-IDENTICAL (shared
// edge discretization), so any grid pairs them; the grid's only job is to not falsely
// merge distinct fine-feature vertices — which the old fixed 1e-6 grid did on µm-scale
// parts, silently masking cracks (#1610, ADR-0042).
type segWelder struct{ grid float64 }

// meshSegWelder derives the weld grid from the meshes' combined bounding box.
func meshSegWelder(fm ...*Mesh) segWelder {
	box := math.EmptyBox()
	for _, m := range fm {
		for _, p := range m.Positions {
			box = box.ExtendPoint(p)
		}
	}
	return segWelder{grid: geom.ResolutionForBox(box).Weld()}
}

func (w segWelder) seg(a, b math.Point3) segKey {
	ka, kb := w.coord(a), w.coord(b)
	if ka[0] > kb[0] || (ka[0] == kb[0] && (ka[1] > kb[1] || (ka[1] == kb[1] && ka[2] > kb[2]))) {
		ka, kb = kb, ka
	}
	return segKey{ka[0], ka[1], ka[2], kb[0], kb[1], kb[2]}
}

func (w segWelder) coord(p math.Point3) [3]int64 {
	return [3]int64{
		int64(float64(p.X) / w.grid),
		int64(float64(p.Y) / w.grid),
		int64(float64(p.Z) / w.grid),
	}
}

// less reports whether a sorts before b in quantized coordinates (the canonical edge direction).
func (w segWelder) less(a, b math.Point3) bool {
	ka, kb := w.coord(a), w.coord(b)
	for i := 0; i < 3; i++ {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return false
}
