// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The two additive (u,v)-core touches for the non-periodic planeUV operand (#1591, ADR-0049 D-c): the
// seamWelder u-fold gated on uPeriodic, and segPolygon's curve-identity sameRun. Unit-tested directly here
// so the mechanism is pinned before planeUV consumes it in Slice A.

// TestSeamWelderUFoldGatedOnUPeriodic: a u-periodic side folds u≈2π onto u=0 (the azimuth seam is one
// ruling); a NON-periodic plane must NOT — u is a real world distance, so a genuine face vertex near u=2π
// stays distinct from the origin.
func TestSeamWelderUFoldGatedOnUPeriodic(t *testing.T) {
	twoPi := math.Scalar(2 * stdmath.Pi)
	origin, seam := math.P2(0, 1), math.P2(twoPi, 1)

	periodic := newSeamWelder(true, false)
	if periodic.add(origin) != periodic.add(seam) {
		t.Error("u-periodic welder must fold u=2π onto u=0 (one azimuth ruling)")
	}

	plane := newSeamWelder(false, false)
	if plane.add(origin) == plane.add(seam) {
		t.Error("non-periodic (plane) welder must NOT fold u≈2π onto u=0 — that welds a real face vertex to the origin (#1591)")
	}
}

// TestSameRunSegPolygonByCurveIdentity: two polygon edges at the SAME v are the same run only if they are the
// same analytic curve — unlike a constant-v rim, where equal v alone means same run. This is why polygon
// edges need their own segKind (#1591).
func TestSameRunSegPolygonByCurveIdentity(t *testing.T) {
	edgeA := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	edgeB := geom.NewLineSegment(math.P3(1, 0, 0), math.P3(1, 1, 0))
	at := func(kind segKind, c geom.Curve3) recoveredEdge {
		return recoveredEdge{kind: kind, curve: c, a: math.P2(0, 5), b: math.P2(1, 5)} // same v=5
	}

	if !sameRun(at(segPolygon, edgeA), at(segPolygon, edgeA)) {
		t.Error("same polygon curve must be one run")
	}
	if sameRun(at(segPolygon, edgeA), at(segPolygon, edgeB)) {
		t.Error("two DISTINCT polygon curves at equal v must NOT merge into one run (the segRim constant-v trap)")
	}
	// Contrast: two rims at equal v DO merge — the behaviour segPolygon deliberately does not inherit.
	if !sameRun(at(segRim, edgeA), at(segRim, edgeB)) {
		t.Error("two rims at equal v are one run (constant-v rim invariant)")
	}
}
