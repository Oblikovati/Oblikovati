// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone–cylinder imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). A tapered rod (a narrow frustum) crossing
// a fatter cylinder must trace as two clean closed loops (the rod's entry and exit), each lying on BOTH the
// cone and the cylinder surface to tolerance — the SSI foundation the later boolean slices build on.

// TestConeCylinderImprintTwoCleanLoops crosses a frustum (axis x, radius 1→2.5 over x∈[-6,6]) through a
// radius-3 cylinder (axis z) and checks the trace is two closed loops, each on both surfaces.
func TestConeCylinderImprintTwoCleanLoops(t *testing.T) {
	t.Parallel()
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	loops, ok := coneCylinderImprint(cone, cyl, nil)
	if !ok {
		t.Fatal("cone–cylinder imprint declined; want two crossing loops")
	}
	if n := len(loops); n != 2 {
		t.Fatalf("imprint traced %d loops, want 2 (the rod's entry and exit)", n)
	}
	cn, _, _, _ := coneSolidParams(facesOfAny(cone))
	cy, _, _, _ := cylinderSolidParams(facesOfAny(cyl))
	if d := worstLoopOffset(loops, cn, cy); d > 1e-4 {
		t.Errorf("imprint loops stray %.2e from the surfaces, want ≤ 1e-4 (must lie on both)", d)
	}
}

// TestConeCylinderImprintOrderIndependent: resolving works whichever body is passed first.
func TestConeCylinderImprintOrderIndependent(t *testing.T) {
	t.Parallel()
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := coneCylinderImprint(cyl, cone, nil); !ok {
		t.Error("cone–cylinder imprint should resolve with the cylinder passed first too")
	}
}

// TestConeCylinderImprintTwoCylindersDefer: two cylinders are not the cone–cylinder case.
func TestConeCylinderImprintTwoCylindersDefer(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := coneCylinderImprint(a, b, nil); ok {
		t.Error("two cylinders should defer from the cone–cylinder imprint (ok=false)")
	}
}

// worstLoopOffset is the largest distance of any loop vertex from either surface.
func worstLoopOffset(loops []geom.Curve3, cone geom.Cone, cyl geom.Cylinder) float64 {
	worst := 0.0
	for _, lp := range loops {
		for _, p := range imprintLoopPoints(lp) {
			worst = stdmath.Max(worst, stdmath.Max(distToCylinderSurface(cyl, p), distToConeSurface(cone, p)))
		}
	}
	return worst
}

func distToCylinderSurface(c geom.Cylinder, p math.Point3) float64 {
	return stdmath.Abs(float64(radialOf(p, cylAxis(c)).Length()) - c.Radius)
}

func distToConeSurface(c geom.Cone, p math.Point3) float64 {
	axis := c.AxisDir.AsVector()
	v := float64(c.Apex.VectorTo(p).Dot(axis))
	radial := c.Apex.VectorTo(p).Sub(axis.Scale(math.Scalar(v)))
	return stdmath.Abs((float64(radial.Length()) - v*stdmath.Tan(c.HalfAngle)) * stdmath.Cos(c.HalfAngle))
}
