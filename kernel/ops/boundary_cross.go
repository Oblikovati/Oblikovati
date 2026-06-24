// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// boundariesCross reports whether the boundary (tessellated surface) of body a crosses the boundary
// of body b. It is the missing half of containment classification (#1315): all-vertices-inside does
// NOT imply containment for non-convex operands — an edge or face of the inner body can pass back out
// through the outer body BETWEEN the inner's vertices — so a boundary crossing must demote the pair to
// the general face-splitting boolean instead of the (wrong) contains-fast-path.
//
// Each body is tessellated once; the O(Ta·Tb) triangle-pair scan is AABB-culled per triangle, and the
// whole test runs only after allVerticesInside already passed (the candidate-containment case), so it
// never burdens the common intersecting path, which exits classify earlier.
func boundariesCross(a, b *topo.Body) bool {
	ma, _ := TessellateBody(a, DefaultQuality())
	mb, _ := TessellateBody(b, DefaultQuality())
	return meshesCrossBounded(ma, mb)
}

// meshesCrossBounded reports whether any triangle of a crosses any triangle of b, culling triangle
// pairs by their axis-aligned bounding boxes before the exact Möller test (trianglesIntersect).
func meshesCrossBounded(a, b *Mesh) bool {
	boxesB := triangleBoxes(b)
	for i := 0; i+2 < len(a.Indices); i += 3 {
		ta := meshTriangle(a, i)
		boxA := math.BoxFromPoints(ta[0], ta[1], ta[2])
		for j := 0; j+2 < len(b.Indices); j += 3 {
			if !boxA.Intersects(boxesB[j/3]) {
				continue
			}
			if _, hit := trianglesIntersect(ta, meshTriangle(b, j)); hit {
				return true
			}
		}
	}
	return false
}

// triangleBoxes precomputes the AABB of every triangle of m for the cull above.
func triangleBoxes(m *Mesh) []math.Box {
	boxes := make([]math.Box, 0, m.TriangleCount())
	for i := 0; i+2 < len(m.Indices); i += 3 {
		t := meshTriangle(m, i)
		boxes = append(boxes, math.BoxFromPoints(t[0], t[1], t[2]))
	}
	return boxes
}
