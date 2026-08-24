// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact ear-clipping triangulation of a simple polygon in a face's projection.
// This is the cavity-retriangulation primitive constraint-edge insertion needs:
// when a constraint segment carves a cavity out of the triangulation, each side of
// the cavity is a simple polygon that must be re-triangulated with the segment as
// one edge. Ear clipping any simple polygon yields a valid triangulation; exact
// orient2 makes the convexity and containment tests exact.

// triangulateSimplePolygon triangulates the simple polygon poly (boundary order,
// either winding) into triangles, PRESERVING every vertex — collinear boundary
// vertices stay (a co-refinement cavity vertex must remain for conformance with
// the neighbouring triangle); such a vertex is simply not an ear until a neighbour
// is clipped and it turns convex. Reflex vertices are handled. PRECONDITION: poly
// is a simple polygon (no self-intersections, no spikes) in the plane of axis.
func triangulateSimplePolygon(poly []Point, axis int) [][3]Point {
	ring := append([]Point(nil), poly...)
	if signedArea2(ring, axis).Sign() < 0 {
		reverseRing(ring) // normalize to CCW so an ear has orient2 > 0
	}
	idx := identityIdx(len(ring))
	var out [][3]Point
	for len(idx) > 3 {
		tri, rest, ok := clipOneEar(ring, idx, axis)
		if !ok {
			return out // defensive: input was not a simple polygon
		}
		idx = rest
		out = append(out, tri)
	}
	if len(idx) == 3 {
		out = append(out, [3]Point{ring[idx[0]], ring[idx[1]], ring[idx[2]]})
	}
	return out
}

// clipOneEar removes one convex ear from idx and returns its triangle. ok is false
// only if no ear was found, which cannot happen for a simple polygon with >3
// vertices (every such polygon has at least one strictly-convex ear).
func clipOneEar(ring []Point, idx []int, axis int) (tri [3]Point, rest []int, ok bool) {
	for i := range idx {
		a := ring[idx[prevMod(i, len(idx))]]
		b := ring[idx[i]]
		c := ring[idx[nextMod(i, len(idx))]]
		if orient2(a, b, c, axis) > 0 && isEar(a, b, c, ring, idx, axis) {
			return [3]Point{a, b, c}, removeAt(idx, i), true
		}
	}
	return tri, idx, false
}

// isEar reports whether the convex corner (a,b,c) is an ear: no other remaining
// vertex lies inside or on triangle abc.
func isEar(a, b, c Point, ring []Point, idx []int, axis int) bool {
	for _, j := range idx {
		p := ring[j]
		if p.Equal(a) || p.Equal(b) || p.Equal(c) {
			continue
		}
		if orient2(a, b, p, axis) >= 0 && orient2(b, c, p, axis) >= 0 && orient2(c, a, p, axis) >= 0 {
			return false // p inside the candidate ear (a,b,c is CCW)
		}
	}
	return true
}

// signedArea2 returns twice the exact signed area of the polygon (shoelace via
// fan determinants from vertex 0).
func signedArea2(poly []Point, axis int) *big.Rat {
	sum := new(big.Rat)
	for i := 1; i+1 < len(poly); i++ {
		sum.Add(sum, orient2Val(poly[0], poly[i], poly[i+1], axis))
	}
	return sum
}

func reverseRing(r []Point) {
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
}

func identityIdx(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func removeAt(idx []int, i int) []int {
	out := make([]int, 0, len(idx)-1)
	out = append(out, idx[:i]...)
	return append(out, idx[i+1:]...)
}

func prevMod(i, n int) int { return (i - 1 + n) % n }
func nextMod(i, n int) int { return (i + 1) % n }
