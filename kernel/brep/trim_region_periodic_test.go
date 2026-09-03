// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A torus half is bounded by two rims that wrap the AZIMUTH, on a surface whose v is itself a period.
// Both of the two bands those rims bound are admissible regions, and the machinery that reads a band
// picked the same one every time — so one half of a torus read as the other (ADR-0062).

// torusHalfRegion is the trim region of a torus half: two azimuth-wrapping rims at v=0 and v=π, wound
// so the material is the band from v=π round through the seam to v=2π.
func torusHalfRegion(t *testing.T, materialAbovePi bool) trimRegion {
	t.Helper()
	rim := func(v float64, forward bool) []math.Point2 {
		const n = 32
		out := make([]math.Point2, 0, n+1)
		for i := 0; i <= n; i++ {
			u := 2 * stdmath.Pi * float64(i) / n
			if !forward {
				u = 2*stdmath.Pi - u
			}
			out = append(out, math.P2(math.Scalar(u), math.Scalar(v)))
		}
		return out
	}
	// Material on the LEFT of the traversal: a rim run in +u carries it above, one run in −u below.
	// For the band above π: the rim at π runs +u (material above it), the rim at 0≡2π runs −u.
	return trimRegion{
		rings:     [][]math.Point2{rim(stdmath.Pi, materialAbovePi), rim(0, !materialAbovePi)},
		uPeriodic: true, vPeriodic: true,
	}
}

// TestPeriodicBandReadsTheHalfItsRimsWind: the crossing count an upward v-ray gives is meaningless when
// v is itself a period — every point has a rim above it, and how many depends only on where the period
// was cut. The rims' own winding says which band is the material.
func TestPeriodicBandReadsTheHalfItsRimsWind(t *testing.T) {
	t.Parallel()
	below := math.P2(1, stdmath.Pi/2)   // v between 0 and π
	above := math.P2(1, 3*stdmath.Pi/2) // v between π and 2π
	upper := torusHalfRegion(t, true)   // material from π round to 2π
	lower := torusHalfRegion(t, false)  // material from 0 to π
	if !upper.contains(above) || upper.contains(below) {
		t.Errorf("the band wound above π holds v=3π/2 (%v) and not v=π/2 (%v)",
			upper.contains(above), upper.contains(below))
	}
	if !lower.contains(below) || lower.contains(above) {
		t.Errorf("the band wound below π holds v=π/2 (%v) and not v=3π/2 (%v)",
			lower.contains(below), lower.contains(above))
	}
}

// TestFluxDomainTakesTheWindowHoldingTheMaterial: the rims' own bounding box is one of the two windows
// they bound and always the same one. A band whose material lies through the SEAM must get the other,
// or every quadrature and probe point over it samples a region containing none of it.
func TestFluxDomainTakesTheWindowHoldingTheMaterial(t *testing.T) {
	t.Parallel()
	r := torusHalfRegion(t, true) // material from π round through the seam to 2π
	_, _, v0, v1, ok := periodicWindowHoldingMaterial(r, 0, 2*stdmath.Pi, 0, stdmath.Pi)
	if !ok {
		t.Fatal("no window was found to hold the material")
	}
	if stdmath.Abs(v0-stdmath.Pi) > 1e-9 || stdmath.Abs(v1-2*stdmath.Pi) > 1e-9 {
		t.Errorf("window v ∈ [%g, %g], want [π, 2π] — the band through the seam", v0, v1)
	}
	// The band that does NOT cross the seam keeps the rims' own box.
	s := torusHalfRegion(t, false)
	_, _, w0, w1, ok := periodicWindowHoldingMaterial(s, 0, 2*stdmath.Pi, 0, stdmath.Pi)
	if !ok || w0 != 0 || stdmath.Abs(w1-stdmath.Pi) > 1e-9 {
		t.Errorf("window v ∈ [%g, %g] ok=%v, want [0, π] unchanged", w0, w1, ok)
	}
}

// TestFluxDomainDeclinesNoWindowForAnIsolineRing: a ring that runs along ONE isoline — a sphere's
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
	// The equator: constant latitude, so zero extent in v.
	const n = 32
	ring := make([]math.Point2, 0, n+1)
	for i := 0; i <= n; i++ {
		ring = append(ring, math.P2(math.Scalar(2*stdmath.Pi*float64(i)/n), 0))
	}
	r := trimRegion{rings: [][]math.Point2{ring}, uPeriodic: true}
	f := curvedFace{surface: sph}
	_, _, v0, v1, ok := fluxDomain(f, r)
	if !ok {
		t.Fatal("an isoline-ringed face got no window at all")
	}
	if v1-v0 < stdmath.Pi-1e-9 {
		t.Errorf("window v ∈ [%g, %g] spans %g, want the sphere's whole latitude range (π)", v0, v1, v1-v0)
	}
}
