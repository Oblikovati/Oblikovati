// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR5 — targeted coverage for the closed-form spring/foot branches the D5/E4 corpus (a torus arm on a
// sphere+plane host) never exercises: the CYLINDER arm on two plane hosts, the plane/sphere host ordering,
// and the line∩plane foot. Real analytic inputs, exact assertions — no coverage padding.

func TestSpherePlaneHostsOrderIndependent(t *testing.T) {
	sp := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 150}
	pl, err := geom.NewPlane(math.P3(0, 0, 10), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	if s, p, ok := spherePlaneHosts(sp, pl); !ok || s.Radius != 150 || p.Origin.Z != 10 {
		t.Fatalf("spherePlaneHosts(sphere,plane) = (%v,%v,%v), want the sphere+plane", s, p, ok)
	}
	if s, p, ok := spherePlaneHosts(pl, sp); !ok || s.Radius != 150 || p.Origin.Z != 10 {
		t.Fatalf("spherePlaneHosts(plane,sphere) = (%v,%v,%v), want order-independent match", s, p, ok)
	}
	if _, _, ok := spherePlaneHosts(pl, pl); ok {
		t.Fatal("spherePlaneHosts(plane,plane) = ok, want decline (no sphere host)")
	}
}

// TestCylinderArmSpringsTwoPlaneHosts drives the cylinder-arm branch of armSprings: a cylinder tangent to
// two orthogonal plane hosts yields two axis rulings through the tangent-contact points.
func TestCylinderArmSpringsTwoPlaneHosts(t *testing.T) {
	cyl, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 10)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	plA, _ := geom.NewPlane(math.P3(10, 0, 0), math.V3(1, 0, 0)) // touched by the cyl at x=+10
	plB, _ := geom.NewPlane(math.P3(0, 10, 0), math.V3(0, 1, 0)) // touched at y=+10
	springs, ok := armSprings(cyl, plA, plB, 0)
	if !ok {
		t.Fatal("armSprings(cylinder, plane, plane) declined, want two rulings")
	}
	ruling, isLine := springs[0].(geom.Line)
	if !isLine {
		t.Fatalf("first cylinder spring is %T, want geom.Line", springs[0])
	}
	// the ruling is an axis-parallel line on the cylinder surface: its origin sits one radius off the axis.
	if r := stdmath.Hypot(float64(ruling.Origin.X), float64(ruling.Origin.Y)); stdmath.Abs(r-10) > 1e-9 {
		t.Fatalf("ruling A origin %v is %.4f from the axis, want the radius 10", ruling.Origin, r)
	}
	if d := stdmath.Abs(float64(ruling.Dir.AsVector().Dot(math.V3(0, 0, 1)))); d < 1-1e-9 {
		t.Fatalf("ruling A direction %v is not axis-parallel (|·ẑ|=%.4f)", ruling.Dir, d)
	}
	if _, ok := cylinderArmSprings(cyl, plA, geom.Sphere{Radius: 1}); ok {
		t.Fatal("cylinderArmSprings with a non-plane host = ok, want decline")
	}
	if _, ok := armSprings(geom.Sphere{Radius: 1}, plA, plB, 0); ok {
		t.Fatal("armSprings on an unrecognized (sphere) arm = ok, want the default decline")
	}
}

// TestSpringCapFootLine covers the ruling∩capping-plane foot and the two declines (non-plane capping, and a
// ruling parallel to the capping plane).
func TestSpringCapFootLine(t *testing.T) {
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(20, 20, 20)})
	ruling, _ := geom.NewLine(math.P3(10, 0, 0), math.V3(0, 0, 1)) // vertical ruling at x=10
	capPlane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, 1))
	foot, ok := springCapFoot(ruling, capPlane, math.P3(10, 0, 5), res)
	if !ok || foot.DistanceTo(math.P3(10, 0, 5)) > 1e-9 {
		t.Fatalf("springCapFoot(line) = (%v,%v), want (10,0,5)", foot, ok)
	}
	if _, ok := springCapFoot(ruling, geom.Sphere{Radius: 1}, math.P3(0, 0, 0), res); ok {
		t.Fatal("springCapFoot with a non-plane capping = ok, want decline")
	}
	parallel, _ := geom.NewLine(math.P3(0, 0, 5), math.V3(1, 0, 0)) // lies in the z=5 capping plane
	if _, ok := linePlaneFoot(parallel, capPlane); ok {
		t.Fatal("linePlaneFoot for a ruling parallel to the plane = ok, want decline")
	}
}
