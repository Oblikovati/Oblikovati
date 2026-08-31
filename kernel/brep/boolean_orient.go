// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/math"

// reorientFaces repairs the global orientation of a stitched shell: it makes every face wound
// consistently with its neighbours (each shared edge traversed in opposite directions by its two
// faces) and then, if the shell ends up inward-facing, flips the whole thing outward. The
// boolean classification orients each kept sub-face from its source face's normal, which is
// correct for clean overlaps but can be flipped for some sub-faces on complex geometry — e.g. a
// helical coil unioned onto a cylinder, where the result came back manifold-but-inconsistent.
// For an already-consistent, outward shell this is a no-op: the flood fill flips nothing and the
// signed volume is positive.
func reorientFaces(faces []builtFace, verts []math.Point3) {
	flip := orientationFlips(faces)
	for i := range faces {
		if flip[i] {
			reverseFaceRings(&faces[i])
		}
	}
	if signedVolume(faces, verts) < 0 {
		for i := range faces {
			reverseFaceRings(&faces[i])
		}
	}
}

// orientFlipNeighbour pairs a neighbour face with whether the shared edge is traversed the SAME
// direction by both (which means one of them must be flipped for consistency).
type orientFlipNeighbour struct {
	face    int
	sameDir bool
}

// faceAdjacency lists, per face, the faces it shares a manifold edge with and whether that edge
// is traversed the SAME direction by both (meaning one must be flipped for consistency). Edges
// used other than exactly twice (tangent contacts, resolved separately) are skipped.
func faceAdjacency(faces []builtFace) [][]orientFlipNeighbour {
	adj := make([][]orientFlipNeighbour, len(faces))
	for _, us := range collectEdgeUses(faces) {
		if len(us) != 2 {
			continue
		}
		same := us[0].reversed == us[1].reversed
		adj[us[0].face] = append(adj[us[0].face], orientFlipNeighbour{us[1].face, same})
		adj[us[1].face] = append(adj[us[1].face], orientFlipNeighbour{us[0].face, same})
	}
	return adj
}

// orientationFlips two-colours the face-adjacency graph (BFS over each connected shell) so that,
// after flipping the marked faces, every shared edge is traversed in opposite directions.
func orientationFlips(faces []builtFace) []bool {
	adj := faceAdjacency(faces)
	flip := make([]bool, len(faces))
	seen := make([]bool, len(faces))
	for s := range faces {
		if seen[s] {
			continue
		}
		seen[s] = true
		for q := []int{s}; len(q) > 0; {
			f := q[0]
			q = q[1:]
			for _, e := range adj[f] {
				if seen[e.face] {
					continue
				}
				seen[e.face] = true
				flip[e.face] = flip[f] != e.sameDir // consistency: flip[nbr] = flip[f] XOR sameDir
				q = append(q, e.face)
			}
		}
	}
	return flip
}

// reverseFaceRings flips a face by reversing the winding of all its loop rings and negating its
// stored outward normal.
func reverseFaceRings(f *builtFace) {
	for ri := range f.rings {
		r := f.rings[ri]
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
	}
	for i, h := range f.exactHoles {
		f.exactHoles[i] = reverseCurvedLoop(h) // detached exact holes flip with the face (ADR-0058)
	}
	f.normal = f.normal.Scale(-1)
}

// signedVolume is six times the signed volume of the shell (sum of origin-tetrahedra over each
// outer ring's fan triangulation). Its sign tells whether the consistently-wound shell faces
// outward (positive) or inward (negative); holes are omitted as they do not change the sign.
func signedVolume(faces []builtFace, verts []math.Point3) float64 {
	var v float64
	for _, f := range faces {
		if len(f.rings) == 0 || len(f.rings[0]) < 3 {
			continue
		}
		r := f.rings[0]
		a := asVec(verts[r[0]])
		for i := 1; i+1 < len(r); i++ {
			v += a.Dot(asVec(verts[r[i]]).Cross(asVec(verts[r[i+1]])))
		}
	}
	return v
}

func asVec(p math.Point3) math.Vector3 { return math.V3(p.X, p.Y, p.Z) }
