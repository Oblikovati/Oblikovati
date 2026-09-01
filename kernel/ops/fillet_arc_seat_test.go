// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// arcBallSeats must offer the four candidates in resolveRim's ladder order — the established convex-shaft
// derivation first, so a case that was already right is reproduced bit for bit before anything else is
// tried — and it must offer both sides of the cap.
func TestArcBallSeatsLadderOrder(t *testing.T) {
	t.Parallel()
	nOut := math.V3(0, 0, 1)
	got := arcBallSeats(nOut, 50, 10)
	if len(got) != 4 {
		t.Fatalf("arcBallSeats gave %d seats, want 4", len(got))
	}
	want := []arcBallSeat{
		{capSide: math.V3(0, 0, -1), majorR: 40}, {capSide: math.V3(0, 0, -1), majorR: 60},
		{capSide: math.V3(0, 0, 1), majorR: 40}, {capSide: math.V3(0, 0, 1), majorR: 60},
	}
	for i, w := range want {
		if got[i].majorR != w.majorR || got[i].capSide.Sub(w.capSide).Length() > 1e-12 {
			t.Errorf("seat %d = (side %v, majorR %g), want (side %v, majorR %g)",
				i, got[i].capSide, got[i].majorR, w.capSide, w.majorR)
		}
	}
}

// ★ THE MOVED REJECTION. `r >= cyl.Radius` used to be a blanket door check on every arc fillet. It is a
// CONVEX-tier fact — only the cylR−r seat needs r < cylR — so it now lives here as "a seat with a
// non-positive majorR is not a seat". A groove or a concave corner takes r ≥ cylR perfectly well at
// cylR+r, and must still be offered those two seats.
func TestArcBallSeatsDropOnlyTheNonPositiveMajorRadius(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ r, wantSeats float64 }{{5, 4}, {9.999, 4}, {10, 2}, {25, 2}} {
		got := arcBallSeats(math.V3(0, 0, 1), 10, tc.r)
		if float64(len(got)) != tc.wantSeats {
			t.Errorf("arcBallSeats(cylR=10, r=%g) gave %d seats, want %g", tc.r, len(got), tc.wantSeats)
		}
		for _, s := range got {
			if s.majorR <= 0 {
				t.Errorf("arcBallSeats(cylR=10, r=%g) offered a seat with majorR %g", tc.r, s.majorR)
			}
		}
	}
}

// checkArcFilletInputs rejects a non-positive radius and a cap plane not perpendicular to the axis — and
// no longer rejects r ≥ cylR, which is the concave tier's business.
func TestCheckArcFilletInputs(t *testing.T) {
	t.Parallel()
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	perp, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, 1))
	oblique, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(1, 0, 1))
	if err := checkArcFilletInputs(cyl, perp, 0); err == nil {
		t.Errorf("checkArcFilletInputs accepted r = 0")
	}
	if err := checkArcFilletInputs(cyl, oblique, 0.5); err == nil {
		t.Errorf("checkArcFilletInputs accepted a cap plane oblique to the axis")
	}
	if err := checkArcFilletInputs(cyl, perp, 5); err != nil {
		t.Errorf("checkArcFilletInputs rejected r=5 on cylR=1: %v — that is the concave tier's seat, not an "+
			"input error: %v", 5.0, err)
	}
}

// materialSideName names the side an unseatable ball was looked for on, for the decline message.
func TestMaterialSideName(t *testing.T) {
	t.Parallel()
	if got := materialSideName(true); got != "material" {
		t.Errorf("materialSideName(true) = %q, want \"material\"", got)
	}
	if got := materialSideName(false); got != "void" {
		t.Errorf("materialSideName(false) = %q, want \"void\"", got)
	}
}

// unwrapNear shifts an angle by whole turns into (ref−π, ref+π] — what lets a band wider than a half turn
// read its far end on the span the arc actually covers rather than the short way round.
func TestUnwrapNear(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ a, ref, want float64 }{
		{-1.5708, 3.9270, 4.7124}, // simple/H6's far end: atan2 says −π/2, the 270° span says 3π/2
		{0.1, 0.2, 0.1},
		{0.1 + 4*stdmath.Pi, 0.2, 0.1},
	} {
		if got := unwrapNear(tc.a, tc.ref); stdmath.Abs(got-tc.want) > 1e-4 {
			t.Errorf("unwrapNear(%g, %g) = %g, want %g", tc.a, tc.ref, got, tc.want)
		}
	}
}

