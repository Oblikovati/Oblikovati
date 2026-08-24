// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact point-in-solid classification by ray-cast parity, replacing the
// generalized winding number. A ray is cast from p and the triangles it crosses
// are counted; odd means inside. Every crossing is decided by exact rational
// Orient3D, so a face whose centroid sits almost on the other operand's surface —
// where the winding number is a borderline ~0.5 and misclassifies (the residual
// left after the CDT fixed co-refinement) — is now classified correctly. A ray that
// grazes an edge or vertex (an exact degeneracy) is retried in another direction;
// degeneracies are measure-zero, so a clean direction is found immediately.
//
// The ray-walk uses the mesh's face grid (a 3D DDA), so it tests only the triangles
// near the ray, not the whole mesh — without it the coil classification is O(faces)
// per sample and does not finish.

// rayDirections are the generic directions tried in turn until one avoids every
// edge/vertex grazing. Their components are small and mutually non-proportional.
var rayDirections = [][3]float64{
	{1, 0, 0}, {1, 2, 3}, {3, -1, 4}, {5, 7, -2}, {-2, 3, 5}, {7, -4, 1}, {2, -5, 3}, {-3, 4, -7},
}

// insideExact reports whether p is inside the closed, outward-oriented mesh indexed
// by grid.
func insideExact(p Point, mesh [][3]Point, grid *faceGrid) bool {
	pf := [3]float64(m3(p))
	for _, dir := range rayDirections {
		if crossings, ok := rayCrossings(p, pf, dir, mesh, grid); ok {
			return crossings%2 == 1
		}
	}
	return false // every direction grazed a degeneracy — astronomically unlikely
}

// rayCrossings counts how many triangles the ray from p along dir crosses, testing
// only the faces the DDA walk reaches, or reports a degeneracy so the caller retries.
func rayCrossings(p Point, pf, dir [3]float64, mesh [][3]Point, grid *faceGrid) (int, bool) {
	q := farPoint(p, dir, grid)
	count := 0
	for _, fi := range grid.rayFaces(pf, dir) {
		t := mesh[fi]
		crosses, degenerate := segmentPiercesTriExact(p, q, t[0], t[1], t[2])
		if degenerate {
			return 0, false
		}
		if crosses {
			count++
		}
	}
	return count, true
}

// segmentPiercesTriExact reports whether the segment p→q pierces the interior of
// triangle (a,b,c), and whether the test hit a degeneracy (an endpoint on the plane
// or a pierce exactly on an edge). All decisions are exact rational Orient3D.
func segmentPiercesTriExact(p, q, a, b, c Point) (crosses, degenerate bool) {
	sp := Orient3D(a, b, c, p)
	sq := Orient3D(a, b, c, q)
	if sp == 0 || sq == 0 {
		return false, true
	}
	if sp == sq {
		return false, false // both endpoints on the same side of the plane
	}
	s1 := Orient3D(p, q, a, b)
	s2 := Orient3D(p, q, b, c)
	s3 := Orient3D(p, q, c, a)
	if s1 == 0 || s2 == 0 || s3 == 0 {
		return false, true // grazes an edge
	}
	return s1 == s2 && s2 == s3, false // strictly inside iff all three agree
}

// farPoint returns p displaced along dir far enough to lie outside the mesh (the
// same distance the DDA walks), so the segment to it captures every crossing.
func farPoint(p Point, dir [3]float64, grid *faceGrid) Point {
	m := big.NewRat(int64(grid.farDistance())+1, 1)
	return Point{
		X: new(big.Rat).Add(p.X, new(big.Rat).Mul(m, ratOf(dir[0]))),
		Y: new(big.Rat).Add(p.Y, new(big.Rat).Mul(m, ratOf(dir[1]))),
		Z: new(big.Rat).Add(p.Z, new(big.Rat).Mul(m, ratOf(dir[2]))),
	}
}
