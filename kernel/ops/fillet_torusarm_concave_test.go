// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// weTestTorus builds the shared host torus for these tests (fatal on the impossible constructor error).
func weTestTorus(t *testing.T, major, minor float64) geom.Torus {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), major, minor)
	if err != nil {
		t.Fatalf("host torus (R=%g a=%g): %v", major, minor, err)
	}
	return tor
}

// TestConcaveTorusHostArmSurfaceExternal pins the J5 cove arm against DRAWEXE 8.0.0: host R=200 a=50,
// crater-floor cap at capH=25 (n̂=+ẑ), rim at ρ=156.7 (inner branch), r=10, material INSIDE the tube
// (s=+1) → spine centre (0,0,35), major 200−√((a+r)²−35²) = 200−√2375 = 151.266028 (the value DRAWEXE's
// blend band poles decode to), minor 10.
func TestConcaveTorusHostArmSurfaceExternal(t *testing.T) {
	host := weTestTorus(t, 200, 50)
	arm, reject := concaveTorusHostArmSurface(host, 25, 156.7, 10, +1, +1, ResolutionForSize(500))
	if reject != torusArmBuilt {
		t.Fatalf("J5 arm rejected: %d", reject)
	}
	wantMajor := 200 - stdmath.Sqrt(2375)
	if d := stdmath.Abs(arm.MajorRadius - wantMajor); d > 1e-9 {
		t.Fatalf("J5 arm major %.9f, want %.9f (Δ %.3g)", arm.MajorRadius, wantMajor, d)
	}
	if d := float64(arm.Center.DistanceTo(math.P3(0, 0, 35))); d > 1e-9 || arm.MinorRadius != 10 {
		t.Fatalf("J5 arm centre %v minor %g, want (0,0,35) / 10", arm.Center, arm.MinorRadius)
	}
}

// TestConcaveTorusHostArmSurfaceInternal pins the A6 internal-tangency branch: host R=200 a=100,
// cap at capH=0, rim at ρ=100 (inner), r=30, material OUTSIDE the tube (s=−1) → η=30, tube offset
// a−r=70, major 200−√(70²−30²) = 200−√4000 = 136.754447 — DRAWEXE grows A6's plate hole to exactly
// this radius.
func TestConcaveTorusHostArmSurfaceInternal(t *testing.T) {
	host := weTestTorus(t, 200, 100)
	arm, reject := concaveTorusHostArmSurface(host, 0, 100, 30, +1, -1, ResolutionForSize(800))
	if reject != torusArmBuilt {
		t.Fatalf("A6 arm rejected: %d", reject)
	}
	wantMajor := 200 - stdmath.Sqrt(4000)
	if d := stdmath.Abs(arm.MajorRadius - wantMajor); d > 1e-9 {
		t.Fatalf("A6 arm major %.9f, want %.9f (Δ %.3g)", arm.MajorRadius, wantMajor, d)
	}
	if d := float64(arm.Center.DistanceTo(math.P3(0, 0, 30))); d > 1e-9 {
		t.Fatalf("A6 arm centre %v, want (0,0,30)", arm.Center)
	}
}

// TestConcaveTorusHostArmSurfaceRejects exercises the guards: an internal ball consuming the tube
// (s=−1, r ≥ a) and a cap so far off-axis the offset plane clears the offset tube.
func TestConcaveTorusHostArmSurfaceRejects(t *testing.T) {
	host := weTestTorus(t, 200, 50)
	if _, reject := concaveTorusHostArmSurface(host, 0, 250, 60, +1, -1, ResolutionForSize(500)); reject != torusArmSpindle {
		t.Fatalf("r=60 ≥ a=50 internal ball: got %d, want torusArmSpindle", reject)
	}
	if _, reject := concaveTorusHostArmSurface(host, 200, 250, 10, +1, +1, ResolutionForSize(500)); reject != torusArmClears {
		t.Fatalf("capH=200 far above the tube: got %d, want torusArmClears", reject)
	}
}

// TestTubeContactScale pins the contact-projection factor on both tangency branches and the reject:
// external (hyp = a+r → k = a/(a+r)), internal (hyp = a−r → k = a/(a−r)), neither → false.
func TestTubeContactScale(t *testing.T) {
	if k, ok := tubeContactScale(60, 50, 10, 1e-6); !ok || stdmath.Abs(k-50.0/60) > 1e-12 {
		t.Fatalf("external: k=%.12f ok=%v, want 50/60", k, ok)
	}
	if k, ok := tubeContactScale(70, 100, 30, 1e-6); !ok || stdmath.Abs(k-100.0/70) > 1e-12 {
		t.Fatalf("internal: k=%.12f ok=%v, want 100/70", k, ok)
	}
	if _, ok := tubeContactScale(90, 50, 10, 1e-6); ok {
		t.Fatal("hyp=90 certifies neither branch — must reject")
	}
}

// TestConcaveTorusHostTubeContact pins the J5 host-tube contact circle against DRAWEXE's band poles:
// arm centre (0,0,35), major 151.266028 → contact centre z = 35·(50/60) = 29.1666667, radius
// 200 + (5/6)(151.266028−200) = 159.388357 (DRAWEXE row-1 pole radius 159.388356896629).
func TestConcaveTorusHostTubeContact(t *testing.T) {
	host := weTestTorus(t, 200, 50)
	arm, err := geom.NewTorusWithRef(math.P3(0, 0, 35), math.V3(0, 0, 1), math.V3(1, 0, 0), 200-stdmath.Sqrt(2375), 10)
	if err != nil {
		t.Fatal(err)
	}
	center, radius, ok := concaveTorusHostTubeContact(host, arm, ResolutionForSize(500))
	if !ok {
		t.Fatal("J5 contact circle unresolved")
	}
	if d := float64(center.DistanceTo(math.P3(0, 0, 35.0*50/60))); d > 1e-6 {
		t.Fatalf("contact centre %v, want (0,0,29.1667)", center)
	}
	want := 200 + (50.0/60)*(200-stdmath.Sqrt(2375)-200)
	if d := stdmath.Abs(radius - want); d > 1e-6 {
		t.Fatalf("contact radius %.9f, want %.9f (DRAWEXE 159.388356896629)", radius, want)
	}
}
