// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Lofted-flange converge (#2086). Converge pinches the cornered profile's corners to points at the
// far profile, so it must (a) retarget exactly the corner points to their neighbour midpoints and
// (b) remove the carried-through corner material — a measurably smaller wall than the un-converged
// reference.

// TestConvergeCornersRetargetsOnlyCorners a square near band has four 90° corners, so converge moves
// exactly those four far points to their neighbour midpoints and leaves a smooth (many-sided) band
// untouched.
func TestConvergeCornersRetargetsOnlyCorners(t *testing.T) {
	t.Parallel()
	square := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}
	farSquare := []math.Point3{math.P3(-1, -1, 3), math.P3(3, -1, 3), math.P3(3, 3, 3), math.P3(-1, 3, 3)}
	got, count := convergeCorners(square, farSquare)
	if count != 4 {
		t.Fatalf("converge pinched %d corners, want 4", count)
	}
	// Corner 1 (index 1) must move to the midpoint of far corners 0 and 2.
	want := farSquare[0].Midpoint(farSquare[2])
	if d := float64(got[1].DistanceTo(want)); d > 1e-9 {
		t.Errorf("converged corner 1 at %v, want the neighbour midpoint %v (off %.4f)", got[1], want, d)
	}
	// A smooth many-sided ring turns too gently to have corners: nothing converges.
	ring := regularRing(24, 2, 0)
	if _, c := convergeCorners(ring, regularRing(24, 3, 3)); c != 0 {
		t.Errorf("a smooth 24-gon converged %d corners, want 0", c)
	}
}

// regularRing is an m-gon of radius r in the z-plane — a stand-in for a rounded profile.
func regularRing(m int, r, z float64) []math.Point3 {
	pts := make([]math.Point3, m)
	for k := range m {
		a := 2 * stdmath.Pi * float64(k) / float64(m)
		pts[k] = math.P3(r*stdmath.Cos(a), r*stdmath.Sin(a), z)
	}
	return pts
}

// TestLoftedFlangeConvergeRemovesCornerMaterial a converged wall pinches the profile corners to
// points, so it holds less material than the same wall carried through un-converged, stays a valid
// watertight solid, and no longer reports the deferral.
func TestLoftedFlangeConvergeRemovesCornerMaterial(t *testing.T) {
	t.Parallel()
	build := func(converge bool) (float64, bool, int) {
		planeB, _ := sketch.NewPlane(math.P3(0, 0, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
		fs := NewPartFeatures(thicknessParams(t))
		pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
			ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1), ProfileB: lProfileOnPlane(planeB, 2, 2),
			Operation: ops.NewBody, Converge: converge,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("lofted flange (converge=%v) sick: %+v", converge, pf.Health())
		}
		body := fs.Result()[0]
		unmodelled := 0
		if hasDiagCode(pf.Diagnostics(), codeLoftedFlangeUnmodeled) {
			unmodelled = 1
		}
		valid, _ := ops.ValidateBodyEntities(body, ops.CheckGeometry, ops.DefaultQuality())
		return ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume, valid, unmodelled
	}
	ref, refValid, _ := build(false)
	conv, convValid, unmodelled := build(true)
	if !refValid || !convValid {
		t.Fatalf("invalid solids: reference valid=%v, converged valid=%v", refValid, convValid)
	}
	if !(conv < ref) {
		t.Errorf("converge did not remove corner material: converged %.4f, reference %.4f", conv, ref)
	}
	if unmodelled != 0 {
		t.Errorf("a converged wall with corners must not report %q", codeLoftedFlangeUnmodeled)
	}
}
