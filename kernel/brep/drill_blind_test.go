// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// A blind hole (depth less than the thickness) is a clean watertight solid: a cylinder wall
// plus a flat bottom disk, removing an inscribed cylinder of material.
func TestCutBlindCylindricalHole(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	_, err := brep.CutBlindCylindricalHole(box(0, 0, 0, 10, 10, 4), math.P3(5, 5, 0), math.V3(0, 0, 1), 2, 5)
	if err == nil {
		t.Error("expected an error for a blind depth that exits the part, got nil")
	}
}

func TestCutBlindCylindricalRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	slab := box(0, 0, 0, 10, 10, 4)
	for _, tc := range []struct {
		name  string
		axis  math.Vector3
		r     float64
		depth float64
		base  math.Point3
	}{
		{"bad radius", math.V3(0, 0, 1), 0, 1, math.P3(5, 5, 0)},
		{"bad depth", math.V3(0, 0, 1), 1, 0, math.P3(5, 5, 0)},
		{"zero axis", math.V3(0, 0, 0), 1, 1, math.P3(5, 5, 0)},
		{"no entry face", math.V3(1, 0, 0), 1, 1, math.P3(5, 5, 0)},
		{"does not fit entry face", math.V3(0, 0, 1), 6, 1, math.P3(5, 5, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := brep.CutBlindCylindricalHole(slab, tc.base, tc.axis, tc.r, tc.depth)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
