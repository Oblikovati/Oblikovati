// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/brep"
	m "oblikovati.org/math"
)

func TestPointInsideRayThroughCornerIsRobust(t *testing.T) {
	t.Parallel()
	// Cube [0,2]^3: centre (1,1,1); (1,1,1)+t·(0.5773,0.5774,0.5775) reaches the corner (2,2,2).
	cube, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(2, 2, 2), "cube")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if !PointInsideBody(cube, m.P3(1, 1, 1)) {
		t.Error("centre of cube classified outside — the corner-grazing ray degeneracy is not handled")
	}
	if PointInsideBody(cube, m.P3(3, 3, 3)) {
		t.Error("far point classified inside")
	}
}

func TestPointInsideCubeFacePlanes(t *testing.T) {
	t.Parallel()
	cube, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(2, 2, 2), "cube")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	const eps = 1e-4
	cases := []struct {
		p    m.Point3
		want bool
	}{
		{m.P3(eps, 1, 1), true}, {m.P3(-eps, 1, 1), false},
		{m.P3(2-eps, 1, 1), true}, {m.P3(2+eps, 1, 1), false},
		{m.P3(1, eps, 1), true}, {m.P3(1, -eps, 1), false},
		{m.P3(1, 1, 2-eps), true}, {m.P3(1, 1, 2+eps), false},
	}
	for _, c := range cases {
		if got := PointInsideBody(cube, c.p); got != c.want {
			t.Errorf("PointInsideBody(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestAllVerticesInsideCubeInCube(t *testing.T) {
	t.Parallel()
	outer, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 10), "outer")
	if err != nil {
		t.Fatalf("outer: %v", err)
	}
	inner, err := brep.SolidBlock(m.P3(3, 3, 3), m.P3(6, 6, 6), "inner")
	if err != nil {
		t.Fatalf("inner: %v", err)
	}
	if !allVerticesInside(inner, outer) {
		t.Error("inner cube vertices should all be inside outer cube")
	}
	if allVerticesInside(outer, inner) {
		t.Error("outer cube vertices should not be inside inner cube")
	}
	apart, err := brep.SolidBlock(m.P3(20, 20, 20), m.P3(22, 22, 22), "apart")
	if err != nil {
		t.Fatalf("apart: %v", err)
	}
	if allVerticesInside(apart, outer) {
		t.Error("disjoint cube vertices should not be inside outer cube")
	}
}
