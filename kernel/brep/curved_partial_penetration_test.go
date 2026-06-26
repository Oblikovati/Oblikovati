// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Partial-penetration intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). A thin rod ending inside a
// fatter cylinder intersects it in the rod plug: a lens cap on the fat wall, a rod stub band, and the rod's
// blind end cap. Volume is checked through ops_test; here the concern is the watertight topology and that
// the surfaces stay analytic (two cylinder faces and one planar end cap).

// TestPartialPenetrationPlugIsWatertight intersects a radius-3 cylinder with a radius-1.5 rod that ends at
// the fat centre and checks the result is a watertight three-face solid: the fat-wall lens, the rod stub
// band, and the planar blind end cap.
func TestPartialPenetrationPlugIsWatertight(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	stub, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6) // ends at x=0, inside the fat

	res, ok := PartialPenetrationIntersect(fat, stub, nil)
	if !ok {
		t.Fatal("partial-penetration intersection declined; want a three-face plug")
	}
	if !res.IsSolid() {
		t.Fatalf("plug result is not a solid: %+v", res)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	cyls, planes := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	if cyls != 2 || planes != 1 {
		t.Errorf("got %d cylinder + %d planar faces, want 2 (lens + stub band) + 1 (blind end cap)", cyls, planes)
	}
}

// TestPartialPenetrationBlindHoleIsWatertight cuts a radius-1.5 rod (ending at the fat centre) from a
// radius-3 cylinder and checks the blind pocket is a watertight solid: two fat caps, the holed fat wall
// (one hole), the rod tunnel, and the blind end cap.
func TestPartialPenetrationBlindHoleIsWatertight(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	stub, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)

	res, ok := PartialPenetrationCut(fat, stub, nil)
	if !ok {
		t.Fatal("blind-hole cut declined; want a five-face pocketed solid")
	}
	if !res.IsSolid() {
		t.Fatalf("blind hole result is not a solid: %+v", res)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	cyls, planes, holed := 0, 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
		if countInnerLoops(f) == 1 {
			holed++ // the fat side wall with its one lens hole
		}
	}
	if cyls != 2 || planes != 3 {
		t.Errorf("got %d cylinder + %d planar faces, want 2 (holed wall + tunnel) + 3 (two fat caps + blind bottom)", cyls, planes)
	}
	if holed != 1 {
		t.Errorf("got %d faces with one hole, want 1 (the pocketed wall)", holed)
	}
}

// TestPartialPenetrationJoinIsWatertight joins a radius-1.5 rod (ending at the fat centre) with a radius-3
// cylinder and checks the result is a watertight solid: two fat caps, the holed fat wall (one hole), the rod
// stub band, and the rod's entry end cap.
func TestPartialPenetrationJoinIsWatertight(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	stub, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)

	res, ok := PartialPenetrationJoin(fat, stub, nil)
	if !ok {
		t.Fatal("partial-penetration join declined; want a five-face stubbed solid")
	}
	if !res.IsSolid() {
		t.Fatalf("join result is not a solid: %+v", res)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	cyls, planes := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	if cyls != 2 || planes != 3 {
		t.Errorf("got %d cylinder + %d planar faces, want 2 (holed wall + stub band) + 3 (two fat caps + entry cap)", cyls, planes)
	}
}

// TestPartialPenetrationRodMinusFatStub cuts the fat cylinder out of the rod (rod − fat) and checks the
// result is the single rod stub outside the fat: a watertight one-shell lump of three analytic faces — the
// fat-wall lens, the rod stub band, and the rod's entry end cap.
func TestPartialPenetrationRodMinusFatStub(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	stub, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)

	res, ok := PartialPenetrationCut(stub, fat, nil) // target is the rod
	if !ok {
		t.Fatal("rod − fat declined; want a single rod stub lump")
	}
	if !res.IsSolid() {
		t.Fatalf("rod − fat result is not a solid: %+v", res)
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("rod − fat has %d shells, want 1 (the rod only breaches one wall)", n)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	if n := len(res.Faces()); n != 3 {
		t.Errorf("rod − fat has %d faces, want 3 (lens, stub band, entry cap)", n)
	}
}

// TestPartialPenetrationFullCrossingDefers: a rod that crosses all the way through gives two imprint loops,
// not the single loop of a partial penetration, so the plug assembler declines.
func TestPartialPenetrationFullCrossingDefers(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thru, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12) // spans both walls
	if _, ok := PartialPenetrationIntersect(fat, thru, nil); ok {
		t.Error("a full crossing (two loops) should defer from the plug assembler (ok=false)")
	}
}

// TestPartialPenetrationContainedRodDefers: a rod wholly inside the fat never breaches a wall (no imprint
// loop), so the plug assembler declines (the intersection is the whole rod, handled elsewhere).
func TestPartialPenetrationContainedRodDefers(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	inside, _ := SolidCylinder(math.P3(-1, 0, 0), math.V3(1, 0, 0), 0.5, 2) // x∈[-1,1], wholly inside
	if _, ok := PartialPenetrationIntersect(fat, inside, nil); ok {
		t.Error("a fully-contained rod should defer from the plug assembler (ok=false)")
	}
}
