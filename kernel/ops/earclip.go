// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/predicates"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// orient2p is the sign of the signed area of triangle (a, b, c) via the one exact predicate package
// (kernel/predicates, ADR-0042): >0 CCW, <0 CW, 0 collinear. It adapts math.Point2 to the flat-coord
// entry point so ear-clipping's convexity/containment tests stay sign-exact on near-collinear samples.
func orient2p(a, b, c math.Point2) int {
	return predicates.Orient2D(a.X, a.Y, b.X, b.Y, c.X, c.Y)
}

// tessellatePlanarFace triangulates a planar face's boundary by ear-clipping (using
// the exact orientation predicate), giving a watertight triangulation. The boundary
// comes from [loopBoundary], so a planar face bordered by a curved edge (e.g. a fillet's
// flat end cap) follows the arc — sampled identically to the curved neighbour, no
// T-junctions — rather than chording across it.
func tessellatePlanarFace(f *topo.Face, q Quality) *Mesh {
	normal := f.Geometry().NormalAt(0, 0)
	flat := planeProjector(normal)
	boundary3D := faceOuterBoundary(f, q)
	boundary2D := project2D(boundary3D, flat)
	if holes3D := faceHoleBoundaries(f, q); len(holes3D) > 0 {
		return holedPlanarMesh(boundary2D, boundary3D, holes3D, flat, normal)
	}
	m := &Mesh{}
	for _, p := range boundary3D {
		m.addVertex(p, normal)
	}
	for _, tri := range planarTris(boundary2D, nil) {
		m.addTriangle(tri[0], tri[1], tri[2])
	}
	return m
}

// holedPlanarMesh triangulates a planar face that has hole loops via earcut, whose indices
// address the outer boundary followed by the holes concatenated — so the 3D vertex buffer is
// built in that same order.
func holedPlanarMesh(outer2D []math.Point2, outer3D []math.Point3, holes3D [][]math.Point3, flat func(math.Point3) math.Point2, normal math.Vector3) *Mesh {
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		holes2D[i] = project2D(h, flat)
	}
	if holesOverlap(holes2D) {
		// Overlapping inner loops are degenerate for direct earcut; union them via the planar
		// arrangement so the mesh covers exactly outer − union(holes) (#873).
		return unionHoledMesh(outer3D, holes3D, normal)
	}
	m := &Mesh{}
	for _, p := range outer3D {
		m.addVertex(p, normal)
	}
	for _, h := range holes3D {
		for _, p := range h {
			m.addVertex(p, normal)
		}
	}
	for _, tri := range planarTris(outer2D, holes2D) {
		m.addTriangle(tri[0], tri[1], tri[2])
	}
	return m
}

// project2D drops each point to the face plane via the projector.
func project2D(pts []math.Point3, flat func(math.Point3) math.Point2) []math.Point2 {
	out := make([]math.Point2, len(pts))
	for i, p := range pts {
		out[i] = flat(p)
	}
	return out
}

// faceOuterBoundary returns the discretized point ring of a face's outer loop.
func faceOuterBoundary(f *topo.Face, q Quality) []math.Point3 {
	for _, l := range f.Loops() {
		if l.IsOuter() {
			return loopBoundary(l, q)
		}
	}
	return nil
}

// faceHoleBoundaries returns the discretized point ring of each inner (hole) loop.
func faceHoleBoundaries(f *topo.Face, q Quality) [][]math.Point3 {
	var holes [][]math.Point3
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			holes = append(holes, loopBoundary(l, q))
		}
	}
	return holes
}

