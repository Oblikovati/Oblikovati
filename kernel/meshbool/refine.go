// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// RefineFace triangulates a face in conformance with a set of constraint segments
// — the segments where the other operand's faces cross this one (from
// IntersectTriangles). Every segment endpoint and every pairwise crossing is
// inserted as a vertex of a Delaunay triangulation, then each segment is forced as
// a chain of edges by exact flip-based recovery, so the result subdivides the face
// along the intersection curve with no segment left crossed by a triangle. The
// Delaunay CDT replaces the old cavity-force method, which #2084's near-tangent
// sliver fans defeat (pinched, non-simple cavities).
//
// PRECONDITION: each segment lies in the face's plane with endpoints on or inside
// the face (as IntersectTriangles guarantees). Points keep exact coordinates; the
// caller rounds the returned triangles.
func RefineFace(face [3]Point, segments [][2]Point) [][3]Point {
	d := newDelaunayInTriangle(face)
	for _, v := range constraintVertices(segments, d.axis) {
		d.Insert(v)
	}
	for _, s := range segments {
		d.forceSegment(s[0], s[1])
	}
	tris := d.triangles()
	// newDelaunayInTriangle stores CCW-in-projection; if the input face was CW there,
	// the sub-triangles carry the opposite normal. Restore the operand's face
	// orientation so downstream winding-number classification and the result mesh see
	// the true outward normal.
	if orient2(face[0], face[1], face[2], d.axis) < 0 {
		for i := range tris {
			tris[i][1], tris[i][2] = tris[i][2], tris[i][1]
		}
	}
	return tris
}

// constraintVertices returns every segment endpoint plus every exact pairwise
// proper-crossing point, so co-refined sub-segments meet only at shared vertices.
func constraintVertices(segments [][2]Point, axis int) []Point {
	out := make([]Point, 0, 2*len(segments))
	for _, s := range segments {
		out = append(out, s[0], s[1])
	}
	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if segmentsProperlyCross(segments[i][0], segments[i][1], segments[j][0], segments[j][1], axis) {
				out = append(out, SegSegCross(segments[i][0], segments[i][1], segments[j][0], segments[j][1], axis))
			}
		}
	}
	return out
}
