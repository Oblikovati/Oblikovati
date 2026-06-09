// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

// TestNopUsbCCSG models the NopSCADlib USB-C connector's silver shell (vitamins/pcb.scad usb_C()).
// The cross-section O() is a rounded_square([w,h], r); the shell is linear_extrude(l)
// difference(O(), offset(-t) O()) — a ROUNDED-rect tube of wall t — plus a solid rounded-rect plug
// (linear_extrude(2.51) O()) filling the back. Net solid: a rounded-rect block of length l with a
// rounded-rect bore that stops short of the back (a blind bore, depth l−plug). What it adds over the
// rectangular shells (HDMI/rj45) is that BOTH the outer wall and the bore wall are rounded — the
// blind-bore boolean runs on two concentric faceted-corner loops (offset(-t) shrinks the corner
// radius from r to r−t), so the four bore-wall arcs and the cap face all meet at the cut limit.
// Dimensions in cm (mm/10).
//
// usb_C = [l=7.35, w=8.94, h=3.26, t=0.4, plug=2.51], r = h/2 − 0.5 mm.
func TestNopUsbCCSG(t *testing.T) {
	const (
		l       = 0.735 // length 7.35 mm (extrusion axis = z)
		w       = 0.894 // 8.94 mm
		h       = 0.326 // 3.26 mm
		wall    = 0.04  // t = 0.4 mm
		plugLen = 0.251 // 2.51 mm solid plug at the back
		r       = h/2 - 0.05
	)
	const steps = 6 // arc segments per rounded corner
	outer := roundedRectPoints(w, h, r, steps)
	inner := roundedRectPoints(w-2*wall, h-2*wall, r-wall, steps)

	body := prismBody(outer, 0, l, "usbc-shell")
	bore := prismBody(inner, plugLen, l+0.05, "usbc-bore") // open at the front, stop at the plug
	body = cutOrFatal(t, body, bore, "usbc bore")

	requireValidNopSolid(t, "usbc", body)

	// Blind bore: full rounded block minus the bore over its open depth (l − plug).
	want := nopPolygonArea(outer)*l - nopPolygonArea(inner)*(l-plugLen)
	if got := vol(body); stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("usb_C shell volume = %.6f cm^3, want ~%.6f (rounded-rect blind-bore boolean)", got, want)
	}
}
