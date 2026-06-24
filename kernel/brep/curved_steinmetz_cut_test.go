// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz cut and join (M2 Phase 2, Oblikovati/Oblikovati#1335). Subtracting and uniting two
// equal-radius perpendicular cylinders: each cylinder's side splits into two saddle-rimmed bands. Volume is
// checked through ops_test; here the concern is the watertight (pinched) topology and analytic surfaces.

// TestSteinmetzCutIsWatertight subtracts two radius-3 cylinders and checks the result is a watertight
// six-face solid: two caps, two cylinder-A bands, and two reversed cylinder-B lobes.
func TestSteinmetzCutIsWatertight(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := EqualRadiusSteinmetzCut(cx, cz)
	if !ok {
		t.Fatal("Steinmetz cut declined; want a six-face bitten solid")
	}
	if !res.IsSolid() {
		t.Fatalf("Steinmetz cut result is not a solid: %+v", res)
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
	if cyls != 4 || planes != 2 {
		t.Errorf("got %d cylinder + %d planar faces, want 4 (two A bands + two B lobes) + 2 (caps)", cyls, planes)
	}
}

// TestSteinmetzJoinIsWatertight unites two radius-3 cylinders and checks the result is a watertight
// eight-face solid: four caps and four cylinder bands (two per cylinder).
func TestSteinmetzJoinIsWatertight(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := EqualRadiusSteinmetzJoin(cx, cz)
	if !ok {
		t.Fatal("Steinmetz join declined; want an eight-face union solid")
	}
	if !res.IsSolid() {
		t.Fatalf("Steinmetz join result is not a solid: %+v", res)
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
	if cyls != 4 || planes != 4 {
		t.Errorf("got %d cylinder + %d planar faces, want 4 (two bands per cylinder) + 4 (caps)", cyls, planes)
	}
}

// TestSteinmetzCutUnequalRadiusDefers: unequal radii are the thin-through-fat crossing case, not Steinmetz.
func TestSteinmetzCutUnequalRadiusDefers(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := EqualRadiusSteinmetzCut(cx, cz); ok {
		t.Error("unequal-radius cut should defer from the Steinmetz assembler (ok=false)")
	}
	if _, ok := EqualRadiusSteinmetzJoin(cx, cz); ok {
		t.Error("unequal-radius join should defer from the Steinmetz assembler (ok=false)")
	}
}
