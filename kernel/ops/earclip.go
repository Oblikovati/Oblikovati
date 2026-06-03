// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/math/predicate"
)

// tessellatePlanarFace triangulates a planar face's outer boundary by ear-clipping
// (using the exact orientation predicate), giving a watertight triangulation whose
// vertices are exactly the boundary vertices — chordal tolerance is met trivially
// since the facets lie in the face plane.
func tessellatePlanarFace(f *topo.Face) *Mesh {
	boundary := faceOuterBoundary(f)
	normal := f.Geometry().NormalAt(0, 0)
	flat := planeProjector(normal)
	poly := make([]math.Point2, len(boundary))
	for i, p := range boundary {
		poly[i] = flat(p)
	}
	m := &Mesh{}
	for _, p := range boundary {
		m.addVertex(p, normal)
	}
	for _, tri := range earClip(poly) {
		m.addTriangle(tri[0], tri[1], tri[2])
	}
	return m
}

// faceOuterBoundary returns the ordered vertices of a face's outer loop.
func faceOuterBoundary(f *topo.Face) []math.Point3 {
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		pts := make([]math.Point3, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			v := u.Edge().StartVertex()
			if u.Reversed() {
				v = u.Edge().EndVertex()
			}
			pts = append(pts, v.Point())
		}
		return pts
	}
	return nil
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
	for i := 0; i < m; i++ {
		a, b, c := poly[idx[(i+m-1)%m]], poly[idx[i]], poly[idx[(i+1)%m]]
		if predicate.Orient2D(a, b, c) <= 0 {
			continue // reflex or collinear vertex — not an ear
		}
		if !anyInside(poly, idx, i, a, b, c) {
			return i, true
		}
	}
	return 0, false
}

// anyInside reports whether any vertex other than the ear's three lies inside abc.
func anyInside(poly []math.Point2, idx []int, ear int, a, b, c math.Point2) bool {
	m := len(idx)
	for k := 0; k < m; k++ {
		if k == ear || k == (ear+m-1)%m || k == (ear+1)%m {
			continue
		}
		if pointInTriangle(poly[idx[k]], a, b, c) {
			return true
		}
	}
	return false
}

// pointInTriangle reports whether p is inside CCW triangle abc (boundary counts as
// inside to reject degenerate ears).
func pointInTriangle(p, a, b, c math.Point2) bool {
	return predicate.Orient2D(a, b, p) >= 0 &&
		predicate.Orient2D(b, c, p) >= 0 &&
		predicate.Orient2D(c, a, p) >= 0
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
