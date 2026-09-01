// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Region-correctness coverage for the torus perpendicular-band cut (Oblikovati/Oblikovati#1375). The
// brep-package tests assert the band stays watertight and analytic; these assert the RIGHT region
// survived, the gap #1497 flags (keep-below and keep-above asserted identical things, so a normal-
// ignoring cut would have passed). Tessellation runs ~1.3–1.6% under analytic here, hence 3% tol.

const (
	torusR = 5.0 // major radius
	torusr = 2.0 // minor (tube) radius
)

// torusBelowVolume is the analytic volume of a solid torus (major R, minor r, axis z, centred at the
// origin) lying at z ≤ c for −r ≤ c ≤ r. The cross-section at height z is an annulus of area
// 4πR·√(r²−z²), so the volume is 4πR·∫_{-r}^{c} √(r²−z²) dz. At c=0 this is π²Rr² (exactly half).
func torusBelowVolume(major, minor, c float64) float64 {
	antideriv := func(z float64) float64 {
		return z/2*stdmath.Sqrt(minor*minor-z*z) + minor*minor/2*stdmath.Asin(z/minor)
	}
	return 4 * stdmath.Pi * major * (antideriv(c) - antideriv(-minor))
}

// TestHalfSpaceCutTorusPerpendicularBandVolume checks both the symmetric mid-plane halves and an
// off-centre slab. The two mid-plane halves have EQUAL volume (so volume alone cannot tell them
// apart — the centroid-side check is what distinguishes keep-below from keep-above); the off-centre
// plane yields a distinct analytic slab volume.
func TestHalfSpaceCutTorusPerpendicularBandVolume(t *testing.T) {
	t.Parallel()
	halfVol := torusBelowVolume(torusR, torusr, 0) // = π²Rr², half the torus
	for _, tc := range []struct {
		name   string
		origin math.Point3
		normal math.Vector3
		want   float64
	}{
		{"keep below mid-plane", math.P3(0, 0, 0), math.V3(0, 0, 1), halfVol},
		{"keep above mid-plane", math.P3(0, 0, 0), math.V3(0, 0, -1), halfVol},
		{"keep below off-centre", math.P3(0, 0, 1), math.V3(0, 0, 1), torusBelowVolume(torusR, torusr, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tor, _ := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), torusR, torusr, "torus")
			plane, _ := geom.NewPlane(tc.origin, tc.normal)
			res, err := brep.HalfSpaceCut(tor, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			assertKeptVolume(t, res, tc.want, 0.03, tc.name)
			assertCentroidKeptSide(t, res, tc.origin, tc.normal, tc.name)
		})
	}
}

// TestHalfSpaceCutTorusMidPlaneHalvesAreComplementary cuts the same mid-plane both ways and checks the
// two halves partition the whole torus (volumes sum to the full torus) and are individually centred on
// opposite sides. A cut that ignored the plane normal would return the same region twice and fail the
// opposite-side requirement.
func TestHalfSpaceCutTorusMidPlaneHalvesAreComplementary(t *testing.T) {
	t.Parallel()
	tor, _ := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), torusR, torusr, "torus")
	full := query.BodyGeometryProperties(tor, ops.DefaultQuality()).Volume

	below, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))  // keep z ≤ 0
	above, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, -1)) // keep z ≥ 0
	lo, err := brep.HalfSpaceCut(tor, below)
	if err != nil {
		t.Fatalf("HalfSpaceCut(below): %v", err)
	}
	hi, err := brep.HalfSpaceCut(tor, above)
	if err != nil {
		t.Fatalf("HalfSpaceCut(above): %v", err)
	}
	vlo := query.BodyGeometryProperties(lo, ops.DefaultQuality()).Volume
	vhi := query.BodyGeometryProperties(hi, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(vlo+vhi-full) / full; rel > 0.01 {
		t.Errorf("torus halves sum to %.4f, want %.4f (full) — rel %.4f > 1%% (not a partition)", vlo+vhi, full, rel)
	}
	// Centroids must straddle the plane: below half centred at −8r/3π, above at +8r/3π (mirror images).
	assertCentroidKeptSide(t, lo, math.P3(0, 0, 0), math.V3(0, 0, 1), "below half")
	assertCentroidKeptSide(t, hi, math.P3(0, 0, 0), math.V3(0, 0, -1), "above half")
}

// TestHalfSpaceCutTorusClearsKeepsFullVolume strengthens the brep-package face-count check: a plane
// clear of the tube must keep the WHOLE torus volume, not merely the same face count (a sliver shave
// preserves face count but loses volume).
func TestHalfSpaceCutTorusClearsKeepsFullVolume(t *testing.T) {
	t.Parallel()
	tor, _ := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), torusR, torusr, "torus")
	full := query.BodyGeometryProperties(tor, ops.DefaultQuality()).Volume
	clear, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1)) // z ≤ 3 holds the whole tube (to z=2)
	res, err := brep.HalfSpaceCut(tor, clear)
	if err != nil {
		t.Fatalf("HalfSpaceCut(clears): %v", err)
	}
	assertKeptVolume(t, res, full, 1e-6, "clearing plane keeps whole torus")
}
