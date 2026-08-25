// SPDX-License-Identifier: GPL-2.0-only

// Package rttest holds the shared renderer.Intersector contract-test suite (M45-F01
// PBI-332, ADR-0053). It is a separate package from renderer (mirroring
// model/feature/occtparity's isolation of *testing.T-taking exported helpers from
// production code) so PBI-333's hardware backend and PBI-334's software backend can both
// import it and be held to the exact same, CPU-verifiable behavior: swapping which
// Intersector the path tracer holds must change only speed, never correctness.
package rttest

import (
	"testing"

	"oblikovati.org/renderer"
)

// unitQuad returns the two triangles of a unit square in the z=0 plane, spanning
// [-0.5,0.5]^2 in x/y.
func unitQuad(instanceID uint32) []renderer.Triangle {
	a := [3]float32{-0.5, -0.5, 0}
	b := [3]float32{0.5, -0.5, 0}
	c := [3]float32{0.5, 0.5, 0}
	d := [3]float32{-0.5, 0.5, 0}
	return []renderer.Triangle{
		{V0: a, V1: b, V2: c, InstanceID: instanceID, PrimitiveID: 0},
		{V0: a, V1: c, V2: d, InstanceID: instanceID, PrimitiveID: 1},
	}
}

// RunIntersectorContractTests exercises an renderer.Intersector implementation built by
// newIntersector against a small quad scene: a straight hit (point/normal/instance id),
// a miss, and TMin/TMax clipping. Call this from the backend's own *_test.go with a
// factory over the real hardware/software backend (PBI-333/334) — it is exactly
// renderer's own FakeIntersector tests, generalized to any factory.
func RunIntersectorContractTests(t *testing.T, newIntersector renderer.IntersectorFactory) {
	t.Helper()
	t.Run("StraightHit", func(t *testing.T) { testStraightHit(t, newIntersector) })
	t.Run("Miss", func(t *testing.T) { testMiss(t, newIntersector) })
	t.Run("RespectsTMinTMax", func(t *testing.T) { testRespectsTMinTMax(t, newIntersector) })
	t.Run("NearestOfTwo", func(t *testing.T) { testNearestOfTwo(t, newIntersector) })
}

// testNearestOfTwo scenes two parallel quads (instances 1 and 2, the second behind the
// first) and asserts a straight-down ray reports the nearer instance — every backend
// must return the CLOSEST hit, not merely *a* hit.
func testNearestOfTwo(t *testing.T, newIntersector renderer.IntersectorFactory) {
	near := unitQuad(1)
	far := unitQuad(2)
	for i := range far {
		far[i].V0[2], far[i].V1[2], far[i].V2[2] = -1, -1, -1
	}
	in := newIntersector(append(near, far...))
	ray := renderer.Ray{Origin: [3]float32{0, 0, 2}, Direction: [3]float32{0, 0, -1}, TMin: 0, TMax: 1e6}

	got := in.TraceRay(ray)
	if !got.Hit || got.InstanceID != 1 {
		t.Errorf("expected the nearer instance 1 (t=2) over instance 2 (t=3), got %+v", got)
	}
}

func testStraightHit(t *testing.T, newIntersector renderer.IntersectorFactory) {
	in := newIntersector(unitQuad(7))
	ray := renderer.Ray{Origin: [3]float32{0, 0, 2}, Direction: [3]float32{0, 0, -1}, TMin: 0, TMax: 1e6}

	got := in.TraceRay(ray)
	if !got.Hit {
		t.Fatal("expected a hit on the quad")
	}
	if got.T != 2 {
		t.Errorf("t = %v, want 2 (2 units along -z from origin)", got.T)
	}
	if got.Point != [3]float32{0, 0, 0} {
		t.Errorf("point = %v, want origin (plane z=0)", got.Point)
	}
	if got.InstanceID != 7 {
		t.Errorf("instance id = %d, want 7", got.InstanceID)
	}
	if got.Normal[0] != 0 || got.Normal[1] != 0 || (got.Normal[2] != 1 && got.Normal[2] != -1) {
		t.Errorf("normal = %v, want (0,0,±1)", got.Normal)
	}
}

func testMiss(t *testing.T, newIntersector renderer.IntersectorFactory) {
	in := newIntersector(unitQuad(1))
	ray := renderer.Ray{Origin: [3]float32{10, 10, 2}, Direction: [3]float32{0, 0, -1}, TMin: 0, TMax: 1e6}
	if got := in.TraceRay(ray); got.Hit {
		t.Errorf("expected a miss, got %+v", got)
	}
}

func testRespectsTMinTMax(t *testing.T, newIntersector renderer.IntersectorFactory) {
	in := newIntersector(unitQuad(1))
	ray := renderer.Ray{Origin: [3]float32{0, 0, 2}, Direction: [3]float32{0, 0, -1}}

	ray.TMin, ray.TMax = 0, 1
	if got := in.TraceRay(ray); got.Hit {
		t.Errorf("TMax=1 should clip the t=2 hit, got %+v", got)
	}
	ray.TMin, ray.TMax = 3, 1e6
	if got := in.TraceRay(ray); got.Hit {
		t.Errorf("TMin=3 should clip the t=2 hit, got %+v", got)
	}
	ray.TMin, ray.TMax = 0, 1e6
	if got := in.TraceRay(ray); !got.Hit {
		t.Error("expected a hit with the full [0, 1e6] range")
	}
}
