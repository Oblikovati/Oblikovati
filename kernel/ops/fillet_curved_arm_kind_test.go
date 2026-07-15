// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// testArmResolution mimics the corpus B3 body scale (pcylinder R=50, height=100 —
// m5-curved-arm-derivation.md §OCCT oracle) so the classification band (§Numerical pitfalls)
// exercises the same order of magnitude as the real fillet corpus, not an arbitrary unit size.
func testArmResolution() Resolution { return ResolutionForSize(150) }

// cylAxis builds a unit-length-axis cylinder at the origin for classifyCurvedArm tests: axis
// (ax,ay,az), radius r. Panics on a zero axis — a test author error, not a runtime case.
func cylAxis(ax, ay, az, r float64) geom.Cylinder {
	c, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(ax, ay, az), r)
	if err != nil {
		panic(err)
	}
	return c
}

// planeWithNormal builds a plane through the origin with normal (nx,ny,nz). Named distinctly
// from the existing face-normal helper planeNormal (fillet_miter.go) — same package, different
// signature, would otherwise collide.
func planeWithNormal(nx, ny, nz float64) geom.Plane {
	p, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(nx, ny, nz))
	if err != nil {
		panic(err)
	}
	return p
}

func TestClassifyCurvedArm(t *testing.T) {
	res := testArmResolution()
	// axis ⊥ plane (|s|=1) → torus arm (B3 top-rim edge)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeWithNormal(0, 0, 1), res); k != armTorus {
		t.Fatalf("axis ⊥ plane: want armTorus, got %v", k)
	}
	// axis ∥ plane (s=0) → cylinder arm (B3 vertical-wall edge)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeWithNormal(1, 0, 0), res); k != armCylinder {
		t.Fatalf("axis ∥ plane: want armCylinder, got %v", k)
	}
	// oblique → rejected (Slice B)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeWithNormal(0, 0.6, 0.8), res); k != armRejected {
		t.Fatalf("oblique: want armRejected, got %v", k)
	}
}
