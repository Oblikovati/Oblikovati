// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// countCylFaces returns how many of a body's faces are true cylinder faces.
func countCylFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// A through counterbore: Ø4 recess 1 deep stepping to a Ø2 bore through an 8×8×4 slab.
// Valid watertight solid with two cylinder walls (recess + bore) and an annular shoulder.
func TestCutCounterboreThrough(t *testing.T) {
	d, err := brep.CutCounterboreHole(box(0, 0, 0, 8, 8, 4), math.P3(4, 4, 4), math.V3(0, 0, -1), 1, 0, 2, 1, true)
	if err != nil {
		t.Fatalf("CutCounterboreHole: %v", err)
	}
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("counterbored slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(d); len(open) != 0 {
		t.Fatalf("counterbore has %d boundary edges, want 0 (watertight)", len(open))
	}
	if n := countCylFaces(d); n != 2 {
		t.Errorf("counterbore has %d cylinder faces, want 2 (recess + bore wall)", n)
	}
	// Removed = recess (Ø4×1) + bore (Ø2×3 below the shoulder), inscribed (a hair under).
	const want = stdmath.Pi*2*2*1 + stdmath.Pi*1*1*3
	removed := 8.0*8.0*4.0 - vol(d)
	if removed <= 0 || removed > want+1e-9 || (want-removed)/want > 0.04 {
		t.Errorf("removed volume = %g, want a hair under %g", removed, want)
	}
}

// A blind counterbore stops inside the part: it adds a flat bore bottom too.
func TestCutCounterboreBlind(t *testing.T) {
	d, err := brep.CutCounterboreHole(box(0, 0, 0, 8, 8, 6), math.P3(4, 4, 6), math.V3(0, 0, -1), 1, 2, 2, 1, false)
	if err != nil {
		t.Fatalf("CutCounterboreHole (blind): %v", err)
	}
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("blind counterbore is not a valid solid: %+v", r)
	}
	if n := countCylFaces(d); n != 2 {
		t.Errorf("blind counterbore has %d cylinder faces, want 2", n)
	}
}
