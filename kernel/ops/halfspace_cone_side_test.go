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

// Region-correctness coverage for the frustum SIDE cut — a plane PARALLEL to the cone axis
// (Oblikovati/Oblikovati#1372 arc-band, #1374 annulus/tongue). The brep-package tests assert the cut
// stays watertight, analytic, and carries a hyperbola; these assert the right region survived. The
// complementary cuts (same plane, opposite normals) must partition the frustum, and each must be
// centred on its kept side — the gap #1497 flags, where annulus and tongue asserted nothing relating
// their volumes. Frustum: axis z∈[0,10], radius 3→6, so ρ(z) = 3 + 0.3z.

// frustumSideRemoved is the analytic volume of the slab of the frustum lying beyond |x| = d (the part a
// side plane at distance d removes), integrated as ∫_0^10 segmentArea(ρ(z), d) dz by Simpson's rule —
// an independent oracle for the kept volume (frustum − removed).
func frustumSideRemoved(d float64) float64 {
	seg := func(rad, m float64) float64 {
		if m >= rad {
			return 0
		}
		return rad*rad*stdmath.Acos(m/rad) - m*stdmath.Sqrt(rad*rad-m*m)
	}
	const n = 2000
	h := 10.0 / float64(n)
	sum := seg(3, d) + seg(3+0.3*10, d)
	for i := 1; i < n; i++ {
		z := float64(i) * h
		w := 2.0
		if i%2 == 1 {
			w = 4
		}
		sum += w * seg(3+0.3*z, d)
	}
	return sum * h / 3
}

// TestHalfSpaceCutConeSideArcBandVolume cuts the frustum by x=2 (∥ axis, |D|=2 < bottom r=3) keeping
// each side, and checks: each kept volume matches its analytic value (frustum ∓ removed), the two sides
// partition the frustum, and each centroid is on its kept side. The axis side is the major part.
func TestHalfSpaceCutConeSideArcBandVolume(t *testing.T) {
	frustum, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6, "frustum")
	full := ops.BodyGeometryProperties(frustum, ops.DefaultQuality()).Volume
	removed := frustumSideRemoved(2)

	axisOrigin, axisNormal := math.P3(2, 0, 0), math.V3(1, 0, 0) // keep x ≤ 2 (axis side, major part)
	farOrigin, farNormal := math.P3(2, 0, 0), math.V3(-1, 0, 0)  // keep x ≥ 2 (far side, the shaved flat)

	axisPlane, _ := geom.NewPlane(axisOrigin, axisNormal)
	farPlane, _ := geom.NewPlane(farOrigin, farNormal)
	axisSide, err := brep.HalfSpaceCut(frustum, axisPlane)
	if err != nil {
		t.Fatalf("HalfSpaceCut(axis): %v", err)
	}
	farSide, err := brep.HalfSpaceCut(frustum, farPlane)
	if err != nil {
		t.Fatalf("HalfSpaceCut(far): %v", err)
	}

	assertKeptVolume(t, axisSide, full-removed, 0.02, "arc-band axis side (x≤2)")
	assertKeptVolume(t, farSide, removed, 0.05, "arc-band far side (x≥2)") // thin slab → looser inscription tol
	assertCentroidKeptSide(t, axisSide, axisOrigin, axisNormal, "arc-band axis side")
	assertCentroidKeptSide(t, farSide, farOrigin, farNormal, "arc-band far side")
	assertComplementaryPartition(t, axisSide, farSide, full, "arc-band x=2")
}

// TestHalfSpaceCutConeSideAnnulusTongueComplementary cuts the frustum by x=4 (bottom r=3 < |D|=4 <
// top r=6) keeping each side: the axis side is the notched ANNULUS (#1374), the far side the TONGUE.
// They must partition the frustum, the annulus must dwarf the tongue, and each be on its kept side.
func TestHalfSpaceCutConeSideAnnulusTongueComplementary(t *testing.T) {
	frustum, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6, "frustum")
	full := ops.BodyGeometryProperties(frustum, ops.DefaultQuality()).Volume
	removed := frustumSideRemoved(4)

	annOrigin, annNormal := math.P3(4, 0, 0), math.V3(1, 0, 0)  // keep x ≤ 4 → annulus (apex kept)
	tonOrigin, tonNormal := math.P3(4, 0, 0), math.V3(-1, 0, 0) // keep x ≥ 4 → tongue (apex dropped)

	annPlane, _ := geom.NewPlane(annOrigin, annNormal)
	tonPlane, _ := geom.NewPlane(tonOrigin, tonNormal)
	annulus, err := brep.HalfSpaceCut(frustum, annPlane)
	if err != nil {
		t.Fatalf("HalfSpaceCut(annulus): %v", err)
	}
	tongue, err := brep.HalfSpaceCut(frustum, tonPlane)
	if err != nil {
		t.Fatalf("HalfSpaceCut(tongue): %v", err)
	}

	annVol := ops.BodyGeometryProperties(annulus, ops.DefaultQuality()).Volume
	tonVol := ops.BodyGeometryProperties(tongue, ops.DefaultQuality()).Volume
	assertKeptVolume(t, annulus, full-removed, 0.02, "annulus (x≤4)")
	assertKeptVolume(t, tongue, removed, 0.05, "tongue (x≥4)") // thin sliver → looser inscription tol
	if annVol <= tonVol {
		t.Errorf("annulus volume %.4f should exceed tongue volume %.4f (axis side is the major part)", annVol, tonVol)
	}
	assertCentroidKeptSide(t, annulus, annOrigin, annNormal, "annulus")
	assertCentroidKeptSide(t, tongue, tonOrigin, tonNormal, "tongue")
	assertComplementaryPartition(t, annulus, tongue, full, "annulus/tongue x=4")
}

// TestHalfSpaceCutConeSideClearsKeepsFullVolume strengthens the brep-package face-count check: a side
// plane clear of the whole frustum (|D| ≥ top radius) on the axis side keeps the WHOLE volume.
func TestHalfSpaceCutConeSideClearsKeepsFullVolume(t *testing.T) {
	frustum, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6, "frustum")
	full := ops.BodyGeometryProperties(frustum, ops.DefaultQuality()).Volume
	keep, _ := geom.NewPlane(math.P3(7, 0, 0), math.V3(1, 0, 0)) // x ≤ 7 ⊇ whole frustum (top r=6)
	res, err := brep.HalfSpaceCut(frustum, keep)
	if err != nil {
		t.Fatalf("HalfSpaceCut(keep): %v", err)
	}
	assertKeptVolume(t, res, full, 1e-6, "clearing side plane keeps whole frustum")
}

// assertComplementaryPartition fails unless two complementary cuts (same plane, opposite normals) sum
// to the whole volume and are individually non-empty (neither side swallowed the whole or nothing).
func assertComplementaryPartition(t *testing.T, a, b *topo.Body, full float64, label string) {
	t.Helper()
	va := ops.BodyGeometryProperties(a, ops.DefaultQuality()).Volume
	vb := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(va+vb-full) / full; rel > 0.01 {
		t.Errorf("%s: complementary cuts sum to %.4f, want %.4f (full) — rel %.4f > 1%% (not a partition)",
			label, va+vb, full, rel)
	}
	if va <= 0 || vb <= 0 {
		t.Errorf("%s: a complementary cut is empty (va=%.4f vb=%.4f); both sides must be non-empty", label, va, vb)
	}
}
