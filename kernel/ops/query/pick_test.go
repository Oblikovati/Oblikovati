// SPDX-License-Identifier: GPL-2.0-only

package query_test

import (
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

func TestRayCastFacesHitsNearestFace(t *testing.T) {
	t.Parallel()
	body := brepfixture.Tetra(2, math.V3(0, 0, 0)) // simplex scaled ×2
	// A ray from far +Z straight down −Z toward (0.3,0.3) passes through the slanted
	// top face and the bottom z=0 face; the nearest hit is the top one.
	origin := math.P3(0.3, 0.3, 10)
	dir := math.V3(0, 0, -1)
	face, dist, ok := query.RayCastFaces(body, origin, dir, ops.DefaultQuality())
	if !ok || face == nil {
		t.Fatal("ray missed a body it passes through")
	}
	// The nearest hit is well above z=0 (it is the slanted cap, not the base).
	hitZ := 10 - dist
	if hitZ <= 0.01 {
		t.Errorf("nearest hit z = %v, want the upper face (>0)", hitZ)
	}
}

func TestRayCastFacesMisses(t *testing.T) {
	t.Parallel()
	body := brepfixture.Tetra(1, math.V3(0, 0, 0))
	// A ray nowhere near the body.
	if _, _, ok := query.RayCastFaces(body, math.P3(100, 100, 100), math.V3(0, 0, -1), ops.DefaultQuality()); ok {
		t.Error("ray that misses the body reported a hit")
	}
}
