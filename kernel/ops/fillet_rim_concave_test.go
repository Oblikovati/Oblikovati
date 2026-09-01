// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// slabWithBore builds a plate with a clean cylindrical through-hole — a synthetic bore-lip fixture (the
// concave dual of brep.SolidCylinder's plain boss): a box drilled through by CutCylindricalHole, so the
// hole's top rim is a CONCAVE circular edge (the plate material sits OUTSIDE the bore radius).
func slabWithBore(t *testing.T, size, height, boreCenter, radius float64) *topo.Body {
	t.Helper()
	m := subd.Box(size, size, height)
	slab := subd.ToBody(m, "slab")
	drilled, err := brep.CutCylindricalHole(slab, math.P3(boreCenter, boreCenter, 0), math.V3(0, 0, 1), radius)
	if err != nil {
		t.Fatalf("slabWithBore: drill radius %g in a %gx%g slab: %v", radius, size, size, err)
	}
	return drilled
}

// circularRimKeyAtZ returns the reference key of the body's circular edge nearest height z — the same
// lookup fillet_rim_op_test.go's topRimKey uses for a plain cylinder's top rim, generalized to a drilled
// slab whose only circular edges are the bore's top and bottom rims.
func circularRimKeyAtZ(t *testing.T, b *topo.Body, z float64) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok && stdmath.Abs(e.RangeBox().Center().Z-z) < 1e-3 {
			return e.ReferenceKey()
		}
		if a, ok := e.Geometry().(geom.Arc3d); ok && stdmath.Abs(a.SweepAngle) > 6 && stdmath.Abs(e.RangeBox().Center().Z-z) < 1e-3 {
			return e.ReferenceKey()
		}
	}
	t.Fatalf("no circular edge near z=%g", z)
	return nil
}

// TestConcaveRimBuildsRPlusRTorus is the brief's KERNEL regression for the bore-lip mirror: a plate
// drilled with a clean through-hole (cylinder radius 3) rounds its TOP rim — a CONCAVE edge, the plate
// material sits OUTSIDE the bore — into a torus band whose MAJOR radius is R+r (3+1=4), the mirror of
// solveRim's convex R−r, and the result stays a fully watertight solid (Valid && Closed && Manifold &&
// HolesContained && IsSolid).
func TestConcaveRimBuildsRPlusRTorus(t *testing.T) {
	t.Parallel()
	const size, height, boreCenter, radius, r = 20.0, 4.0, 10.0, 3.0, 1.0
	drilled := slabWithBore(t, size, height, boreCenter, radius)
	rimKey := circularRimKeyAtZ(t, drilled, height)
	res, err := ops.FilletEdges(drilled, [][]byte{rimKey}, r)
	if err != nil {
		t.Fatalf("bore-lip rim fillet: %v", err)
	}
	rep := ops.Validate(res)
	if !rep.Valid || !rep.Closed || !rep.Manifold || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("bore-lip rim result not watertight: valid=%v closed=%v manifold=%v holes=%v solid=%v issues=%v",
			rep.Valid, rep.Closed, rep.Manifold, rep.HolesContained, res.IsSolid(), rep.Issues)
	}
	tor, count := geom.Torus{}, 0
	for _, f := range res.Faces() {
		if g, ok := f.Geometry().(geom.Torus); ok {
			tor, count = g, count+1
		}
	}
	if count != 1 {
		t.Fatalf("bore-lip rim result has %d torus faces, want exactly 1", count)
	}
	if want := radius + r; tor.MajorRadius != want {
		t.Errorf("bore-lip torus major radius = %g, want R+r = %g (R=%g, r=%g)", tor.MajorRadius, want, radius, r)
	}
	if tor.MinorRadius != r {
		t.Errorf("bore-lip torus minor radius = %g, want r = %g", tor.MinorRadius, r)
	}
}

// TestConvexBossRimStaysRMinusR is the brief's KERNEL do-no-harm regression: an I9-shaped fixture (a
// plain solid cylinder, boss material INSIDE the rim) still rounds its top rim through the UNCHANGED
// convex solveRim path — torus major R−r, never the concave R+r mirror — confirming the new concave
// branch in resolveRim (fillet_rim.go) never fires on a rim solveRim already builds.
func TestConvexBossRimStaysRMinusR(t *testing.T) {
	t.Parallel()
	const radius, height, r = 5.0, 8.0, 1.5
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), radius, height)
	if err != nil {
		t.Fatal(err)
	}
	rimKey := circularRimKeyAtZ(t, b, height)
	res, err := ops.FilletEdges(b, [][]byte{rimKey}, r)
	if err != nil {
		t.Fatalf("convex boss rim fillet: %v", err)
	}
	if rep := ops.Validate(res); !rep.Valid || !res.IsSolid() {
		t.Fatalf("convex boss rim result not a valid solid: %+v", rep.Issues)
	}
	tor, count := geom.Torus{}, 0
	for _, f := range res.Faces() {
		if g, ok := f.Geometry().(geom.Torus); ok {
			tor, count = g, count+1
		}
	}
	if count != 1 {
		t.Fatalf("convex boss rim result has %d torus faces, want exactly 1", count)
	}
	if want := radius - r; tor.MajorRadius != want {
		t.Errorf("convex boss torus major radius = %g, want R−r = %g (R=%g, r=%g) — the concave branch must not fire here", tor.MajorRadius, want, radius, r)
	}
}
