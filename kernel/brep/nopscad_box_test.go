// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestNopBoxTrayCSG re-models a printed box base (NopSCADlib printed/box.scad) the
// OpenSCAD-CSG way: a solid outer cube minus an inner cube raised by the floor thickness and
// over-shooting the top, leaving a tray with a floor and four walls of thickness t, open on
// top. The exact planar boolean cuts it; the wall is the difference of two analytic boxes.
//
// Reference: NopSCADlib/printed/box.scad (box_base — a shelled rectangular tray).
func TestNopBoxTrayCSG(t *testing.T) {
	t.Parallel()
	const w, d, h, wall = 4.0, 3.0, 2.0, 0.2 // outer dims + wall thickness (cm)

	outer := box(0, 0, 0, w, d, h)
	// Inner void: shrunk by wall on the four sides, floor of thickness wall at the bottom,
	// and over-shooting the top so the tray is open (the OpenSCAD epsilon trick).
	inner := box(wall, wall, wall, w-2*wall, d-2*wall, h)
	tray, err := ops.Boolean(ops.Cut, outer, inner)
	if err != nil {
		t.Fatalf("Boolean(Cut tray): %v", err)
	}

	if r := ops.Validate(tray); !r.Valid || !tray.IsSolid() {
		t.Fatalf("tray is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(tray); len(open) != 0 {
		t.Fatalf("tray has %d boundary edges, want 0 (watertight)", len(open))
	}

	got := vol(tray)
	want := w*d*h - (w-2*wall)*(d-2*wall)*(h-wall)
	if rel := stdmath.Abs(got-want) / want; rel > 1e-3 {
		t.Errorf("tray volume = %.5f, want %.5f (rel %.4f)", got, want, rel)
	}
}
