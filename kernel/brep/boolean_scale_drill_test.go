// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestDrilledPlateBuildsAcrossScales is the corpus row for the band cull margin (ADR-0061): the
// SIMPLEST curved boolean there is — a block cut by a cylinder, six faces against three — must build
// the same seven-face solid whatever the part's size. The margin bandPlacement used to call a section
// "strictly inside the wall's band" was ten planar stitch grids as an ABSOLUTE length, so at a 100 µm
// plate the top cap's section sat exactly one pad below the bore's rim, read as a rim contact, and the
// drill declined. The margin is the band's own now (bandCullPad), which is scale-invariant.
func TestDrilledPlateBuildsAcrossScales(t *testing.T) {
	t.Parallel()
	for _, s := range []float64{1e-4, 1e-3, 1, 1e3, 1e4} {
		res, err := drilledPlateAtScale(t, s)
		if err != nil {
			t.Errorf("scale %g: the general pipeline declined the simplest drill: %v", s, err)
			continue
		}
		if res == nil {
			t.Errorf("scale %g: the drill returned an EMPTY body; the plate is mostly material", s)
			continue
		}
		assertDrilledPlate(t, res, s)
	}
}

// drilledPlateAtScale cuts a cylinder through a slab, both scaled by s.
func drilledPlateAtScale(t *testing.T, s float64) (*topo.Body, error) {
	t.Helper()
	slab, err := SolidBlock(math.P3(math.Scalar(-s), math.Scalar(-s), 0),
		math.P3(math.Scalar(s), math.Scalar(s), math.Scalar(0.6*s)), "slab")
	if err != nil {
		t.Fatalf("scale %g: SolidBlock: %v", s, err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, math.Scalar(-0.1*s)), math.V3(0, 0, 1),
		math.Scalar(0.3*s), math.Scalar(0.8*s))
	if err != nil {
		t.Fatalf("scale %g: SolidCylinder: %v", s, err)
	}
	return BooleanDiag(Difference, slab, rod, nil)
}

// assertDrilledPlate pins the result's shape: six box planes plus ONE analytic bore wall, and every
// edge used twice (the through-hole is closed).
func assertDrilledPlate(t *testing.T, b *topo.Body, s float64) {
	t.Helper()
	if got := len(b.Faces()); got != 7 {
		t.Errorf("scale %g: drilled plate has %d faces, want 7 (6 box planes + the bore wall)", s, got)
	}
	walls := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			walls++
		}
	}
	if walls != 1 {
		t.Errorf("scale %g: drilled plate carries %d analytic cylinder face(s), want 1 — the bore must "+
			"stay a cylinder, not a fan of chords", s, walls)
	}
	for _, e := range b.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Errorf("scale %g: edge %v has %d uses, want 2 — the drilled plate must be closed", s, e.Lineage(), n)
		}
	}
}
