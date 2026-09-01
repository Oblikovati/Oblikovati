// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// foldDihedralTol: two triangles sharing an edge are a FOLD when their geometric (cross-product)
// normals point apart by more than this — cos(angle) < −foldDihedralTol — i.e. the meshed surface
// creases back on itself. A (u,v) Delaunay triangulation lifted to 3D can fold where the (u,v)→3D
// map is strongly non-conformal (the imported-NURBS "staircase"); repairFolds flips such edges.
const foldDihedralTol = 0.2

// repairFolds flips interior edges whose two triangles fold (oppose in 3D) to the quad's other
// diagonal when that removes the fold, sweeping up to maxPasses times. Each new triangle is rewound
// to agree with its vertex normals (probe.WindingOpposesNormals), so orientation stays consistent.
// Returns the number of flips applied.
func repairFolds(m *Mesh, maxPasses int) int {
	total := 0
	for range maxPasses {
		flips := repairFoldPass(m)
		total += flips
		if flips == 0 {
			break
		}
	}
	return total
}

// repairFoldPass flips every flippable fold edge once, skipping edges whose triangles were already
// flipped this pass (their adjacency is stale until the next pass rebuilds it). Returns the flips.
// Edges are visited in sorted order, not Go's randomized map order, so the flip set — and thus the
// repaired mesh — is reproducible (which edge flips first changes which triangles become dirty).
func repairFoldPass(m *Mesh) int {
	adj := edgeTriMap(m)
	dirty := map[int]bool{}
	flips := 0
	for _, e := range sortedEdgeKeys(adj) {
		ts := adj[e]
		if len(ts) != 2 || dirty[ts[0]] || dirty[ts[1]] {
			continue
		}
		if tryFlipFold(m, e, ts[0], ts[1], adj) {
			dirty[ts[0]], dirty[ts[1]] = true, true
			flips++
		}
	}
	return flips
}

// sortedEdgeKeys returns adj's edge keys in ascending (a, b) order — a deterministic visit order.
func sortedEdgeKeys(adj map[edgeKey][]int) []edgeKey {
	keys := make([]edgeKey, 0, len(adj))
	for e := range adj {
		keys = append(keys, e)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].a != keys[j].a {
			return keys[i].a < keys[j].a
		}
		return keys[i].b < keys[j].b
	})
	return keys
}

// tryFlipFold flips edge e (shared by t0,t1) to the quad's other diagonal if t0,t1 fold and the
// flip is valid (apexes found, no duplicate edge, new triangles non-degenerate + not folding).
func tryFlipFold(m *Mesh, e edgeKey, t0, t1 int, adj map[edgeKey][]int) bool {
	if !trianglesFold(m, t0, t1) {
		return false
	}
	c, d := apexOf(m, t0, e), apexOf(m, t1, e)
	if c < 0 || d < 0 {
		return false
	}
	if _, exists := adj[sortedEdge(c, d)]; exists {
		return false // flipping would duplicate an existing edge (non-manifold)
	}
	return attemptFlip(m, t0, t1, e.a, e.b, c, d)
}

type edgeKey struct{ a, b int }

func sortedEdge(a, b int) edgeKey {
	if a > b {
		a, b = b, a
	}
	return edgeKey{a, b}
}

// edgeTriMap maps each undirected edge to the triangles using it.
func edgeTriMap(m *Mesh) map[edgeKey][]int {
	adj := make(map[edgeKey][]int, len(m.Indices))
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2]}
		for k := range 3 {
			e := sortedEdge(v[k], v[(k+1)%3])
			adj[e] = append(adj[e], t)
		}
	}
	return adj
}

// apexOf returns triangle t's vertex not on edge e, or -1 if e is not an edge of t.
func apexOf(m *Mesh, t int, e edgeKey) int {
	v := [3]int{m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2]}
	onEdge := func(x int) bool { return x == e.a || x == e.b }
	if onEdge(v[0]) && onEdge(v[1]) && !onEdge(v[2]) {
		return v[2]
	}
	if onEdge(v[1]) && onEdge(v[2]) && !onEdge(v[0]) {
		return v[0]
	}
	if onEdge(v[0]) && onEdge(v[2]) && !onEdge(v[1]) {
		return v[1]
	}
	return -1
}

