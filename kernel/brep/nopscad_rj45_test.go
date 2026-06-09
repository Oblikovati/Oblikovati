// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestNopRj45CSG models the NopSCADlib RJ45 Ethernet connector's silver metal shell
// (vitamins/pcb.scad rj45()). The shell is linear_extrude(l) of difference(square([h,w]),
// mouth-square): a rectangular outer tube with a rectangular mouth bored through it. What makes
// it distinct from the HDMI shell (TestNopHdmiCSG, a symmetric keystone blind bore) is that the
// mouth is OFFSET toward one wall — it leaves a thin 0.75 mm wall on the +x side and a thick
// (tab) wall on the −x side — so the through-bore boolean must cut an asymmetric annulus where
// one wall is an order of magnitude thinner than the others. Dimensions in cm (mm/10), matching
// the other nopscad ports.
//
// rj45 = [l=21, w=16, h=13.5, plug_h=6.8, plug_z=4, tab_z=0.8, plug_w=12] (mm).
// mouth = plug_z + plug_h - tab_z = 10 mm. Mouth centred at x = h/2 - tab_z - mouth/2.
func TestNopRj45CSG(t *testing.T) {
	const (
		l     = 2.1  // length 21 mm (extrusion axis = z)
		w     = 1.6  // width 16 mm  (y)
		h     = 1.35 // height 13.5 mm (x)
		mouth = 1.0  // plug_z + plug_h - tab_z
		plugW = 1.2  // plug_w
		tabZ  = 0.08 // tab_z
		kerf  = 0.01 // the +0.1 mm clearance NopSCADlib adds to the mouth cut
	)
	// Outer square cross-section, CCW, centred.
	outer := []math.Point3{
		math.P3(-h/2, -w/2, 0), math.P3(h/2, -w/2, 0),
		math.P3(h/2, w/2, 0), math.P3(-h/2, w/2, 0),
	}
	// Mouth bore: (mouth+kerf) tall in x, (plugW+kerf) wide in y, centred at cx (offset toward +x).
	cx := h/2 - tabZ - mouth/2
	mx, my := (mouth+kerf)/2, (plugW+kerf)/2
	inner := []math.Point3{
		math.P3(cx-mx, -my, 0), math.P3(cx+mx, -my, 0),
		math.P3(cx+mx, my, 0), math.P3(cx-mx, my, 0),
	}

	body := prismBody(outer, 0, l, "rj45-shell")
	bore := prismBody(inner, -0.05, l+0.05, "rj45-mouth") // overshoot both ends → through-bore
	body = cutOrFatal(t, body, bore, "rj45 mouth")

	requireValidNopSolid(t, "rj45", body)

	// Volume = (outer − mouth) · l: a clean asymmetric rectangular tube.
	want := (nopPolygonArea(outer) - nopPolygonArea(inner)) * l
	if got := vol(body); stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("rj45 shell volume = %.6f cm^3, want ~%.6f (asymmetric through-bore boolean)", got, want)
	}
}
