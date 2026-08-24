// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Coplanar-coincident face handling. When a co-refined face lies in the same plane
// as a face of the other operand and overlaps it, its centroid sits ON that
// surface, so the generalized winding number is ~0.5 — undecidable. Such a face is
// classified instead by whether the two coincident faces' outward normals agree,
// the standard coplanar rule for mesh booleans. Co-refinement (the coplanar-overlap
// imprint, layer 1c-iv-b) has already split the face so it is either wholly
// coincident with one other face or not at all, making the centroid test decisive.

// coplanarPartner reports whether face f is coincident with a face of other — a
// coplanar face whose region contains f's centroid — and, if so, whether their
// outward normals point the same way.
func coplanarPartner(f [3]Point, other [][3]Point) (sameDir, found bool) {
	c := centroid(f)
	for _, g := range other {
		if faceOnPlane(f, g) && inTriangle(g, c) {
			return rdot(triNormal(f), triNormal(g)).Sign() > 0, true
		}
	}
	return false, false
}

// faceOnPlane reports whether every vertex of g lies exactly on the plane of f.
func faceOnPlane(f, g [3]Point) bool {
	return Orient3D(f[0], f[1], f[2], g[0]) == 0 &&
		Orient3D(f[0], f[1], f[2], g[1]) == 0 &&
		Orient3D(f[0], f[1], f[2], g[2]) == 0
}

// inTriangle reports whether p, assumed coplanar with g, lies inside or on triangle
// g (via exact projected edge orientations).
func inTriangle(g [3]Point, p Point) bool {
	axis := planeAxis(g)
	d1 := orient2(g[0], g[1], p, axis)
	d2 := orient2(g[1], g[2], p, axis)
	d3 := orient2(g[2], g[0], p, axis)
	if (d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0) {
		return false // p is strictly outside at least one edge
	}
	return true
}
