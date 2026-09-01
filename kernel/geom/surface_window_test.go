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
	t.Parallel()
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

// TestSurfaceWindowTightHasNoPad: SurfaceWindowTight clips an unbounded direction to EXACTLY the box
// projection (no outward pad), so a cylinder side windowed to its own body box reproduces the cap band
// bit-for-bit — the exactness the near-pinch imprint weld needs (#1818). The padded SurfaceWindow is wider.
func TestSurfaceWindowTightHasNoPad(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	box := math.NewBox(math.P3(-2, -2, 1), math.P3(2, 2, 5)) // axis-aligned: axial span z ∈ [1,5] exactly
	tight := SurfaceWindowTight(cyl, box)
	if stdmath.Abs(tight.VMin-1) > 1e-12 || stdmath.Abs(tight.VMax-5) > 1e-12 {
		t.Errorf("tight axial window [%g,%g] must equal the exact cap band [1,5]", tight.VMin, tight.VMax)
	}
	padded := SurfaceWindow(cyl, box)
	if !(padded.VMin < tight.VMin && padded.VMax > tight.VMax) {
		t.Errorf("padded window [%g,%g] must be strictly wider than the tight [%g,%g]",
			padded.VMin, padded.VMax, tight.VMin, tight.VMax)
	}
}

// TestSurfaceWindowUnboundedPlaneProjectsBothDirections: a plane is unbounded in BOTH directions, so both
// are clipped to the box projection — neither stays the infinite domain.
func TestSurfaceWindowUnboundedPlaneProjectsBothDirections(t *testing.T) {
	t.Parallel()
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	box := math.NewBox(math.P3(-4, -2, -1), math.P3(4, 2, 1))
	win := SurfaceWindow(pl, box)
	for _, v := range []float64{win.UMin, win.UMax, win.VMin, win.VMax} {
		if stdmath.IsInf(v, 0) {
			t.Fatalf("an unbounded plane direction must be clipped to the box, got infinite bound in %+v", win)
		}
	}
}
