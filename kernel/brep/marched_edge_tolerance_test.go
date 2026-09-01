// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// crossingCylinderPair is the corpus's "crossing cylinders" fixture: an r=3 cylinder up the z axis and an
// r=1.5 rod across it on x. Its section is EXACT now — a ruled∩quadric closed form (#3489) — which is
// why the marched-tolerance contract below is pinned on the rim-crossing pair instead.
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

// TestExactCrossingIntersectReportsZeroTolerance: the crossing-cylinder section is a ruled∩quadric
// CLOSED FORM (#3489) — along each straight ruling the other cylinder is a quadratic, and its roots
// are the curve — so the stitched body's boundary IS exact and must say so. This test used to assert
// the opposite on the same fixture, back when the imprint was marched; the fixture is kept precisely
// so the contract flip is recorded where it happened.
func TestExactCrossingIntersectReportsZeroTolerance(t *testing.T) {
	t.Parallel()
	fat, rod := crossingCylinderPair(t)
	res, ok := RuledCrossingIntersectGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("crossing cylinders ∩ declined")
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("a body stitched from the exact ruled∩quadric section reports AchievedBoundaryTolerance %g, want 0", tol)
	}
}

// TestMarchedCutBodyReportsAchievedTolerance: a body stitched from a MARCHED imprint must be able to
// say how exact its boundary is — before #3489 it reported nothing, and every consumer had to assume
// the boundary was exact, which is why those bodies missed their exact volume by ~1e-4. The fixture is
// the rim-crossing cut: the oblique rod's exit ellipse crosses the top rim, so the marching window
// cuts the section loop OPEN, the closed-form path refuses a clipped chain by contract
// (exactImprintLoops), and this imprint genuinely marches — the crossing pair no longer does.
//
// The assertion is on the ORDER, not on a frozen constant: the marched imprint's chord bow is bounded
// below by round-off and above by the rod-circle sagitta of a coarse march, and it must be the same
// number every edge stitched from that imprint carries.
func TestMarchedCutBodyReportsAchievedTolerance(t *testing.T) {
	t.Parallel()
	target, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	s := math.Scalar(1 / stdmath.Sqrt2)
	rod, err := SolidCylinder(math.P3(-5.6, 0, 2), math.V3(s, 0, s), 0.9, 16)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	res, ok := RimCrossingCutGeneral(target, rod, nil)
	if !ok {
		t.Fatal("rim-crossing cut declined; the marched path is what this test measures")
	}
	tol := res.AchievedBoundaryTolerance()
	if tol <= 0 {
		t.Fatalf("a body stitched from a marched imprint reports AchievedBoundaryTolerance %g; a chord approximation is never exact", tol)
	}
	// A 16-chord march of the r=3 wall circle bows by R(1−cos(π/16)) ≈ 5.8e-2; the tracer is far finer
	// than that, so the achieved tolerance must sit well below it while staying above float round-off.
	// (The wall, not the rod: the chains live on the target wall's chart, so its circle sets the bow.)
	coarseBow := 3 * (1 - stdmath.Cos(stdmath.Pi/16))
	if tol > coarseBow {
		t.Errorf("AchievedBoundaryTolerance %.6g exceeds a 16-chord march's bow %.6g: the imprint is coarser than any usable march", tol, coarseBow)
	}
	assertMarchedEdgesShareTolerance(t, res, tol)
}

// assertMarchedEdgesShareTolerance checks that every inexact edge carries a POSITIVE achieved
// tolerance no worse than the body's own reading, that the body's reading IS the worst edge's, and
// that at least one edge is marched. A body may hold chains from more than one trace (the
// rim-crossing cut stitches a closed entry loop and a corner-snapped exit chain, each with its own
// measured deviation), so edges need not share one number — but none may exceed what the body
// reports, and the body must not report more than any edge carries.
func assertMarchedEdgesShareTolerance(t *testing.T, body *topo.Body, want float64) {
	t.Helper()
	marched, worst := 0, 0.0
	for _, e := range body.Edges() {
		if e.Tolerance() == 0 {
			continue
		}
		marched++
		worst = stdmath.Max(worst, e.Tolerance())
		if e.Tolerance() > want {
			t.Errorf("edge %d reports tolerance %.6g, above the body's AchievedBoundaryTolerance %.6g", e.ID(), e.Tolerance(), want)
		}
	}
	if marched == 0 {
		t.Error("no edge of a marched-imprint body carries an achieved tolerance; the residual did not reach the edges")
	}
	if worst != want {
		t.Errorf("the body reports %.6g but its worst edge carries %.6g; the reading must be the measured worst", want, worst)
	}
}

// TestAnalyticBodyReportsZeroAchievedTolerance: a bare cylinder is built entirely from analytic circles
// and line segments, so its boundary IS exact and it must report 0. This is the control that keeps a
// non-zero reading meaningful.
func TestAnalyticBodyReportsZeroAchievedTolerance(t *testing.T) {
	t.Parallel()
	fat, _ := crossingCylinderPair(t)
	if tol := fat.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("an analytic cylinder reports AchievedBoundaryTolerance %g, want 0", tol)
	}
}
