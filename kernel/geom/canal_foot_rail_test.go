// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// footRailStations builds a synthetic surf-rst runout band: centres walking a parabola at height
// r above the z=0 host plane (so the B feet are exact plane projections at ball distance r) and
// A feet displaced horizontally by r (exact ball distance). Mirrors the shape contactLocusRail
// resolves on the setback corpus: 73 stations = 6 rail chords refined 12-fold.
func footRailStations(n int, r float64) (centers, feetA, feetB []math.Point3) {
	for i := 0; i < n; i++ {
		x := -3 + 6*float64(i)/float64(n-1)
		y := 0.1 * x * x
		c := math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(r))
		centers = append(centers, c)
		feetA = append(feetA, math.P3(math.Scalar(x), math.Scalar(y-r), math.Scalar(r)))
		feetB = append(feetB, math.P3(math.Scalar(x), math.Scalar(y), 0))
	}
	return centers, feetA, feetB
}

// TestCanalFootLocusRailIsTheLoftBoundaryIsoparm proves the rail IS the canal loft's foot-side
// boundary: CanalFootLocusRail from the same (centers, feetB) rows LoftCanalStations consumes must
// reproduce the surface's u=1 isoparm to evaluation noise. This is the property that makes a
// boundary edge carrying the rail lie ON the patch (the degree-1 node polyline it replaced sagged
// off it by the chord sagitta — S9 7.927e-04·diag, railb-locus-report.md).
func TestCanalFootLocusRailIsTheLoftBoundaryIsoparm(t *testing.T) {
	const r = 2.0
	centers, feetA, feetB := footRailStations(73, r)
	surf, err := LoftCanalStations(centers, feetA, feetB, r, 1e-9)
	if err != nil {
		t.Fatalf("LoftCanalStations: %v", err)
	}
	rail, err := CanalFootLocusRail(centers, feetB)
	if err != nil {
		t.Fatalf("CanalFootLocusRail: %v", err)
	}
	worst := 0.0
	for i := 0; i <= 100; i++ {
		v := float64(i) / 100
		if d := float64(rail.PointAt(v).DistanceTo(surf.PointAt(1, v))); d > worst {
			worst = d
		}
	}
	if worst > 1e-12 {
		t.Errorf("rail leaves the loft's u=1 boundary isoparm by %.3e (want <= 1e-12): "+
			"the rail is NOT the surface's own foot row", worst)
	}
}

// TestCanalFootLocusRailInterpolatesEveryContact proves the rail passes THROUGH every exact
// station contact at that station's own chord-length parameter — interpolation, never a smoothing
// fit (a scheme that smooths the solved contacts would silently trade the exact envelope for
// fairness).
func TestCanalFootLocusRailInterpolatesEveryContact(t *testing.T) {
	const r = 2.0
	centers, _, feetB := footRailStations(73, r)
	rail, err := CanalFootLocusRail(centers, feetB)
	if err != nil {
		t.Fatalf("CanalFootLocusRail: %v", err)
	}
	vParams, err := alphaParams(coords3(centers), 1)
	if err != nil {
		t.Fatalf("alphaParams: %v", err)
	}
	worst := 0.0
	for j, fb := range feetB {
		if d := float64(rail.PointAt(vParams[j]).DistanceTo(fb)); d > worst {
			worst = d
		}
	}
	if worst > 1e-9 {
		t.Errorf("rail misses a solved contact by %.3e (want <= 1e-9): the locus no longer "+
			"interpolates the exact tangency feet", worst)
	}
}

// TestCanalFootLocusRailStaysInTheContactPlane proves the interpolant of in-plane contacts stays
// in that plane (each control point is an affine combination of the contacts), so the pInner host
// plane's boundary consumer remains exact when the rail replaces the polyline.
func TestCanalFootLocusRailStaysInTheContactPlane(t *testing.T) {
	const r = 2.0
	centers, _, feetB := footRailStations(73, r)
	rail, err := CanalFootLocusRail(centers, feetB)
	if err != nil {
		t.Fatalf("CanalFootLocusRail: %v", err)
	}
	worst := 0.0
	for i := 0; i <= 100; i++ {
		if d := stdmath.Abs(float64(rail.PointAt(float64(i) / 100).Z)); d > worst {
			worst = d
		}
	}
	if worst > 1e-12 {
		t.Errorf("rail leaves the z=0 contact plane by %.3e (want <= 1e-12)", worst)
	}
}
