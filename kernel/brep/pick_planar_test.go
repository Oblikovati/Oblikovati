// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestRayCastPlanarCapHitMiss covers the point-in-polygon planar-face pick path (ops.rayCastFace):
// a planar face is now hit-tested by ray–plane intersection + point-in-polygon on its boundary
// rather than by triangulating it, so the per-frame hover pick never runs the exact-predicate
// ear-clipper (the O(m³) big.Rat freeze that deadlocked add-in placement). This verifies the new
// path still reports the correct nearest face: a ray down the axis hits the top cap, an off-axis
// ray misses the solid entirely.
func TestRayCastPlanarCapHitMiss(t *testing.T) {
	const r, h = 5.0, 8.0
	mer := []math.Point2{math.P2(0, 0), math.P2(r, 0), math.P2(r, h), math.P2(0, h)}
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "cyl")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution = %v, %v", body, err)
	}
	q := ops.DefaultQuality()

	// Straight down the axis from above the top cap (z=h): must hit a face at ~ (100-h).
	_, dist, ok := ops.RayCastFaces(body, math.P3(0, 0, 100), math.V3(0, 0, -1), q)
	if !ok {
		t.Fatal("axis ray missed the cylinder; want a hit on the top cap")
	}
	if want := 100.0 - h; dist < want-1e-6 || dist > want+1e-6 {
		t.Errorf("axis-ray hit distance = %g, want %g (top cap at z=%g)", dist, want, h)
	}

	// A ray parallel to the axis but well outside the radius must miss every face.
	if _, _, ok := ops.RayCastFaces(body, math.P3(3*r, 0, 100), math.V3(0, 0, -1), q); ok {
		t.Error("off-axis ray hit the cylinder; want a miss (outside every face's boundary)")
	}
}
