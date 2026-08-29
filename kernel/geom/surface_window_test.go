// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// SurfaceWindow (ADR-0058 phase 3) clips a surface's marching window to a bounding box: a periodic or
// bounded direction keeps its whole domain; an unbounded direction is clipped to the box's projection.

// TestSurfaceWindowKeepsPeriodicClipsAxial: a cylinder's angular U is periodic (kept [0,2π]); its
// unbounded axial V is clipped to the box's axial span (with a small pad so a boundary crossing survives).
func TestSurfaceWindowKeepsPeriodicClipsAxial(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	box := math.NewBox(math.P3(-3, -3, 1), math.P3(3, 3, 5)) // axial span z ∈ [1,5]
	win := SurfaceWindow(cyl, box)

	if win.UMin != 0 || stdmath.Abs(win.UMax-2*stdmath.Pi) > 1e-9 {
		t.Errorf("periodic U must stay the whole [0,2π] domain, got [%g,%g]", win.UMin, win.UMax)
	}
	// V is the axial projection [1,5] padded outward; the [1,5] span must sit strictly inside the window.
	if !(win.VMin < 1 && win.VMax > 5) {
		t.Errorf("axial V window [%g,%g] must contain the box span [1,5] with pad", win.VMin, win.VMax)
	}
	if win.VMin < -1 || win.VMax > 7 { // pad is 5% of the span (=0.2), nowhere near this loose bound
		t.Errorf("axial V window [%g,%g] clipped far too wide for a span-4 box", win.VMin, win.VMax)
	}
}

// TestSurfaceWindowUnboundedPlaneProjectsBothDirections: a plane is unbounded in BOTH directions, so both
// are clipped to the box projection — neither stays the infinite domain.
func TestSurfaceWindowUnboundedPlaneProjectsBothDirections(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	box := math.NewBox(math.P3(-4, -2, -1), math.P3(4, 2, 1))
	win := SurfaceWindow(pl, box)
	for _, v := range []float64{win.UMin, win.UMax, win.VMin, win.VMax} {
		if stdmath.IsInf(v, 0) {
			t.Fatalf("an unbounded plane direction must be clipped to the box, got infinite bound in %+v", win)
		}
	}
}
