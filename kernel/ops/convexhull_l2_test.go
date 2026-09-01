// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops/query"
	m "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1323 L2: the incremental hull's "point sees face" test used a
// float visibility tolerance, fragile on the coplanar inputs hull() is built for (boxes). It now uses
// the exact Orient3D predicate, so the cube hull is correct and stable under input reordering.

func cubeCorners(s float64) []m.Point3 {
	return []m.Point3{
		m.P3(0, 0, 0), m.P3(m.Scalar(s), 0, 0), m.P3(m.Scalar(s), m.Scalar(s), 0), m.P3(0, m.Scalar(s), 0),
		m.P3(0, 0, m.Scalar(s)), m.P3(m.Scalar(s), 0, m.Scalar(s)), m.P3(m.Scalar(s), m.Scalar(s), m.Scalar(s)), m.P3(0, m.Scalar(s), m.Scalar(s)),
	}
}

// TestConvexHullCubeExactFaceCountAndVolume asserts the cube hull is exactly 6 quads = 12 triangles
// with the right volume and no sliver faces.
func TestConvexHullCubeExactFaceCountAndVolume(t *testing.T) {
	t.Parallel()
	const s = 2.0
	body, err := query.ConvexHull(cubeCorners(s), "cube")
	if err != nil {
		t.Fatalf("query.ConvexHull: %v", err)
	}
	if nf := len(body.Faces()); nf != 12 {
		t.Errorf("cube hull has %d triangular faces, want 12 (6 quads)", nf)
	}
	if v := query.BodyGeometryProperties(body, DefaultQuality()).Volume; math.Abs(v-s*s*s) > 1e-9 {
		t.Errorf("cube hull volume = %g, want %g", v, s*s*s)
	}
}

// TestConvexHullStableUnderReordering permutes the input corner order and perturbs coordinates by a
// hair; the hull volume and triangle count must be invariant (the exact predicate removes the
// reordering/coplanar fragility).
func TestConvexHullStableUnderReordering(t *testing.T) {
	t.Parallel()
	const s = 2.0
	base := cubeCorners(s)
	orders := [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7},
		{7, 6, 5, 4, 3, 2, 1, 0},
		{0, 2, 4, 6, 1, 3, 5, 7},
		{6, 0, 3, 5, 1, 7, 2, 4},
	}
	for oi, order := range orders {
		pts := make([]m.Point3, len(order))
		for i, idx := range order {
			pts[i] = base[idx]
		}
		body, err := query.ConvexHull(pts, "cube")
		if err != nil {
			t.Fatalf("order %d: query.ConvexHull: %v", oi, err)
		}
		if nf := len(body.Faces()); nf != 12 {
			t.Errorf("order %d: %d faces, want 12", oi, nf)
		}
		if v := query.BodyGeometryProperties(body, DefaultQuality()).Volume; math.Abs(v-s*s*s) > 1e-9 {
			t.Errorf("order %d: volume %g, want %g", oi, v, s*s*s)
		}
	}
}

// TestConvexHullNearCoplanarValid feeds a box-with-an-extra-near-coplanar-top-point set: a point a
// hair above a face plane. The hull must be a valid closed solid (no inverted/extra sliver faces),
// the failure mode the float tolerance produced on near-coplanar inputs.
func TestConvexHullNearCoplanarValid(t *testing.T) {
	t.Parallel()
	pts := cubeCorners(2)
	// A point just above the centre of the top face (z = 2 + tiny): the hull gains a shallow apex.
	pts = append(pts, m.P3(1, 1, 2+1e-9))
	body, err := query.ConvexHull(pts, "near-coplanar")
	if err != nil {
		t.Fatalf("query.ConvexHull: %v", err)
	}
	if !body.IsSolid() {
		t.Error("near-coplanar hull is not a solid")
	}
	if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold {
		t.Errorf("near-coplanar hull invalid: %+v", r)
	}
	// Volume is essentially the cube (the apex adds a negligible sliver).
	if v := query.BodyGeometryProperties(body, DefaultQuality()).Volume; math.Abs(v-8) > 1e-6 {
		t.Errorf("near-coplanar hull volume = %g, want ~8", v)
	}
}
