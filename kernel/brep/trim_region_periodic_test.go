// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// What a face's rings do and do NOT determine on a periodic surface (ADR-0063).
//
// This file used to assert the four rules trim_region.go carried for reading open, period-turning
// polylines — an upward ray for an azimuth rim, the nearest rim above for a tube-turning ring, which of
// two windows holds the material, and the outerless complement. They are gone, with the rules. Both of
// the bands two rims bound really are admissible regions, and no reading of the rims alone can say
// which the face is; the producer records it (face_chart.go), and the end-to-end gate is
// TestFaceChartCoversTheKeptSide, which judges the carried chart against the boolean's own keep rule.
//
// What remains here is the boundary of what a derivation may claim: a band on an axis that is not
// itself periodic has ONE strip between its rims and is derivable; a band on a doubly-periodic surface
// has two and is refused.

// wrappingRim is one period-turning rim at constant across-coordinate, sampled one step short of its
// full turn the way loopToUV samples a rim.
func wrappingRim(across float64, forward, alongU bool) []math.Point2 {
	const n = 32
	out := make([]math.Point2, 0, n)
	for i := 0; i < n; i++ {
		t := twoPi * float64(i) / n
		if !forward {
			t = twoPi - t
		}
		if alongU {
			out = append(out, math.P2(t, across))
			continue
		}
		out = append(out, math.P2(across, t))
	}
	return out
}

// TestDerivedBandChartHoldsTheStripBetweenItsRims: a cylinder's v is not a period, so its two rims bound
// exactly one strip and the seam that closes them is determined. The derivation supplies it.
func TestDerivedBandChartHoldsTheStripBetweenItsRims(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	rings := [][]math.Point2{wrappingRim(2, true, true), wrappingRim(0, false, true)}
	contours, ok := closeRingsIntoChart(cyl, rings, false, true, false)
	if !ok {
		t.Fatal("a two-rim cylinder band was refused, but its strip is determined")
	}
	r := trimRegion{contours: contours, decided: true, uPeriodic: true}
	for _, u := range []float64{0, 1, 3, 6.2, 6.28} { // 6.2 lies in the rims' unsampled last step
		if !r.contains(math.P2(u, 1)) {
			t.Errorf("(%g, 1) between the rims reads outside", u)
		}
		if r.contains(math.P2(u, 2.5)) || r.contains(math.P2(u, -0.5)) {
			t.Errorf("(%g, ±) beyond a rim reads inside", u)
		}
	}
}

// TestDoublyPeriodicBandIsRefusedNotGuessed: a torus half is bounded by two rims that turn the azimuth
// on a surface whose v is ITSELF a period, so both bands they bound are admissible and the rims say
// nothing. The derivation refuses instead of picking, which is what the deleted rules did — and picked
// the same one every time, so one half of a torus read as the other (ADR-0062).
func TestDoublyPeriodicBandIsRefusedNotGuessed(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	rings := [][]math.Point2{wrappingRim(stdmath.Pi, true, true), wrappingRim(0, false, true)}
	if _, ok := closeRingsIntoChart(tor, rings, false, true, true); ok {
		t.Error("two azimuth rims on a torus were read as one band, but they bound two")
	}
	// The same is true of the ovals a plane parallel to the axis cuts, which turn the TUBE instead.
	ovals := [][]math.Point2{wrappingRim(4, true, false), wrappingRim(1, false, false)}
	if _, ok := closeRingsIntoChart(tor, ovals, false, true, true); ok {
		t.Error("two tube-turning ovals were read as one band, but they bound two")
	}
}

// TestOuterlessFaceIsFramedByItsSurfaceDomain: a face whose loops are ALL holes wraps its closed surface
// minus them, and its outer contour in the chart is the parameter rectangle — a full turn on a periodic
// axis, the surface's own domain on a bounded one. A sphere's latitude is the bounded case.
func TestOuterlessFaceIsFramedByItsSurfaceDomain(t *testing.T) {
	t.Parallel()
	sph, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	hole := [][]math.Point2{{math.P2(2.9, 0.1), math.P2(3.1, 0.1), math.P2(3.1, 0.3), math.P2(2.9, 0.3)}}
	contours, ok := closeRingsIntoChart(sph, hole, true, true, false)
	if !ok {
		t.Fatal("an outerless sphere face got no frame")
	}
	r := trimRegion{contours: contours, decided: true, uPeriodic: true}
	if r.contains(math.P2(3.0, 0.2)) {
		t.Error("a point inside the hole reads as on the face")
	}
	if !r.contains(math.P2(1.0, -0.9)) {
		t.Error("a point far from the hole reads as off the face, but the face is everything else")
	}
}

// TestFluxDomainTakesTheSurfaceDomainForAnIsolineRing: a ring that runs along ONE isoline — a sphere's
// equator, a band's rim — has no extent across it, so its bounding box is a zero-height rectangle.
// Reading that box as the quadrature window measured nothing, and the shell was left uncertified: a
// hemisphere's outward sense then came from its loop winding rather than from its geometry, and the
// boolean is free to wind that loop either way (ADR-0062). The surface's own domain is the window.
func TestFluxDomainTakesTheSurfaceDomainForAnIsolineRing(t *testing.T) {
	t.Parallel()
	sph, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	r := trimRegion{contours: [][]math.Point2{wrappingRim(0, true, true)}, decided: true, uPeriodic: true}
	_, _, v0, v1, ok := fluxDomain(curvedFace{surface: sph}, r)
	if !ok {
		t.Fatal("an isoline-ringed face got no window at all")
	}
	if v1-v0 < stdmath.Pi-1e-9 {
		t.Errorf("window v ∈ [%g, %g] spans %g, want the sphere's whole latitude range (π)", v0, v1, v1-v0)
	}
}