// earClip triangulates a simple polygon, returning triangles as index triples into
// poly. It orients the polygon CCW first, then repeatedly clips convex ears.
func earClip(poly []math.Point2) [][3]int {
	n := len(poly)
	if n < 3 {
		return nil
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	if signedArea(poly) < 0 {
		reverse(idx)
	}
	var tris [][3]int
	for len(idx) > 3 {
		i, ok := findEar(poly, idx)
		if !ok {
			break // degenerate / non-simple polygon: stop with what we have
		}
		m := len(idx)
		tris = append(tris, [3]int{idx[(i+m-1)%m], idx[i], idx[(i+1)%m]})
		idx = append(idx[:i], idx[i+1:]...)
	}
	if len(idx) == 3 {
		tris = append(tris, [3]int{idx[0], idx[1], idx[2]})
	}
	return tris
}

// findEar returns the position in idx of a clippable ear (convex vertex whose
// triangle contains no other vertex).
func findEar(poly []math.Point2, idx []int) (int, bool) {
	m := len(idx)
	for i := range m {
		a, b, c := poly[idx[(i+m-1)%m]], poly[idx[i]], poly[idx[(i+1)%m]]
		if orient2p(a, b, c) <= 0 {
			continue // reflex or collinear vertex — not an ear
		}
		if !anyInside(poly, idx, i, a, b, c) {
			return i, true
		}
	}
	return 0, false
}

// anyInside reports whether any reflex vertex (other than the ear's three) lies strictly
// inside abc. Only reflex vertices can block an ear, and coincident bridge vertices (equal
// to a corner) are skipped — both are required so hole-bridged polygons, which carry doubled
// bridge vertices, still triangulate instead of stalling.
func anyInside(poly []math.Point2, idx []int, ear int, a, b, c math.Point2) bool {
	m := len(idx)
	for k := range m {
		if k == ear || k == (ear+m-1)%m || k == (ear+1)%m {
			continue
		}
		p := poly[idx[k]]
		if p == a || p == b || p == c {
			continue // a doubled bridge vertex coincident with an ear corner — not blocking
		}
		if orient2p(poly[idx[(k+m-1)%m]], p, poly[idx[(k+1)%m]]) > 0 {
			continue // convex vertex — cannot block an ear
		}
		if pointInTriangle(p, a, b, c) {
			return true
		}
	}
	return false
}

// pointInTriangle reports whether p is strictly inside CCW triangle abc.
func pointInTriangle(p, a, b, c math.Point2) bool {
	return orient2p(a, b, p) > 0 &&
		orient2p(b, c, p) > 0 &&
		orient2p(c, a, p) > 0
}

// signedArea returns the shoelace signed area (positive = CCW).
func signedArea(poly []math.Point2) float64 {
	var s float64
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		s += p.X*q.Y - q.X*p.Y
	}
	return s / 2
}

func reverse(idx []int) {
	for i, j := 0, len(idx)-1; i < j; i, j = i+1, j-1 {
		idx[i], idx[j] = idx[j], idx[i]
	}
}

func bestSingleLoopTriangulation(poly []math.Point2) [][3]int {
	clip := earClip(poly)
	cut := earcut(poly, nil)
	want := stdmath.Abs(signedArea(poly))
	if areaError(poly, cut, want) < areaError(poly, clip, want) {
		return cut
	}
	return clip
}

func areaError(poly []math.Point2, tris [][3]int, want float64) float64 {
	return stdmath.Abs(triangleArea2D(poly, tris) - want)
}

func triangleArea2D(poly []math.Point2, tris [][3]int) float64 {
	var area float64
	for _, tri := range tris {
		a, b, c := poly[tri[0]], poly[tri[1]], poly[tri[2]]
		// Twice the signed area is the cross product (a−c)×(b−c); this needs the determinant
		// MAGNITUDE, not an orientation sign, so it computes it directly rather than through the
		// exact predicate (whose exact path returns ±1, not the area) — #2287.
		det := (a.X-c.X)*(b.Y-c.Y) - (a.Y-c.Y)*(b.X-c.X)
		area += stdmath.Abs(det) / 2
	}
	return area
}

// planeProjector returns a 2D projection that drops the normal's dominant axis,
// preserving in-plane geometry for triangulation. The two retained axes are ordered so
// their cross product equals the normal's sign — i.e. a counter-clockwise loop in the
// projection corresponds to the face's outward normal. This keeps the triangulation
// wound consistently outward (every emitted triangle's geometric normal matches the
// face normal), which the renderer ignores (it uses per-vertex normals) but boolean CSG
// and divergence-theorem volume depend on. Without the sign-aware axis order, faces with
// a negative-axis normal come out inward and the triangle soup is not a coherent solid.
func planeProjector(n math.Vector3) func(math.Point3) math.Point2 {
	ax, ay, az := stdmath.Abs(n.X), stdmath.Abs(n.Y), stdmath.Abs(n.Z)
	switch {
	case ax >= ay && ax >= az: // drop X: (Y,Z) for +X, (Z,Y) for −X
		if n.X >= 0 {
			return func(p math.Point3) math.Point2 { return math.P2(p.Y, p.Z) }
		}
		return func(p math.Point3) math.Point2 { return math.P2(p.Z, p.Y) }
	case ay >= ax && ay >= az: // drop Y: (Z,X) for +Y, (X,Z) for −Y
		if n.Y >= 0 {
			return func(p math.Point3) math.Point2 { return math.P2(p.Z, p.X) }
		}
		return func(p math.Point3) math.Point2 { return math.P2(p.X, p.Z) }
	default: // drop Z: (X,Y) for +Z, (Y,X) for −Z
		if n.Z >= 0 {
			return func(p math.Point3) math.Point2 { return math.P2(p.X, p.Y) }
		}
		return func(p math.Point3) math.Point2 { return math.P2(p.Y, p.X) }
	}
}
