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

// TestCrossingCylinderCutRodTargetDefers: rod − fat (the disconnected stub case) is not the drill; the
// assembler declines so the caller keeps its fallback.
func TestCrossingCylinderCutRodTargetDefers(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if _, ok := CrossingCylinderCut(thin, fat); ok { // target is the rod
		t.Error("rod − fat (stubs) should defer from the drill assembler (ok=false)")
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
