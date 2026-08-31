// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Per-face-dispatch mixed boolean (ADR-0058): a body with curved faces booleans EXACTLY through the
// planar pipeline when the curved (and curved-edged planar) faces are clear of the tool — they pass
// through whole into the unified stitch. Out-of-scope configurations decline with ErrNonPlanar so the
// curved/CSG fallbacks run as before.

// bossedBlock is the mixed fixture: a 10³ block with an r=2 h=3 cylindrical boss on its top face —
// straight-edged planar sides, plus a curved wall, a circular-hole seat and a circular-rim cap.
func bossedBlock(t *testing.T) *topo.Body {
	t.Helper()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	cyl, err := brep.SolidCylinder(math.P3(5, 5, 10), math.V3(0, 0, 1), 2, 3)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	bossed, ok := brep.JoinCylindricalBoss(block, cyl)
	if !ok || bossed == nil {
		t.Fatalf("JoinCylindricalBoss declined; fixture unavailable")
	}
	return bossed
}

func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestMixedBooleanNotchAwayFromBoss: cutting a notch into the bossed block's side, far from the boss,
// runs the exact planar pipeline on the block faces while the boss (wall, seat hole, cap) passes
// through analytically — valid solid, exact volume, cylinder wall intact.
func TestMixedBooleanNotchAwayFromBoss(t *testing.T) {
	bossed := bossedBlock(t)
	notch, _ := brep.SolidBlock(math.P3(-1, 4, 1), math.P3(2, 6, 3), "notch")
	res, err := brep.BooleanDiag(brep.Difference, bossed, notch, nil)
	if err != nil {
		t.Fatalf("mixed difference declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("mixed difference result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 - 2*2*2 // block + boss − notch overlap
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("mixed difference volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanUnionAwayFromBoss: welding an add-on block onto a side face away from the boss.
func TestMixedBooleanUnionAwayFromBoss(t *testing.T) {
	bossed := bossedBlock(t)
	addon, _ := brep.SolidBlock(math.P3(10, 4, 1), math.P3(12, 6, 3), "addon")
	res, err := brep.BooleanDiag(brep.Union, bossed, addon, nil)
	if err != nil {
		t.Fatalf("mixed union declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("mixed union result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 + 2*2*2
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("mixed union volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanDeclinesCurvedInteraction: a cut that reaches the boss region must DECLINE
// (ErrNonPlanar) — the curved wall and the seat's circular hole would need imprinting, which is the
// bespoke curved handlers' and the reconstruction path's job.
func TestMixedBooleanDeclinesCurvedInteraction(t *testing.T) {
	bossed := bossedBlock(t)
	through, _ := brep.SolidBlock(math.P3(4, 4, 9), math.P3(6, 6, 12), "through")
	if _, err := brep.BooleanDiag(brep.Difference, bossed, through, nil); err == nil {
		t.Fatal("cut through the boss did not decline; want ErrNonPlanar (conservative scope)")
	}
}

// TestMixedBooleanDisjointDifference: an all-pass-through operand (a lone cylinder — curved wall +
// circular-rim caps) minus a disjoint box keeps every face whole through the unified stitch.
func TestMixedBooleanDisjointDifference(t *testing.T) {
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	far, _ := brep.SolidBlock(math.P3(20, 20, 20), math.P3(22, 22, 22), "far")
	res, err := brep.BooleanDiag(brep.Difference, cyl, far, nil)
	if err != nil {
		t.Fatalf("disjoint mixed difference declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("disjoint difference did not keep the cylinder whole (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	if got, want := vol(res), stdmath.Pi*4*4; stdmath.Abs(got-want) > 0.5 {
		t.Errorf("disjoint difference volume = %g, want %g", got, want)
	}
}
