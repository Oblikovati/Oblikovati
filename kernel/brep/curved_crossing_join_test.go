// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Crossing-cylinder join assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). Joining a fat cylinder with a
// crossing rod must weld into one watertight solid: the fat's two planar caps, its side wall carrying the
// two lens holes, and a rod-wall stub out of each hole capped by the rod's own end disc. Volume is checked
// through ops_test; here the concern is the watertight topology and that the surfaces stay analytic.

// TestCrossingCylinderJoinStubs joins a radius-3 cylinder with a radius-1.5 rod and checks the result is a
// watertight single-shell solid of seven analytic faces: two fat caps, the holed fat wall (two holes), two
// rod stub bands, and two rod end caps.
func TestCrossingCylinderJoinStubs(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	res, ok := CrossingCylinderJoin(fat, thin, nil)
	if !ok {
		t.Fatal("join (fat ∪ rod) declined; want a single-shell stub solid")
	}
	if !res.IsSolid() {
		t.Fatalf("join result is not a solid: %+v", res)
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("join has %d shells, want 1 (one connected body)", n)
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
		if countInnerLoops(f) == 2 {
			holed++ // the fat side wall with its two lens holes
		}
	}
	if cyls != 3 || planes != 4 {
		t.Errorf("got %d cylinder + %d planar faces, want 3 (fat wall + two stub bands) + 4 (two fat caps + two rod caps)", cyls, planes)
	}
	if holed != 1 {
		t.Errorf("got %d faces with two holes, want 1 (the fat side wall)", holed)
	}
}

// TestCrossingCylinderJoinNonCylinderDefers: the join only handles two bare cylinders.
func TestCrossingCylinderJoinNonCylinderDefers(t *testing.T) {
	block, _ := SolidBlock(math.P3(-2, -2, -2), math.P3(2, 2, 2), "b")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 1, 12)
	if _, ok := CrossingCylinderJoin(block, cyl, nil); ok {
		t.Error("a block ∪ cylinder should defer from the join assembler (ok=false)")
	}
}
