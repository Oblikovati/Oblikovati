// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// testArmResolution mimics the corpus B3 body scale (pcylinder R=50, height=100 —
// m5-curved-arm-derivation.md §OCCT oracle) so the classification band (§Numerical pitfalls)
// exercises the same order of magnitude as the real fillet corpus, not an arbitrary unit size.
func testArmResolution() tol.Resolution { return tol.ForSize(150) }

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
	t.Parallel()
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
	// axis flipped, |s|=1 via dot=-1 → still torus arm. Every case above has a non-negative
	// axis·normal dot (1, 0, 0.8); this is the regression case for a dropped stdmath.Abs in
	// classifyCurvedArm's s := |cyl.AxisDir.Dot(n)| — without Abs this dot lands at s=-1,
	// which fails both the armTorus and armCylinder bands and misclassifies as armRejected.
	if k := classifyCurvedArm(cylAxis(0, 0, -1, 50), planeWithNormal(0, 0, 1), res); k != armTorus {
		t.Fatalf("flipped axis (dot=-1): want armTorus, got %v", k)
	}
	// oblique with a negative raw dot (axis flipped, dot=0.64) → rejected. Covers the sign
	// path on the reject branch too, not just the accept branches exercised above.
	if k := classifyCurvedArm(cylAxis(0, 0, -1, 50), planeWithNormal(0, 0.6, -0.8), res); k != armRejected {
		t.Fatalf("oblique, negative dot: want armRejected, got %v", k)
	}
	// near-threshold: a hair inside/outside the oblique band epsAng = angArmClassifyCoef *
	// res.Weld() / R (m5-curved-arm-derivation.md §Numerical pitfalls) — stresses the |s|
	// comparison itself, which the coarse-angle cases above (s=1, s=0, s=0.8) don't reach.
	const armRadius = 50.0
	epsAng := angArmClassifyCoef * res.Weld() / armRadius
	sInside, sOutside := 1-epsAng/2, 1-epsAng*2
	axInside := stdmath.Sqrt(1 - sInside*sInside)
	axOutside := stdmath.Sqrt(1 - sOutside*sOutside)
	if k := classifyCurvedArm(cylAxis(axInside, 0, sInside, armRadius), planeWithNormal(0, 0, 1), res); k != armTorus {
		t.Fatalf("just inside oblique band (s=%.17g): want armTorus, got %v", sInside, k)
	}
	if k := classifyCurvedArm(cylAxis(axOutside, 0, sOutside, armRadius), planeWithNormal(0, 0, 1), res); k != armRejected {
		t.Fatalf("just outside oblique band (s=%.17g): want armRejected, got %v", sOutside, k)
	}
}