// angleGap folds an azimuth difference into [0, π] so the section's branch pick is periodic-safe.
func TestAngleGap(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ a, b, want float64 }{
		{0, 0.5, 0.5}, {0, 2*stdmath.Pi - 0.5, 0.5}, {3, -3, 2*stdmath.Pi - 6}, {1, 1 + 4*stdmath.Pi, 0},
	} {
		if got := angleGap(tc.a, tc.b); stdmath.Abs(got-tc.want) > 1e-12 {
			t.Errorf("angleGap(%g, %g) = %g, want %g", tc.a, tc.b, got, tc.want)
		}
	}
}

// ★ THE RUN-OUT GENERALISES THE FLAT CROSS-SECTION. A side plane THROUGH the torus axis has K = C = 0, so
// w(v) ≡ 0 and u(v) ≡ Phi ± π/2 — a constant azimuth, i.e. the radial-plane tube arc. That is why a
// diametral wall and a stand-off wall are the same construction, and why adding the run-out could not move
// simple/B2 (whose two walls contain the axis) by a single bit.
func TestArcSideSectionThroughTheAxisIsAConstantAzimuth(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 60, 10)
	if err != nil {
		t.Fatal(err)
	}
	through, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0)) // contains the axis
	sec, ok := arcSideSection(tor, through, stdmath.Pi, 0)
	if !ok {
		t.Fatalf("arcSideSection declined a plane through the axis")
	}
	for i := 0; i <= 8; i++ {
		v := stdmath.Pi + (capTube-stdmath.Pi)*float64(i)/8
		if d := angleGap(sec.UAt(v), sec.UAt(stdmath.Pi)); d > 1e-12 {
			t.Errorf("section azimuth drifts %g at v=%g — a plane through the axis cuts a constant-u arc", d, v)
		}
	}
}

// A side plane that STANDS OFF the axis cuts a genuine run-out: u sweeps between the cyl-tangent and
// cap-tangent contacts, and every point of the section lies on BOTH the torus and that plane.
func TestArcSideSectionStandingOffTheAxisSweepsAndStaysOnBothSurfaces(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorusWithRef(math.P3(3, 0.2, 0.9999), math.V3(0, -1, 0), math.V3(-1, 0, 0), 1.2, 0.2)
	if err != nil {
		t.Fatal(err)
	}
	bottom, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // simple/W2's own bottom plane
	sec, ok := arcSideSection(tor, bottom, stdmath.Pi, 1.5567)
	if !ok {
		t.Fatalf("arcSideSection declined simple/W2's bottom plane")
	}
	if d := angleGap(sec.UAt(stdmath.Pi), sec.UAt(capTube)); d < 0.5 {
		t.Errorf("section azimuth sweeps only %g — a stand-off plane must cut a real run-out lobe", d)
	}
	for i := 0; i <= 16; i++ {
		p := sec.PointAt(float64(i) / 16)
		if stdmath.Abs(p.Z) > 1e-9 {
			t.Errorf("section point %v is %g off the plane z=0 it is supposed to lie in", p, p.Z)
		}
	}
	// The two endpoints are the closed forms DRAWEXE reports: the cyl-tangent circle and the cap-tangent
	// circle each meeting z=0 (3 − √(1²−0.9999²) = 2.985858 and 3 − √(1.2²−0.9999²) = 2.336524).
	assertPointX(t, sec.PointAt(0), 3-stdmath.Sqrt(1-0.9999*0.9999))
	assertPointX(t, sec.PointAt(1), 3-stdmath.Sqrt(1.44-0.9999*0.9999))
}

func assertPointX(t *testing.T, p math.Point3, want float64) {
	t.Helper()
	if stdmath.Abs(p.X-want) > 1e-6 {
		t.Errorf("section endpoint x = %.8g, closed form %.8g", p.X, want)
	}
}

// arcSectionOnTorus declines a plane that misses the torus over part of the band's tube range — a band
// cannot be terminated on a curve that does not exist along it, and SpiricArc clamps rather than erroring.
func TestArcSectionOnTorusDeclinesAPlaneThatMissesTheTube(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 60, 10)
	far, _ := geom.NewPlane(math.P3(0, 500, 0), math.V3(0, 1, 0))
	if _, ok := arcSideSection(tor, far, stdmath.Pi, 0); ok {
		t.Errorf("arcSideSection accepted a plane 500 away from a radius-70 torus")
	}
	perp, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // perpendicular to the axis: circles, not a section
	if _, ok := arcSideSection(tor, perp, stdmath.Pi, 0); ok {
		t.Errorf("arcSideSection accepted a plane perpendicular to the torus axis")
	}
}
