// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/brep"
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// countConeFaces returns how many of a body's faces are true cone faces.
func countConeFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			n++
		}
	}
	return n
}

// A through countersink: a 90° cone widening to Ø4 at the surface, narrowing to a Ø2 bore
// through a 10×10×6 slab. Valid watertight solid with a cone wall + a cylinder bore wall.
func TestCutCountersinkThrough(t *testing.T) {
	const ha = stdmath.Pi / 4 // 90° included → 45° half-angle, tan = 1
	d, err := brep.CutCountersinkHole(box(0, 0, 0, 10, 10, 6), math.P3(5, 5, 6), math.V3(0, 0, -1), 1, 0, 2, ha, true)
	if err != nil {
		t.Fatalf("CutCountersinkHole: %v", err)
	}
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("countersunk slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(d); len(open) != 0 {
		t.Fatalf("countersink has %d boundary edges, want 0 (watertight)", len(open))
	}
	if n := countConeFaces(d); n != 1 {
		t.Errorf("countersink has %d cone faces, want 1 (the sink wall)", n)
	}
	if n := countCylFaces(d); n != 1 {
		t.Errorf("countersink has %d cylinder faces, want 1 (the bore wall)", n)
	}
	// Removed = a frustum (R=2 → r=1 over depthCS=1) + the bore (Ø2 over the remaining 5).
	const depthCS = (2.0 - 1.0) / 1.0
	frustum := stdmath.Pi * depthCS / 3 * (2*2 + 2*1 + 1*1)
	bore := stdmath.Pi * 1 * 1 * (6 - depthCS)
	want := frustum + bore
	removed := 10.0*10.0*6.0 - vol(d)
	if removed <= 0 || removed > want+1e-9 || (want-removed)/want > 0.04 {
		t.Errorf("removed volume = %g, want a hair under %g (frustum + bore)", removed, want)
	}
}