// triGeomNormal returns triangle t's (un-normalized) geometric normal.
func triGeomNormal(m *Mesh, t int) math.Vector3 {
	a, b, c := m.Positions[m.Indices[3*t]], m.Positions[m.Indices[3*t+1]], m.Positions[m.Indices[3*t+2]]
	return a.VectorTo(b).Cross(a.VectorTo(c))
}

// trianglesFold reports whether triangles t0,t1 oppose (a fold).
func trianglesFold(m *Mesh, t0, t1 int) bool {
	return normalsOppose(triGeomNormal(m, t0), triGeomNormal(m, t1))
}

// slantNormalRatio is how much smaller one triangle's geometric normal may be than its neighbour's
// before the pair carries no usable orientation. It is a RATIO, not a length, so it holds at any model
// scale — an absolute area floor would misjudge a micro-scale part (ADR-0042).
//
// A cross product's magnitude is twice the triangle's area, so this admits a neighbour up to a billion
// times smaller before declining to judge. Anything beyond that is a null sliver whose normal
// DIRECTION is pure rounding: the vertices are collinear to the last bit, and the direction it points
// is decided by which way the final ulp fell.
const slantNormalRatio = 1e-9

// normalsOppose reports whether two adjacent triangles' geometric normals point against each other
// (a fold). It declines to judge when either normal is degenerate NEXT TO the other's, because an
// orientation read off a null triangle is noise, not evidence.
//
// Found on macOS/arm64 (PR #2013): simple/Z1's planar face meshed a 1.24e-17-area triangle beside a
// 0.0937-area one — 5.29e-18% of the face, three decades below representable noise at that scale —
// and its rounding-decided normal read as "opposed", failing the Z1 fold gate on arm64 while amd64
// stayed clean. The repair path beside this (attemptFlip) already refuses to act on a degenerate
// normal; the DETECTOR did not, so it reported a fold the repair could never have caused or fixed.
//
// This narrows what counts as evidence of a fold; it does not narrow the gate. A fold between two
// well-formed triangles still fails exactly as before — see TestFoldDetectorIgnoresNullSliver, which
// pins both directions.
func normalsOppose(n0, n1 math.Vector3) bool {
	l0, l1 := stdmath.Sqrt(float64(n0.Dot(n0))), stdmath.Sqrt(float64(n1.Dot(n1)))
	if l0 == 0 || l1 == 0 {
		return false
	}
	if l0 < slantNormalRatio*l1 || l1 < slantNormalRatio*l0 {
		return false
	}
	return float64(n0.Dot(n1))/(l0*l1) < -foldDihedralTol
}

// attemptFlip rewrites triangles t0,t1 (sharing edge a-b, apexes c,d) to the diagonal c-d when the
// new triangles are non-degenerate and do not themselves fold — otherwise leaves the mesh unchanged
// and returns false. New triangles are rewound to their vertex normals.
func attemptFlip(m *Mesh, t0, t1, a, b, c, d int) bool {
	n0 := m.Positions[c].VectorTo(m.Positions[a]).Cross(m.Positions[c].VectorTo(m.Positions[d]))
	n1 := m.Positions[c].VectorTo(m.Positions[d]).Cross(m.Positions[c].VectorTo(m.Positions[b]))
	if degenerateNormal(n0) || degenerateNormal(n1) || normalsOppose(n0, n1) {
		return false
	}
	writeTriangle(m, t0, c, a, d)
	writeTriangle(m, t1, c, d, b)
	return true
}

func degenerateNormal(n math.Vector3) bool { return float64(n.Dot(n)) < 1e-20 }

// writeTriangle sets triangle t to (i,j,k), rewound so its geometric normal agrees with the
// vertices' surface normals (consistent with the rest of the patch).
func writeTriangle(m *Mesh, t, i, j, k int) {
	if probe.WindingOpposesNormals(m.Positions[i], m.Positions[j], m.Positions[k], m.Normals[i], m.Normals[j], m.Normals[k]) {
		j, k = k, j
	}
	m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2] = i, j, k
}
