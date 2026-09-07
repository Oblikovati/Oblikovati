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
	res, err := Boolean(Intersection, fat, rod)
	if err != nil {
		t.Fatalf("crossing cylinders ∩: %v", err)
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("a body stitched from the exact ruled∩quadric section reports AchievedBoundaryTolerance %g, want 0", tol)
	}
}

// TestRimCrossingCutIsExactToo: the oblique rod whose exit ellipse CROSSES the target's top rim used to
// be the corpus's one marched body. Its bespoke driver clipped the section loop open at the rim, the
// closed-form path refuses a clipped chain by contract (exactImprintLoops), and the imprint therefore
// marched — so this test asserted a POSITIVE AchievedBoundaryTolerance and checked the residual reached
// every edge. Deleting the driver (ADR-0061 stage 4) put the pair through the general per-face pipeline,
// which meets each face's own section in closed form: the wall's ruled∩quadric arc, the cap's ellipse,
// the rim's circle. Every edge of the result is an analytic curve, so the body is exact and must say 0.
//
// The tolerance machinery itself is not left uncovered — kernel/topo's achieved_tolerance_test.go pins
// how an edge inherits a curve's deviation and how the body reports the worst of them. What is gone is a
// boolean in this corpus that produces an inexact one.
func TestRimCrossingCutIsExactToo(t *testing.T) {
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
	res, err := Boolean(Difference, target, rod)
	if err != nil {
		t.Fatalf("rim-crossing cut: %v", err)
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("the rim-crossing cut reports AchievedBoundaryTolerance %g, want 0 — every section is a closed form", tol)
	}
	for _, e := range res.Edges() {
		if e.Tolerance() != 0 {
			t.Errorf("edge %d carries tolerance %.6g on a curve of %T; the general pipeline meets these faces exactly",
				e.ID(), e.Tolerance(), e.Geometry())
		}
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
