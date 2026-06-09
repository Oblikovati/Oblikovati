// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

func TestRayCastFacesHitsNearestFace(t *testing.T) {
	body := tetra(2, math.V3(0, 0, 0)) // simplex scaled ×2
	// A ray from far +Z straight down −Z toward (0.3,0.3) passes through the slanted
	// top face and the bottom z=0 face; the nearest hit is the top one.
	origin := math.P3(0.3, 0.3, 10)
	dir := math.V3(0, 0, -1)
	face, dist, ok := RayCastFaces(body, origin, dir, DefaultQuality())
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
	body := tetra(1, math.V3(0, 0, 0))
	// A ray nowhere near the body.
	if _, _, ok := RayCastFaces(body, math.P3(100, 100, 100), math.V3(0, 0, -1), DefaultQuality()); ok {
		t.Error("ray that misses the body reported a hit")
	}
}

func TestRayTriangleDistForwardOnly(t *testing.T) {
	a, b, c := math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)
	// Ray from below pointing up hits the triangle at distance 5.
	if t1, ok := rayTriangleDist(math.P3(0.2, 0.2, -5), math.V3(0, 0, 1), a, b, c); !ok || t1 < 4.99 || t1 > 5.01 {
		t.Errorf("forward hit dist = %v ok=%v, want ~5", t1, ok)
	}
	// A ray pointing away (triangle behind it) does not hit.
	if _, ok := rayTriangleDist(math.P3(0.2, 0.2, 5), math.V3(0, 0, 1), a, b, c); ok {
		t.Error("triangle behind the ray should not hit")
	}
	// A ray parallel to the triangle plane misses.
	if _, ok := rayTriangleDist(math.P3(0.2, 0.2, 1), math.V3(1, 0, 0), a, b, c); ok {
		t.Error("parallel ray should not hit")
	}
}
