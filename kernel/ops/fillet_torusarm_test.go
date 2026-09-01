// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The E7 fixture is OCCT blend/simple/E7: a host torus (centre O, axis +ẑ, major R=100, minor a=100)
// latitude-cut by a cap plane at z = 100/√2 (material-outward normal +ẑ), rolling-ball r=10. These pin
// torusHostArmSurface's closed form and torusHostContactCircle to the mathematician-derived arm
// (fillet_torusarm.go header): M_a = R + ε·√(b²−η²) with b = a−r, η = H − r·(ẑ·n̂). The values are exact
// closed forms of {r, torus, cap plane}, so a mismatch beyond torusArmExactTol is a transcription bug.
const (
	torusArmR        = 10.0
	torusArmExactTol = 1e-9               // exact-arithmetic transcription tolerance (mirrors coneArmExactTol)
	e7CapZ           = 70.71067811865476  // the E7 cap-plane z: 100/√2 = a·sin(45°)
	e7Rho            = 170.71067811865476 // E7 rim radial coord ρ_rim (outer branch, > R) — only its sign vs R matters
)

// e7HostTorus builds the E7 host torus: centre origin, axis +ẑ, major 100, minor 100.
func e7HostTorus(t *testing.T) geom.Torus {
	t.Helper()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 100, 100)
	if err != nil {
		t.Fatalf("E7 host torus: %v", err)
	}
	return tor
}

// TestTorusHostArmSurface_E7 pins the E7 latitude-cut torus arm to its derived spine circle: major
// M_a = 166.4395, centre (0,0,60.7107), minor r — an exact closed form of {r, torus, cap} (a wrong η sign
// or branch moves the major by tens of units). This is the "latitude accept" leg of the regression.
func TestTorusHostArmSurface_E7(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	host := e7HostTorus(t)
	tor, reason := torusHostArmSurface(host, e7CapZ, e7Rho, torusArmR, +1, res)
	if reason != torusArmBuilt {
		t.Fatalf("E7: torusHostArmSurface declined a valid convex latitude arm (reason %d)", reason)
	}
	wantMajor := 100 + sqrt(90*90-(e7CapZ-torusArmR)*(e7CapZ-torusArmR)) // R + √(b²−η²)
	if stdmath.Abs(tor.MajorRadius-wantMajor) > torusArmExactTol {
		t.Fatalf("E7 arm major = %.12f, want %.12f (Δ %.3g)", tor.MajorRadius, wantMajor, tor.MajorRadius-wantMajor)
	}
	if stdmath.Abs(tor.MinorRadius-torusArmR) > torusArmExactTol {
		t.Fatalf("E7 arm minor = %.12f, want %g", tor.MinorRadius, torusArmR)
	}
	if d := float64(tor.Center.DistanceTo(math.P3(0, 0, e7CapZ-torusArmR))); d > torusArmExactTol {
		t.Fatalf("E7 arm centre = %v, want (0,0,%.6f) (off by %.3g)", tor.Center, e7CapZ-torusArmR, d)
	}
}

// TestTorusHostContactCircle_E7 pins the torus↔host-torus contact circle (the retrim rail on the torus
// host) to its derived centre z = (a/b)·η = 67.4563 and radius R + (a/b)·√(b²−η²) = 173.8217, built from
// the real arm — the (a/b)-scaled circle the E7 derivation gives (fillet_torusarm.go header).
func TestTorusHostContactCircle_E7(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	host := e7HostTorus(t)
	arm, reason := torusHostArmSurface(host, e7CapZ, e7Rho, torusArmR, +1, res)
	if reason != torusArmBuilt {
		t.Fatalf("E7: arm build declined (reason %d)", reason)
	}
	center, radius, ok := torusHostContactCircle(host, arm, res)
	if !ok {
		t.Fatal("E7: torusHostContactCircle declined a coaxial internally-tangent arm")
	}
	k := 100.0 / 90.0
	eta := e7CapZ - torusArmR
	wantZ := k * eta
	wantRad := 100 + k*(arm.MajorRadius-100)
	if d := float64(center.DistanceTo(math.P3(0, 0, wantZ))); d > torusArmExactTol {
		t.Fatalf("E7 contact centre = %v, want (0,0,%.6f) (off by %.3g)", center, wantZ, d)
	}
	if stdmath.Abs(radius-wantRad) > torusArmExactTol {
		t.Fatalf("E7 contact radius = %.12f, want %.12f", radius, wantRad)
	}
}

// TestTorusLatitudeCut gates the accept/reject boundary: a cap ⊥ the axis (|ẑ·n̂|=1, E7) is admitted; a
// meridian cap ∥ the axis (|ẑ·n̂|=0, the spiric E5/E9) is rejected — the gate that keeps E5/E9 flooring.
func TestTorusLatitudeCut(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	if !torusLatitudeCut(1.0, e7Rho, res) {
		t.Fatal("latitude cap (|ẑ·n̂|=1) must be admitted")
	}
	if !torusLatitudeCut(-1.0, e7Rho, res) {
		t.Fatal("latitude cap with a Reversed outward normal (|ẑ·n̂|=1) must be admitted")
	}
	if torusLatitudeCut(0.0, e7Rho, res) {
		t.Fatal("meridian cap (|ẑ·n̂|=0, spiric E5/E9) must be rejected")
	}
}

// TestTorusHostArmSurface_Rejects covers the spindle (r ≥ a → b ≤ 0) and clearance (cap clears the offset
// torus → d² ≤ 0) declines, each an honest do-no-harm reject rather than a wrong arm.
func TestTorusHostArmSurface_Rejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	host := e7HostTorus(t)
	if _, reason := torusHostArmSurface(host, e7CapZ, e7Rho, 100, +1, res); reason != torusArmSpindle {
		t.Fatalf("r=a=100 must reject as spindle, got reason %d", reason)
	}
	// A cap far above the tube: η = 95 − 0 = 95 > b = 90 ⇒ d² = 90²−95² < 0 ⇒ the cap clears the offset torus.
	if _, reason := torusHostArmSurface(host, 95, e7Rho, torusArmR, 0, res); reason != torusArmClears {
		t.Fatalf("a cap clearing the offset torus (d² ≤ 0) must reject as clears, got reason %d", reason)
	}
}
