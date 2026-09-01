// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/predicates"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// orient3p is the sign of the signed volume of tetrahedron (a, b, c, d) via the one exact predicate
// package (kernel/predicates, ADR-0042): it adapts math.Point3 to the flat-coord entry point so the
// hull's face-visibility and seed-independence tests stay sign-exact on coplanar/cocircular inputs.
func orient3p(a, b, c, d math.Point3) int {
	return predicates.Orient3D(a.X, a.Y, a.Z, b.X, b.Y, b.Z, c.X, c.Y, c.Z, d.X, d.Y, d.Z)
}

// ConvexHull returns the convex hull of a point set as a closed, triangulated solid body.
// It is the kernel behind OpenSCAD's hull() — the smallest convex solid containing the input
// (e.g. the hull of two cylinders is a capsule). It errors if the points are degenerate
// (fewer than four, or all collinear/coplanar so no 3D hull exists).
//
//	body, _ := ConvexHull([]math.Point3{ /* cube corners */ }, "hull") // → the cube
func ConvexHull(points []math.Point3, feat string) (*topo.Body, error) {
	verts, faces, err := convexHull3D(points)
	if err != nil {
		return nil, err
	}
	b := mesh.CageToBody(verts, faces, feat)
	if b == nil || !b.IsSolid() {
		return nil, errHull("hull assembly did not close into a solid")
	}
	return b, nil
}

// ConvexHullOf returns the convex hull of every input body's vertices — the OpenSCAD
// hull(a, b, …) of a set of solids. Convex inputs are reproduced exactly (up to their own
// faceting); concave inputs are filled to their hull.
func ConvexHullOf(feat string, bodies ...*topo.Body) (*topo.Body, error) {
	var pts []math.Point3
	for _, b := range bodies {
		if b == nil {
			continue
		}
		for _, v := range b.Vertices() {
			pts = append(pts, v.Point())
		}
	}
	return ConvexHull(pts, feat)
}

func errHull(msg string) error { return &hullError{msg} }

type hullError struct{ msg string }

func (e *hullError) Error() string { return "convex hull: " + e.msg }

// convexHull3D computes the hull by incremental insertion (the quickhull horizon method): seed
// a tetrahedron, then for each point fold the faces it can see into a fan to the point. Faces
// are wound so each triangle's (b−a)×(c−a) points outward, which is what cageToBody needs.
func convexHull3D(points []math.Point3) ([]math.Point3, [][3]int, error) {
	// Fold onto the shared model-relative resolution (ADR-0042): the dedup weld grid
	// and the visibility tolerance both scale with the point set's size. (The visibility
	// tolerance was already 1e-9 × bounds-diagonal — i.e. exactly Weld() — the precedent
	// this whole scheme generalises.)
	res := tol.ForPoints(points)
	pts := dedupPoints(points, res.Weld())
	if len(pts) < 4 {
		return nil, nil, errHull("need at least 4 distinct points, got " + itoa(len(pts)))
	}
	tet, ok := initialTetra(pts)
	if !ok {
		return nil, nil, errHull("points are collinear or coplanar (no 3D hull)")
	}
	faces := initialFaces(pts, tet)
	for i := range pts {
		if i == tet[0] || i == tet[1] || i == tet[2] || i == tet[3] {
			continue
		}
		faces = insertPoint(pts, faces, i)
	}
	return compactHull(pts, faces)
}

// insertPoint folds the faces visible from pts[pi] into a triangle fan anchored at pi, leaving
// the hull closed. A point that sees no face is interior and changes nothing.
func insertPoint(pts []math.Point3, faces [][3]int, pi int) [][3]int {
	p := pts[pi]
	visible := make([]bool, len(faces))
	anyVisible := false
	for fi, f := range faces {
		if faceVisible(pts, f, p) {
			visible[fi], anyVisible = true, true
		}
	}
	if !anyVisible {
		return faces
	}
	seen := visibleEdgeSet(faces, visible)
	out := make([][3]int, 0, len(faces))
	for fi, f := range faces {
		if !visible[fi] {
			out = append(out, f)
		}
	}
	// A directed edge a→b of a visible face is on the horizon when its twin b→a is not also a
	// visible-face edge (i.e. the face across it survives). Fan the new point to each.
	for fi, f := range faces {
		if !visible[fi] {
			continue
		}
		for e := range 3 {
			a, b := f[e], f[(e+1)%3]
			if !seen[[2]int{b, a}] {
				out = append(out, [3]int{a, b, pi})
			}
		}
	}
	return out
}

// visibleEdgeSet collects every directed edge of the visible faces.
func visibleEdgeSet(faces [][3]int, visible []bool) map[[2]int]bool {
	seen := map[[2]int]bool{}
	for fi, f := range faces {
		if !visible[fi] {
			continue
		}
		for e := range 3 {
			seen[[2]int{f[e], f[(e+1)%3]}] = true
		}
	}
	return seen
}

// faceVisible reports whether p lies strictly outside the face (on its outward side) — i.e. the face
// must fold to admit p. The test is the EXACT orientation predicate (#1323 L2): with the face wound so
// (b−a)×(c−a) points outward, p sees it iff Orient3D(a,b,c,p) < 0 (p on the +normal side). A float
// dot-product tolerance was fragile on the coplanar/cocircular inputs hull() is built for (boxes,
// cylinders), producing inverted or extra sliver faces; predicate.Orient3D is sign-exact there.
func faceVisible(pts []math.Point3, f [3]int, p math.Point3) bool {
	return orient3p(pts[f[0]], pts[f[1]], pts[f[2]], p) < 0
}

// initialFaces builds the seed tetrahedron's four faces, each wound so its normal points away
// from the opposite vertex (outward).
func initialFaces(pts []math.Point3, tet [4]int) [][3]int {
	i0, i1, i2, i3 := tet[0], tet[1], tet[2], tet[3]
	return [][3]int{
		outwardTri(pts, i0, i1, i2, i3),
		outwardTri(pts, i0, i1, i3, i2),
		outwardTri(pts, i0, i2, i3, i1),
		outwardTri(pts, i1, i2, i3, i0),
	}
}

// outwardTri orders (a,b,c) so its normal points away from the interior vertex `inside`.
func outwardTri(pts []math.Point3, a, b, c, inside int) [3]int {
	n := pts[a].VectorTo(pts[b]).Cross(pts[a].VectorTo(pts[c]))
	if pts[a].VectorTo(pts[inside]).Dot(n) > 0 { // interior point on the +normal side → flip
		return [3]int{a, c, b}
	}
	return [3]int{a, b, c}
}

// compactHull renumbers the faces' vertices to the subset actually used by the hull.
func compactHull(pts []math.Point3, faces [][3]int) ([]math.Point3, [][3]int, error) {
	if len(faces) < 4 {
		return nil, nil, errHull("degenerate result (fewer than 4 faces)")
	}
	remap := map[int]int{}
	var verts []math.Point3
	out := make([][3]int, len(faces))
	for i, f := range faces {
		var nf [3]int
		for k := range 3 {
			idx, ok := remap[f[k]]
			if !ok {
				idx = len(verts)
				remap[f[k]] = idx
				verts = append(verts, pts[f[k]])
			}
			nf[k] = idx
		}
		out[i] = nf
	}
	return verts, out, nil
}
