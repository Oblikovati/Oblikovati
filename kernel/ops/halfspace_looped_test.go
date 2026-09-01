// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Looped split acceptance (M2 Phase 1, Oblikovati/Oblikovati#1334): the second of two composed
// HalfSpaceCuts crosses a curved face (a sphere cap) and its planar lid, exercising loopedSplit on both
// plus chaining the section arc + line into one lid. The oracle is SYMMETRY: cutting a body by a plane
// of symmetry through its centroid yields exactly half its volume — exact and tessellation-robust when
// the kept sphere patch stays within a hemisphere (a larger patch is a spherePatchMesh follow-up, not a
// split-logic concern).

// TestLoopedSplitHalvesACapBySymmetry cuts a small spherical cap (kept z ≤ −3, a sub-hemisphere cap) by
// a plane of symmetry (x ≤ 0, y ≤ 0): the result must be a valid analytic solid of half the cap's volume.
func TestLoopedSplitHalvesACapBySymmetry(t *testing.T) {
	t.Parallel()
	const R = 5.0
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
	capPlane, _ := geom.NewPlane(math.P3(0, 0, -3), math.V3(0, 0, 1)) // keep z ≤ −3
	cap, err := brep.HalfSpaceCut(sphere, capPlane)
	if err != nil {
		t.Fatalf("cap cut: %v", err)
	}
	capVol := query.BodyGeometryProperties(cap, ops.DefaultQuality()).Volume

	for _, sym := range []struct {
		name   string
		normal math.Vector3
	}{
		{"halve by x=0", math.V3(1, 0, 0)},
		{"halve by y=0", math.V3(0, 1, 0)},
	} {
		t.Run(sym.name, func(t *testing.T) {
			plane, _ := geom.NewPlane(math.P3(0, 0, 0), sym.normal) // symmetry plane through the axis
			half, err := brep.HalfSpaceCut(cap, plane)
			if err != nil {
				t.Fatalf("looped cut: %v", err)
			}
			if r := ops.Validate(half); !r.Valid || !r.Closed || !r.Manifold || !half.IsSolid() {
				t.Fatalf("looped cut is not a valid closed manifold solid: %+v", r)
			}
			assertOnlyAnalyticFaces(t, half)
			got := query.BodyGeometryProperties(half, ops.DefaultQuality()).Volume
			want := capVol / 2
			if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
				t.Errorf("symmetric half volume %.4f, want %.4f (cap/2) — rel %.4f > 2%%", got, want, rel)
			}
		})
	}
}

// TestHalfSpaceCutSphereBoxVsAnalytic composes two cuts into a sphere∩box corner (z ≤ 0, x ≤ 2). The
// kept sphere face exceeds a hemisphere, so it exercises the stereographic patch chart. The result is an
// exact curved B-rep whose volume matches the analytic lower-hemisphere-minus-side-cap (acceptance #1).
func TestHalfSpaceCutSphereBoxVsAnalytic(t *testing.T) {
	t.Parallel()
	const R = 5.0
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
	pZ, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // keep z ≤ 0 (lower hemisphere)
	pX, _ := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0)) // keep x ≤ 2 (trim the +x side)

	hemi, err := brep.HalfSpaceCut(sphere, pZ)
	if err != nil {
		t.Fatalf("cut z: %v", err)
	}
	res, err := brep.HalfSpaceCut(hemi, pX)
	if err != nil {
		t.Fatalf("cut x: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("sphere∩box corner is not a valid closed manifold solid: %+v", r)
	}
	assertOnlyAnalyticFaces(t, res)
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	// Lower hemisphere minus the z<0 half of the x>2 spherical cap (height R−2=3); the two removed
	// regions do not overlap, so the volume is exact.
	capV := stdmath.Pi * 9 * (3*R - 3) / 3
	want := (2.0/3.0)*stdmath.Pi*R*R*R - capV/2
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("sphere∩box volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// assertOnlyAnalyticFaces checks the exact path kept analytic surfaces (a sphere patch + planar faces),
// not tessellated soup.
func assertOnlyAnalyticFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere, geom.Plane:
		default:
			t.Errorf("result face surface %T is not analytic (curved boolean must keep exact surfaces)", f.Geometry())
		}
	}
}
