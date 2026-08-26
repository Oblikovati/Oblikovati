// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// weJ3Spine is the J3 spiric frame in host coordinates (R=200 a=50, cap = the u=270° meridian plane
// x=0 with material-outward n̂=+x̂, rim-circle centre toward m̂=(0,−1,0), convex σ=−1, s=+1, r=10 →
// w=−10, b=40).
func weJ3Spine(t *testing.T) spiricRimSpine {
	t.Helper()
	host := weTestTorus(t, 200, 50)
	n, err := math.UnitVector3FromVector(math.V3(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	m, err := math.UnitVector3FromVector(math.V3(0, -1, 0))
	if err != nil {
		t.Fatal(err)
	}
	return spiricRimSpine{host: host, nHat: n, mHat: m, capD: 0, w: -10, b: 40, side: -1, r: 10}
}

// TestSpiricStationExactness sweeps the J3 spine and asserts every station's four closed-form
// tangency invariants: the centre rides the offset torus (distance b from the tube-centre circle),
// both feet sit exactly r from the centre, the tube foot is ON the host tube (distance a from the
// tube-centre circle), and the cap foot is ON the cap plane (n̂-coordinate = capD).
func TestSpiricStationExactness(t *testing.T) {
	sp := weJ3Spine(t)
	for k := range 64 {
		psi := 2 * stdmath.Pi * float64(k) / 64
		c, tubeFoot, capFoot, ok := sp.station(psi)
		if !ok {
			t.Fatalf("station ψ=%.3f declined", psi)
		}
		if d := stdmath.Abs(weTubeCentreDist(sp, c) - sp.b); d > 1e-9 {
			t.Fatalf("ψ=%.3f: centre off the offset tube by %.3g", psi, d)
		}
		if stdmath.Abs(float64(c.DistanceTo(tubeFoot))-sp.r) > 1e-9 {
			t.Fatalf("ψ=%.3f: tube foot at %.9f from the centre, want r=%g", psi, float64(c.DistanceTo(tubeFoot)), sp.r)
		}
		if stdmath.Abs(float64(c.DistanceTo(capFoot))-sp.r) > 1e-9 {
			t.Fatalf("ψ=%.3f: cap foot at %.9f from the centre, want r=%g", psi, float64(c.DistanceTo(capFoot)), sp.r)
		}
		if d := stdmath.Abs(weTubeCentreDist(sp, tubeFoot) - sp.host.MinorRadius); d > 1e-9 {
			t.Fatalf("ψ=%.3f: tube foot off the host tube by %.3g", psi, d)
		}
		if d := stdmath.Abs(float64(sp.host.Center.VectorTo(capFoot).Dot(sp.nHat.AsVector())) - sp.capD); d > 1e-9 {
			t.Fatalf("ψ=%.3f: cap foot off the cap plane by %.3g", psi, d)
		}
	}
}

// weTubeCentreDist is a point's distance from the host's tube-centre circle.
func weTubeCentreDist(sp spiricRimSpine, p math.Point3) float64 {
	z := sp.host.AxisDir.AsVector()
	d := sp.host.Center.VectorTo(p)
	axial := float64(d.Dot(z))
	rho := float64(d.Sub(z.Scale(math.Scalar(axial))).Length())
	return stdmath.Hypot(rho-sp.host.MajorRadius, axial)
}

// TestSpiricStationClearsInnerTurn guards the ξ² ≤ 0 decline: a spine whose offset plane exits the
// offset torus at the inner turn (|w| ≥ R − b) must decline at ψ=π rather than emit a NaN station.
func TestSpiricStationClearsInnerTurn(t *testing.T) {
	sp := weJ3Spine(t)
	sp.w = -(sp.host.MajorRadius - sp.b + 1) // one unit past the inner-turn clearance
	if _, _, _, ok := sp.station(stdmath.Pi); ok {
		t.Fatal("station at the inner turn with |w| > R−b must decline")
	}
}

// TestSpiricMeridianCut pins the meridian gate: a cap plane containing the axis (ẑ·n̂ = 0) is spiric;
// a latitude cap (ẑ·n̂ = 1) is not; the band is model-relative to the rim radius.
func TestSpiricMeridianCut(t *testing.T) {
	res := ResolutionForSize(500)
	if !spiricMeridianCut(0, 250, res) {
		t.Fatal("ẑ·n̂=0 must classify as a meridian cut")
	}
	if spiricMeridianCut(1, 250, res) {
		t.Fatal("ẑ·n̂=1 (latitude cap) must NOT classify as a meridian cut")
	}
	if spiricMeridianCut(0.05, 250, res) {
		t.Fatal("a 3° oblique cap is neither meridian nor latitude — must not classify as meridian")
	}
}
