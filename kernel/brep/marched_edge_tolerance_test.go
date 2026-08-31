// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// crossingCylinderPair is the corpus's "crossing cylinders" fixture: an r=3 cylinder up the z axis and an
// r=1.5 rod across it on x. Their intersection is a quartic with NO closed form, so the imprint is marched.
func crossingCylinderPair(t *testing.T) (fat, rod *topo.Body) {
	t.Helper()
	fat, err := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("fat cylinder: %v", err)
	}
	rod, err = SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatalf("rod cylinder: %v", err)
	}
	return fat, rod
}

// TestMarchedIntersectBodyReportsAchievedTolerance: the crossing-cylinder INTERSECT is stitched from a
// MARCHED imprint, so the result body must be able to say how exact its boundary is. Before #3489 it
// reported nothing and every consumer — closure post-conditions, mass properties — had to assume the
// boundary was exact, which is precisely why those bodies miss their exact volume by ~1e-4.
//
// The assertion is on the ORDER, not on a frozen constant: the marched imprint's chord bow is bounded
// below by round-off and above by the rod-circle sagitta of a coarse march, and it must be the same
// number every edge stitched from that imprint carries.
func TestMarchedIntersectBodyReportsAchievedTolerance(t *testing.T) {
	fat, rod := crossingCylinderPair(t)
	res, ok := RuledCrossingIntersectGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("crossing cylinders ∩ declined; the general marched path is what this test measures")
	}
	tol := res.AchievedBoundaryTolerance()
	if tol <= 0 {
		t.Fatalf("a body stitched from a marched imprint reports AchievedBoundaryTolerance %g; a chord approximation is never exact", tol)
	}
	// A 16-chord march of the r=1.5 rod circle bows by r(1−cos(π/16)) ≈ 1.4e-2; the tracer is far finer
	// than that, so the achieved tolerance must sit well below it while staying above float round-off.
	coarseBow := 1.5 * (1 - stdmath.Cos(stdmath.Pi/16))
	if tol > coarseBow {
		t.Errorf("AchievedBoundaryTolerance %.6g exceeds a 16-chord march's bow %.6g: the imprint is coarser than any usable march", tol, coarseBow)
	}
	assertMarchedEdgesShareTolerance(t, res, tol)
}

// assertMarchedEdgesShareTolerance checks that every inexact edge of the body carries the SAME achieved
// tolerance — the one the imprint trace measured — and that at least one edge does. Two different
// readings on one imprint would mean the residual was measured per edge instead of threaded from the
// trace that produced them.
func assertMarchedEdgesShareTolerance(t *testing.T, body *topo.Body, want float64) {
	t.Helper()
	marched := 0
	for _, e := range body.Edges() {
		if e.Tolerance() == 0 {
			continue
		}
		marched++
		if e.Tolerance() != want {
			t.Errorf("edge %d reports tolerance %.6g, want the imprint's %.6g shared by every marched edge", e.ID(), e.Tolerance(), want)
		}
	}
	if marched == 0 {
		t.Error("no edge of a marched-imprint body carries an achieved tolerance; the residual did not reach the edges")
	}
}

// TestAnalyticBodyReportsZeroAchievedTolerance: a bare cylinder is built entirely from analytic circles
// and line segments, so its boundary IS exact and it must report 0. This is the control that keeps a
// non-zero reading meaningful.
func TestAnalyticBodyReportsZeroAchievedTolerance(t *testing.T) {
	fat, _ := crossingCylinderPair(t)
	if tol := fat.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("an analytic cylinder reports AchievedBoundaryTolerance %g, want 0", tol)
	}
}
