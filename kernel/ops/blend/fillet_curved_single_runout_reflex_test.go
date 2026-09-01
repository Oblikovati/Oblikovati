// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// reflexContactRail is the reflex single-arm runout's fix (C5/D8): a >180° picked rim's host contact rail
// must span the MAJOR sector of the contact circle, not the minor complement a three-point re-fit snaps to.
// These regressions pin that on a named synthetic rim fixture (a 270° reflex arc, a 90° convex arc, and the
// nil edge) rather than only through the whole-corpus area gate.

// reflexRimEdgeFixture is a NAMED fake picked-rim edge: a coaxial +z circular arc of the given signed sweep
// (its geometry is all reflexContactRail reads — its Normal and SweepAngle). rimRadius is immaterial to the
// rail (which lives on the SEPARATE contact circle) but is chosen distinct so a test never conflates them.
func reflexRimEdgeFixture(t *testing.T, sweep float64) *topo.Edge {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "reflexrim", 0))
	bld := topo.NewBuilder(false, lin)
	const rimRadius = 40.0
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), rimRadius, 0, sweep)
	if err != nil {
		t.Fatalf("reflexRimEdgeFixture: NewArc3d sweep %g: %v", sweep, err)
	}
	v0 := bld.AddVertex(arc.PointAt(0), lin)
	v1 := bld.AddVertex(arc.PointAt(1), lin)
	return bld.AddEdge(arc, v0, v1, lin)
}

// contactCircleFeet returns the two rail feet on a contact circle (centre origin, +z axis, given radius) at
// azimuths 0 and `sweep` — the coaxial images of the rim's two end vertices, the endpoints reflexContactRail
// must join the MAJOR way.
func contactCircleFeet(radius, sweep float64) (math.Point3, math.Point3) {
	circle, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), radius, 0, sweep)
	return circle.PointAt(0), circle.PointAt(1)
}

// TestReflexContactRailKeepsMajorSector asserts a 270° reflex rim yields a rail arc that spans the MAJOR
// sector (|sweep| ≈ 270° > π), joins the two feet, and passes through the material midpoint at 135° — NOT the
// 90° minor complement a three-point re-fit would return.
func TestReflexContactRailKeepsMajorSector(t *testing.T) {
	t.Parallel()
	const railRadius = 42.28
	sweep := 1.5 * stdmath.Pi // +270°
	foot0, foot1 := contactCircleFeet(railRadius, sweep)
	seg, ok := reflexContactRail(reflexRimEdgeFixture(t, sweep), math.P3(0, 0, 0), railRadius, foot0, foot1)
	if !ok {
		t.Fatal("reflexContactRail declined a 270° reflex rim it should keep major")
	}
	arc, isArc := seg.curve.(geom.Arc3d)
	if !isArc || !seg.arc {
		t.Fatalf("reflex rail is not an Arc3d (curve %T arc=%v)", seg.curve, seg.arc)
	}
	if got := stdmath.Abs(arc.SweepAngle); stdmath.Abs(got-sweep) > 1e-6*sweep {
		t.Fatalf("reflex rail sweep %.6f, want the MAJOR %.6f (a minor snap reads ~%.6f)", got, sweep, stdmath.Pi/2)
	}
	wantMid := math.P3(railRadius*stdmath.Cos(0.75*stdmath.Pi), railRadius*stdmath.Sin(0.75*stdmath.Pi), 0)
	if d := float64(seg.mid.DistanceTo(wantMid)); d > 1e-6*railRadius {
		t.Fatalf("reflex rail midpoint %v, want the 135° material point %v (dist %.3g) — a minor arc bulges the wrong way", seg.mid, wantMid, d)
	}
	assertEndpoints(t, seg, foot0, foot1)
}

// TestReflexContactRailDeclinesConvexRim asserts a convex (<180°) rim and the nil edge both decline, so the
// caller keeps the byte-identical three-point minor rail (never routed through subArcMajor).
func TestReflexContactRailDeclinesConvexRim(t *testing.T) {
	t.Parallel()
	const railRadius = 42.28
	sweep := 0.5 * stdmath.Pi // +90°, convex
	foot0, foot1 := contactCircleFeet(railRadius, sweep)
	if _, ok := reflexContactRail(reflexRimEdgeFixture(t, sweep), math.P3(0, 0, 0), railRadius, foot0, foot1); ok {
		t.Fatal("reflexContactRail fired on a 90° convex rim — it must decline so the three-point rail is kept")
	}
	if _, ok := reflexContactRail(nil, math.P3(0, 0, 0), railRadius, foot0, foot1); ok {
		t.Fatal("reflexContactRail fired on a nil edge — it must decline")
	}
}

// assertEndpoints checks the rail seg joins foot0→foot1 exactly.
func assertEndpoints(t *testing.T, seg endSeg, foot0, foot1 math.Point3) {
	t.Helper()
	if d := float64(seg.from.DistanceTo(foot0)); d > 1e-9 {
		t.Fatalf("reflex rail from %v, want foot0 %v (dist %.3g)", seg.from, foot0, d)
	}
	if d := float64(seg.to.DistanceTo(foot1)); d > 1e-9 {
		t.Fatalf("reflex rail to %v, want foot1 %v (dist %.3g)", seg.to, foot1, d)
	}
}
