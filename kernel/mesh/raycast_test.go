// SPDX-License-Identifier: GPL-2.0-only

package mesh_test

import (
	"testing"

	"oblikovati.org/kernel/mesh"
	"oblikovati.org/math"
)

func TestRayTriangleDistForwardOnly(t *testing.T) {
	t.Parallel()
	a, b, c := math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)
	// Ray from below pointing up hits the triangle at distance 5.
	if t1, ok := mesh.RayTriangleDist(math.P3(0.2, 0.2, -5), math.V3(0, 0, 1), a, b, c); !ok || t1 < 4.99 || t1 > 5.01 {
		t.Errorf("forward hit dist = %v ok=%v, want ~5", t1, ok)
	}
	// A ray pointing away (triangle behind it) does not hit.
	if _, ok := mesh.RayTriangleDist(math.P3(0.2, 0.2, 5), math.V3(0, 0, 1), a, b, c); ok {
		t.Error("triangle behind the ray should not hit")
	}
	// A ray parallel to the triangle plane misses.
	if _, ok := mesh.RayTriangleDist(math.P3(0.2, 0.2, 1), math.V3(1, 0, 0), a, b, c); ok {
		t.Error("parallel ray should not hit")
	}
}
