// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// crossingCylinderWindowArea is the area one perpendicular drill of radius r removes from the wall
// of a cylinder of radius bigR, as a closed-form integrand solved by 1-D Simpson quadrature.
//
// A wall point is (bigR·cosθ, bigR·sinθ, z); it lies in the drill (axis x, radius r) when
// bigR²sin²θ + z² ≤ r², so the window spans z = ±√(r² − bigR²sin²θ) over |sinθ| ≤ r/bigR and its
// area is ∮ 2·bigR·√(r² − bigR²sin²θ) dθ. This is an INDEPENDENT oracle: a one-dimensional
// quadrature of a formula, where the code under test runs a two-dimensional Green integral over
// sampled loops.
func crossingCylinderWindowArea(bigR, r float64) float64 {
	const steps = 200000 // even, for Simpson
	limit := math.Asin(r / bigR)
	h := 2 * limit / steps
	sum := 0.0
	for i := 0; i <= steps; i++ {
		theta := -limit + float64(i)*h
		v := 2 * bigR * math.Sqrt(math.Max(r*r-bigR*bigR*math.Sin(theta)*math.Sin(theta), 0))
		switch {
		case i == 0 || i == steps:
			sum += v
		case i%2 == 1:
			sum += 4 * v
		default:
			sum += 2 * v
		}
	}
	return sum * h / 3
}

// TestDrilledWallAreaSubtractsItsWindows pins the bug where a face's holes were ADDED to its region
// instead of subtracted (Oblikovati/Oblikovati#3489).
//
// Every loop is unwrapped from its own first sample, so the drilled wall's outer loop came back on
// u ∈ [−2π, 0] and its two windows on [0, 2π). The nesting test compared them across branches, read
// each window as OUTSIDE the loop containing it, and gave both depth 0 — the windows counted as
// outer loops and the wall measured 240.81 where 211.57 is right, carrying the body's volume 9.8%
// over OpenCASCADE.
func TestDrilledWallAreaSubtractsItsWindows(t *testing.T) {
	t.Parallel()
	// The same operands the OCC corpus drills: r=3 h=12 through-cut by a perpendicular r=1.5.
	const bigR, drillR, height = 3.0, 1.5, 12.0
	fat, drill := crossingCylinders(t)
	res, err := ops.Boolean(ops.Cut, fat, drill)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}

	wall := widestCylinderFace(res, bigR)
	if wall == nil {
		t.Fatalf("no face on the drilled cylinder of radius %g in the result", bigR)
	}
	got, ok := query.AnalyticFaceArea(wall)
	if !ok {
		t.Fatalf("the drilled wall declined analytic integration; it is a plain trimmed cylinder")
	}

	band := 2 * math.Pi * bigR * height
	want := band - 2*crossingCylinderWindowArea(bigR, drillR)
	if rel := math.Abs(got-want) / want; rel > 1e-3 {
		t.Errorf("drilled wall area = %.6f, want %.6f (rel %.3e); full band is %.6f, so a value ABOVE it means the windows were added rather than subtracted", got, want, rel, band)
	}
}

// widestCylinderFace returns the face lying on a cylinder of the given radius, or nil.
func widestCylinderFace(b *topo.Body, radius float64) *topo.Face {
	for _, f := range b.Faces() {
		cyl, isCyl := f.Geometry().(geom.Cylinder)
		if isCyl && math.Abs(cyl.Radius-radius) < 1e-9 {
			return f
		}
	}
	return nil
}
