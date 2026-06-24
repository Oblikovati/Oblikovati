// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Acceptance for the curved half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334): cutting an
// analytic curved solid by ONE plane yields an EXACT curved B-rep — the kept faces preserve their
// analytic surface (a geom.Sphere, not tessellated soup) and the body's volume matches the analytic
// formula. brep.HalfSpaceCut keeps the part on the plane's negative side, capped by a planar lid.

// capVolume is the analytic spherical-cap volume π·h²(3R−h)/3, the oracle for the kept −n cap.
func capVolume(radius, height float64) float64 {
	return stdmath.Pi * height * height * (3*radius - height) / 3
}

// TestHalfSpaceCutSphereIsExactCap cuts a sphere by several planes and checks the result is a valid
// closed solid whose two faces are an exact sphere + plane, with the analytic spherical-cap volume.
func TestHalfSpaceCutSphereIsExactCap(t *testing.T) {
	const R = 5.0
	cases := []struct {
		name        string
		planePoint  math.Point3
		planeNormal math.Vector3
		height      float64 // kept (−normal) cap height h = R + d, d = (planePoint − centre)·normal
	}{
		{"hemisphere", math.P3(0, 0, 0), math.V3(0, 0, 1), R},
		{"shallow cap", math.P3(0, 0, 3), math.V3(0, 0, 1), R + 3},     // keep −z side, h = 8
		{"deep keep", math.P3(0, 0, -3), math.V3(0, 0, 1), R - 3},      // keep −z side, h = 2
		{"oblique", math.P3(0, 0, 0), math.V3(1, 1, 1), R},             // hemisphere, tilted cut
		{"oblique offset", math.P3(-2, 0, 0), math.V3(1, 0, 0), R - 2}, // keep −x side, h = 3
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sphere, err := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
			if err != nil {
				t.Fatalf("SolidSphere: %v", err)
			}
			plane, err := geom.NewPlane(tc.planePoint, tc.planeNormal)
			if err != nil {
				t.Fatalf("NewPlane: %v", err)
			}
			cap, err := brep.HalfSpaceCut(sphere, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			if r := ops.Validate(cap); !r.Valid || !r.Closed || !r.Manifold || !cap.IsSolid() {
				t.Fatalf("cap not a valid closed manifold solid: %+v", r)
			}
			assertSphereAndPlaneFaces(t, cap)
			got := ops.BodyGeometryProperties(cap, ops.DefaultQuality()).Volume
			want := capVolume(R, tc.height)
			if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
				t.Errorf("cap volume %.4f, want %.4f (rel err %.4f > 2%%)", got, want, rel)
			}
		})
	}
}

// assertSphereAndPlaneFaces verifies the result preserves the EXACT analytic surfaces: one
// geom.Sphere face (the cap) and one geom.Plane face (the lid) — not a tessellated approximation.
func assertSphereAndPlaneFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	spheres, planes := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere:
			spheres++
		case geom.Plane:
			planes++
		default:
			t.Errorf("unexpected result face surface %T (curved boolean must keep exact surfaces)", f.Geometry())
		}
	}
	if spheres != 1 || planes != 1 {
		t.Errorf("cap has %d sphere + %d plane faces, want 1 + 1", spheres, planes)
	}
}

// TestHalfSpaceCutSpherePlaneMissKeepsOrEmpties: a plane clear of the sphere keeps the whole sphere
// (centre on the negative side) or empties it (centre on the positive side).
func TestHalfSpaceCutSpherePlaneMissKeepsOrEmpties(t *testing.T) {
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	full := ops.BodyGeometryProperties(sphere, ops.DefaultQuality()).Volume

	below, _ := geom.NewPlane(math.P3(0, 0, 8), math.V3(0, 0, 1)) // sphere entirely on −z (kept) side
	kept, err := brep.HalfSpaceCut(sphere, below)
	if err != nil {
		t.Fatalf("HalfSpaceCut (miss, keep): %v", err)
	}
	if got := ops.BodyGeometryProperties(kept, ops.DefaultQuality()).Volume; stdmath.Abs(got-full) > 1e-6 {
		t.Errorf("plane clear above sphere should keep it whole: volume %.4f, want %.4f", got, full)
	}

	above, _ := geom.NewPlane(math.P3(0, 0, -8), math.V3(0, 0, 1)) // sphere entirely on +z (dropped) side
	empty, err := brep.HalfSpaceCut(sphere, above)
	if err != nil {
		t.Fatalf("HalfSpaceCut (miss, empty): %v", err)
	}
	if empty.IsSolid() && len(empty.Faces()) > 0 {
		t.Errorf("plane clear below sphere should empty it, got %d faces", len(empty.Faces()))
	}
}
