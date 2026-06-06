// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// A blind hole (depth less than the thickness) is a clean watertight solid: a cylinder wall
// plus a flat bottom disk, removing an inscribed cylinder of material.
func TestCutBlindCylindricalHole(t *testing.T) {
	// 10×10×4 slab, Ø4 blind hole 3 deep along +Z from the bottom face (z=0).
	d, err := brep.CutBlindCylindricalHole(box(0, 0, 0, 10, 10, 4), math.P3(5, 5, 0), math.V3(0, 0, 1), 2, 3)
	if err != nil {
		t.Fatalf("CutBlindCylindricalHole: %v", err)
	}
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("blind-drilled slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(d); len(open) != 0 {
		t.Fatalf("blind hole has %d boundary edges, want 0 (watertight)", len(open))
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
	if nCyl != 1 || nPlane != 7 {
		t.Errorf("faces = %d cylinder / %d plane, want 1 / 7 (wall + 6 slab faces + flat bottom)", nCyl, nPlane)
	}
	// Removed = an inscribed Ø4 cylinder 3 deep: a hair under π·r²·depth.
	const bore = stdmath.Pi * 2 * 2 * 3
	removed := 10.0*10.0*4.0 - vol(d)
	if removed <= 0 || removed > bore+1e-9 || (bore-removed)/bore > 0.04 {
		t.Errorf("removed volume = %g, want a hair under %g (π·r²·depth)", removed, bore)
	}
}

// A depth that would punch through the part is rejected (the through specialization or the
// general boolean handles that), rather than building a broken body.
func TestCutBlindRejectsThroughDepth(t *testing.T) {
	_, err := brep.CutBlindCylindricalHole(box(0, 0, 0, 10, 10, 4), math.P3(5, 5, 0), math.V3(0, 0, 1), 2, 5)
	if err == nil {
		t.Error("expected an error for a blind depth that exits the part, got nil")
	}
}
