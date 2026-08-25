// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "testing"

// TestFakeIntersectorParallelRayMisses covers rayTriangle's degenerate branch (a ray
// parallel to the triangle's plane, det ≈ 0) directly — the black-box Intersector
// contract suite (renderer/rttest) always fires straight-down rays, so it never reaches
// this internal codepath.
func TestFakeIntersectorParallelRayMisses(t *testing.T) {
	quad := []Triangle{
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, -0.5, 0}, V2: [3]float32{0.5, 0.5, 0}},
	}
	fi := NewFakeIntersector(quad)
	// Travels within the z=0 plane the triangle lies in — never converges on it.
	ray := Ray{Origin: [3]float32{-10, 0, 0}, Direction: [3]float32{1, 0, 0}, TMin: 0, TMax: 1e6}
	if got := fi.TraceRay(ray); got.Hit {
		t.Errorf("a ray parallel to the triangle's plane must miss, got %+v", got)
	}
}
