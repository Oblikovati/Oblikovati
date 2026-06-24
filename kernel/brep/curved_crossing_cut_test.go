// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder cut assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). Drilling a fat cylinder with a
// crossing rod must weld into a watertight solid: the fat's two planar caps, its side wall carrying the two
// lens holes, and the rod-wall tunnel (flipped inward). Volume is checked through ops_test; here the concern
// is the watertight topology and that the surfaces stay analytic.

// TestCrossingCylinderCutDrill drills a radius-3 cylinder with a radius-1.5 rod and checks the result is a
// watertight four-face solid: two planar caps, the holed cylinder wall (two holes), the tunnel wall.
func TestCrossingCylinderCutDrill(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	res, ok := CrossingCylinderCut(fat, thin)
	if !ok {
		t.Fatal("drill (fat − rod) declined; want a four-face solid")
	}
	if !res.IsSolid() {
		t.Fatalf("drill result is not a solid: %+v", res)
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
	if cyls != 2 || planes != 2 {
		t.Errorf("got %d cylinder + %d planar faces, want 2 + 2", cyls, planes)
	}
	if holed != 1 {
		t.Errorf("got %d faces with two holes, want 1 (the drilled side wall)", holed)
	}
}

// countInnerLoops returns how many of a face's loops are inner (hole) loops.
func countInnerLoops(f *topo.Face) int {
	n := 0
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			n++
		}
	}
	return n
}

// TestCrossingCylinderCutRodMinusFatStubs: rod − fat is the two disconnected rod stubs (the rod sticking
// out either side of the fat). Each stub is a closed lump of three faces (the rod stub band, the rod end
// cap, and the fat-wall lens), merged into one two-shell solid.
func TestCrossingCylinderCutRodMinusFatStubs(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	res, ok := CrossingCylinderCut(thin, fat) // target is the rod
	if !ok {
		t.Fatal("rod − fat declined; want two disconnected rod stubs")
	}
	if !res.IsSolid() {
		t.Fatalf("rod − fat result is not a solid: %+v", res)
	}
	if n := len(res.Shells()); n != 2 {
		t.Errorf("rod − fat has %d shells, want 2 (a disconnected stub each side)", n)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	if n := len(res.Faces()); n != 6 {
		t.Errorf("rod − fat has %d faces, want 6 (band + rod cap + lens, twice)", n)
	}
}

// TestCrossingCylinderCutNonCylinderDefers: the drill only handles two bare cylinders.
func TestCrossingCylinderCutNonCylinderDefers(t *testing.T) {
	block, _ := SolidBlock(math.P3(-2, -2, -2), math.P3(2, 2, 2), "b")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 1, 12)
	if _, ok := CrossingCylinderCut(block, cyl); ok {
		t.Error("a block − cylinder should defer from the drill assembler (ok=false)")
	}
}
