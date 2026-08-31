// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// drilledSlab is a 10×10×4 box with a through-hole of radius 2 along +Z at its centre.
func drilledSlab(t *testing.T) *topo.Body {
	t.Helper()
	got, err := brep.CutCylindricalHole(box(0, 0, 0, 10, 10, 4), math.P3(5, 5, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("CutCylindricalHole: %v", err)
	}
	return got
}

// K1b slice 3: a cylinder cut through a planar slab is a clean watertight solid — six planar
// faces (the two pierced ones now holed) plus a single TRUE cylinder wall face.
func TestCutCylindricalHoleIsValidSolid(t *testing.T) {
	d := drilledSlab(t)
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("drilled slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(d); len(open) != 0 {
		t.Fatalf("drilled slab has %d boundary edges, want 0 (watertight)", len(open))
	}
	nCyl, nPlane := 0, 0
	for _, f := range d.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			nCyl++
		case geom.Plane:
			nPlane++
		}
	}
	if nCyl != 1 || nPlane != 6 {
		t.Errorf("faces = %d cylinder / %d plane, want 1 / 6 (one hole wall, six slab faces)", nCyl, nPlane)
	}
}

// The drilled volume is the slab minus the (inscribed) cylinder — proving the wall's material
// side faces the axis (a reversed face), so the removed core is subtracted, not added.
func TestCutCylindricalHoleVolume(t *testing.T) {
	const slab, bore = 10.0 * 10.0 * 4.0, stdmath.Pi * 2 * 2 * 4 // box, π·r²·h
	removed := slab - vol(drilledSlab(t))
	if removed <= 0 {
		t.Fatalf("removed volume = %g, want positive (material drilled out)", removed)
	}
	if removed > bore+1e-9 || (bore-removed)/bore > 0.04 {
		t.Errorf("removed volume = %g, want a hair under %g (π·r²·h, inscribed)", removed, bore)
	}
}

// A slab face the hole runs alongside is copied unchanged, so its reference key survives the
// cut and a pick made before drilling rebinds to the result face (K1a/K1b on a curved cut).
func TestDrillPreservesUntouchedFaceKey(t *testing.T) {
	slab := box(0, 0, 0, 10, 10, 4)
	side := faceWithNormal(slab, math.V3(1, 0, 0)) // the +X wall, parallel to the drill axis
	if side == nil {
		t.Fatal("no +X face on the slab")
	}
	key := side.ReferenceKey()
	d, err := brep.CutCylindricalHole(slab, math.P3(5, 5, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.FindFaceByKey(key); !ok {
		t.Error("the +X face's reference key did not survive the drill")
	}
}

// After the ADR-0045 dedup, CutCylindricalHole delegates to the shared curvedStitch drill path, so it now
// drills a slab that ALREADY carries a curved face (a prior bore) — the old bespoke planar welder required
// an all-planar slab and errored here. Drill a second, clearing hole along +X through a once-bored slab and
// assert an exact analytic solid results (two cylinder walls now, no CSG fallback).
func TestCutCylindricalHoleThroughAlreadyCurvedSlab(t *testing.T) {
	bored := drilledSlab(t) // 10×10×4 with a +Z bore at centre → carries a cylinder face
	got, err := brep.CutCylindricalHole(bored, math.P3(0, 1.5, 2), math.V3(1, 0, 0), 1)
	if err != nil {
		t.Fatalf("CutCylindricalHole through a curved-faced slab: %v", err)
	}
	if r := ops.Validate(got); !r.Valid || !got.IsSolid() {
		t.Fatalf("second-bore slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(got); len(open) != 0 {
		t.Fatalf("second-bore slab has %d boundary edges, want 0 (watertight)", len(open))
	}
	nCyl := 0
	for _, f := range got.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			nCyl++
		}
	}
	if nCyl != 2 {
		t.Errorf("cylinder faces = %d, want 2 (both bore walls kept analytic, no CSG)", nCyl)
	}
}

// A hole whose circle spills past the face boundary needs the general boolean, not this
// through-hole specialization — it must error rather than build a broken body.
func TestDrillRejectsOversizeHole(t *testing.T) {
	_, err := brep.CutCylindricalHole(box(0, 0, 0, 10, 10, 4), math.P3(5, 5, 0), math.V3(0, 0, 1), 8)
	if err == nil {
		t.Error("expected an error for a hole larger than the face, got nil")
	}
}

// TestDrillThroughHoleDeclinesEmbeddedTool is the span-gate regression: a tool cylinder that does not
// axially SPAN the slab (an embedded or blind cut) must decline rather than drill the unbounded axis
// as a full through-hole (the silent wrong-volume misfire the per-face dispatch exposed, ADR-0058).
func TestDrillThroughHoleDeclinesEmbeddedTool(t *testing.T) {
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	embedded, _ := brep.SolidCylinder(math.P3(5, 5, 3), math.V3(0, 0, 1), 1, 4)
	if _, ok := brep.DrillThroughHole(block, embedded); ok {
		t.Fatal("embedded (non-spanning) tool did not decline the through-hole recipe")
	}
	spanning, _ := brep.SolidCylinder(math.P3(5, 5, -1), math.V3(0, 0, 1), 1, 12)
	if _, ok := brep.DrillThroughHole(block, spanning); !ok {
		t.Fatal("spanning tool declined; the through-hole recipe should still accept it")
	}
}
