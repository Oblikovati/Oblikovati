// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"
)

// TestNopUsbAx2CSG models the NopSCADlib dual USB-A shell (vitamins/pcb.scad usb_Ax2 → usb_A with
// bar=3.4). The cross-section is difference(square([h,w]), two stacked socket squares separated by
// a central bar): a rectangular block with TWO rectangular bores. Extruded over the length it is a
// multi-hole tube — the boolean must subtract two disjoint bores from one block and leave the thin
// 0.4 mm flange walls and the 3.4 mm bar between them. Multi-hole faces are a known tessellation/
// boolean trouble spot, so this exercises the 2-hole through-bore path end to end. Dimensions in cm.
//
// usb_Ax2 = usb_A(h=15.6, bar=3.4, l=17, flange_t=0.4, w=13.25). socket_h = (h−2·t−bar)/2 = 5.7;
// each socket = [socket_h, w−2·t] centred at x = ±(bar/2 + socket_h/2).
func TestNopUsbAx2CSG(t *testing.T) {
	t.Parallel()
	const (
		l        = 1.7                       // 17 mm (extrusion axis = z)
		w        = 1.325                     // 13.25 mm (y)
		h        = 1.56                      // 15.6 mm (x)
		flangeT  = 0.04                      // 0.4 mm
		bar      = 0.34                      // 3.4 mm central bar
		socketH  = (h - 2*flangeT - bar) / 2 // 0.57
		socketW  = w - 2*flangeT             // 1.245
		socketCx = bar/2 + socketH/2         // 0.455
	)
	outer := rectAtPoints(0, 0, h, w)
	upper := rectAtPoints(socketCx, 0, socketH, socketW)
	lower := rectAtPoints(-socketCx, 0, socketH, socketW)

	body := prismBody(outer, 0, l, "usba-shell")
	body = cutOrFatal(t, body, prismBody(upper, -0.05, l+0.05, "usba-upper"), "usba upper socket")
	body = cutOrFatal(t, body, prismBody(lower, -0.05, l+0.05, "usba-lower"), "usba lower socket")

	requireValidNopSolid(t, "usba", body)

	// Volume = (outer − two sockets) · l.
	want := (nopPolygonArea(outer) - nopPolygonArea(upper) - nopPolygonArea(lower)) * l
	if got := vol(body); stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("usb_Ax2 shell volume = %.6f cm^3, want ~%.6f (two-bore multi-hole boolean)", got, want)
	}
}
