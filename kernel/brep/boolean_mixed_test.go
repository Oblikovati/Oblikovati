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

// TestMixedBooleanDeclinesCurvedInteraction: a cut that genuinely CROSSES the boss wall must DECLINE
// (ErrNonPlanar) — imprinting a cylinder wall is the bespoke curved handlers' and the reconstruction
// path's job. The tool reaches x=8, well past the wall (radius 2 about (5,5)), and the exact
// interaction gate proves the contact (a ruling-line pair inside the tool face's trim and the band).
func TestMixedBooleanDeclinesCurvedInteraction(t *testing.T) {
	bossed := bossedBlock(t)
	through, _ := brep.SolidBlock(math.P3(4, 4, 9), math.P3(8, 6, 12), "through")
	if _, err := brep.BooleanDiag(brep.Difference, bossed, through, nil); err == nil {
		t.Fatal("cut crossing the boss wall did not decline; want ErrNonPlanar (conservative scope)")
	}
}

// TestMixedBooleanCavityThroughSeatHole: a tool box passing through the SEAT PLANE inside the boss's
// hole region — fully interior to the block∪boss solid, touching no face (the exact interaction gate
// proves the wall clear; the exact trim clipping mints no imprint inside the hole void; the membership
// oracle sees the TRUE hole boundary). The cut removes EXACTLY the tool volume as an embedded cavity.
// Asserted as a volume DELTA against the uncut fixture, isolating the boolean from any fixture bias.
func TestMixedBooleanCavityThroughSeatHole(t *testing.T) {
	bossed := bossedBlock(t)
	before := vol(bossed)
	tool, _ := brep.SolidBlock(math.P3(4, 4, 9), math.P3(6, 6, 12), "tool")
	res, err := brep.BooleanDiag(brep.Difference, bossed, tool, nil)
	if err != nil {
		t.Fatalf("through-seat-hole cavity declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("cavity result is not a solid")
	}
	if removed := before - vol(res); stdmath.Abs(removed-12) > 1e-6 {
		t.Errorf("cavity removed %g, want exactly 12 (the embedded tool's volume)", removed)
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

// TestMixedBooleanEmbeddedCavityCut: subtracting a cylinder wholly inside the block cuts an exact
// cylindrical cavity — the tool's pass-through faces (curved wall + circular-edged caps) are kept
// REVERSED into the void, welded by the unified stitch as an inner shell.
func TestMixedBooleanEmbeddedCavityCut(t *testing.T) {
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	tool, _ := brep.SolidCylinder(math.P3(5, 5, 3), math.V3(0, 0, 1), 1, 4)
	res, err := brep.BooleanDiag(brep.Difference, block, tool, nil)
	if err != nil {
		t.Fatalf("embedded cavity cut declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("cavity result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("cavity wall not analytic: %d cylinder faces, want 1", n)
	}
	want := 1000 - stdmath.Pi*1*4
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("cavity volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanPocketOnSeatFace: cutting a pocket into the TOP face of the bossed block — the face
// that carries the boss's rim circle as a hole. The curved hole is detached, the seat splits through
// the exact polygonal pipeline, and the rim circle re-attaches EXACTLY to the fragment containing it,
// so it still welds with the pass-through boss wall (previously this declined: the seat face was
// pass-through-only and its box spans the whole top).
func TestMixedBooleanPocketOnSeatFace(t *testing.T) {
	bossed := bossedBlock(t)
	pocket, _ := brep.SolidBlock(math.P3(1, 1, 9), math.P3(2.8, 2.8, 11), "pocket")
	res, err := brep.BooleanDiag(brep.Difference, bossed, pocket, nil)
	if err != nil {
		t.Fatalf("seat-face pocket declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("seat-face pocket result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 - 1.8*1.8*1 // pocket bites 1 deep into the top
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("seat-face pocket volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanCavityInsideCylinder: a box wholly inside a cylinder — the box's faces overlap the
// wall's bounding box everywhere, so the retired box gate always declined; the exact interaction gate
// (plane∩cylinder ruling/circle curves against both trims, OCCT IntTools style) proves the wall clear
// and the cut becomes an exact embedded cavity in a CURVED body.
func TestMixedBooleanCavityInsideCylinder(t *testing.T) {
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	before := vol(cyl)
	tool, _ := brep.SolidBlock(math.P3(-0.5, -0.5, 2), math.P3(0.5, 0.5, 3), "tool")
	res, err := brep.BooleanDiag(brep.Difference, cyl, tool, nil)
	if err != nil {
		t.Fatalf("cavity inside cylinder declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("cavity-in-cylinder invalid (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	// A volume DELTA against the uncut fixture: the cut must remove exactly the tool's volume,
	// isolating the boolean from a pre-existing cylinder-wall mass-properties bias (~0.08·h at r=2).
	if removed := before - vol(res); stdmath.Abs(removed-1) > 1e-6 {
		t.Errorf("cavity-in-cylinder removed %g, want exactly 1", removed)
	}
}
