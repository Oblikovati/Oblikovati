// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// TestBoundaryPatchConsistentWinding guards the curved-patch orientation fix: a curved face
// triangulated from its boundary must wind EVERY triangle to agree with its own vertices'
// normals, so none renders back-facing among front-facing (the inverted-normal patches seen on
// imported freeform b-spline faces). The earlier centroid-sampled flip mis-oriented triangles on
// a curved patch because the flat triangle's centroid lies off the surface.
func TestBoundaryPatchConsistentWinding(t *testing.T) {
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 10)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// A cap rim: a ring of points on the sphere at 45° latitude (a strongly curved patch with no
	// interior point, so its triangles span the curvature — where the centroid sample failed).
	const n = 24
	var rim []math.Point3
	for i := 0; i < n; i++ {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		rim = append(rim, math.P3(
			math.Scalar(10*stdmath.Cos(stdmath.Pi/4)*stdmath.Cos(a)),
			math.Scalar(10*stdmath.Cos(stdmath.Pi/4)*stdmath.Sin(a)),
			math.Scalar(10*stdmath.Sin(stdmath.Pi/4))))
	}
	m := boundaryPatchMesh(sphere, rim, nil)
	if m.TriangleCount() == 0 {
		t.Fatal("cap patch produced no triangles")
	}
	bad := 0
	for tri := 0; tri+2 < len(m.Indices); tri += 3 {
		i0, i1, i2 := m.Indices[tri], m.Indices[tri+1], m.Indices[tri+2]
		a, b, c := m.Positions[i0], m.Positions[i1], m.Positions[i2]
		gn := a.VectorTo(b).Cross(a.VectorTo(c))
		sn := m.Normals[i0].Add(m.Normals[i1]).Add(m.Normals[i2])
		if gn.Dot(sn) < 0 {
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d patch triangles wind against their vertex normals (inconsistent)", bad, m.TriangleCount())
	}
}
