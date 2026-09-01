// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// offsetConvexCCW grows a CCW convex polygon outward by d with mitred (sharp) corners: each
// edge is pushed out along its outward normal, and every vertex is reset to the intersection
// of its two neighbouring offset edges. NopSCADlib's offset() rounds corners, but a mitred
// offset is the exact wall an Inventor extrude of a sharp-cornered offset profile produces; the
// unit layer checks the BOOLEAN (cap + walls), not the corner blend, so it verifies the built
// solid against its own shoelace area (see TestNopHdmiCSG).
func offsetConvexCCW(pts []math.Point3, d float64) []math.Point3 {
	n := len(pts)
	// One offset line per edge i→i+1: a point on it plus the (un-normalised) edge direction.
	op := make([]math.Point3, n)
	od := make([]math.Point3, n)
	for i := range n {
		a, b := pts[i], pts[(i+1)%n]
		dx, dy := b.X-a.X, b.Y-a.Y
		l := stdmath.Hypot(dx, dy)
		// Outward normal for a CCW polygon is to the right of the edge direction: (dy,-dx).
		nx, ny := dy/l, -dx/l
		op[i] = math.P3(a.X+nx*d, a.Y+ny*d, 0)
		od[i] = math.P3(dx, dy, 0)
	}
	res := make([]math.Point3, n)
	for i := range n {
		j := (i - 1 + n) % n // vertex i is shared by edge j (j→i) and edge i (i→i+1)
		res[i] = lineLineIntersect(op[j], od[j], op[i], od[i])
	}
	return res
}

// lineLineIntersect returns the intersection of lines p+t·r and q+u·s (r,s non-parallel).
func lineLineIntersect(p, r, q, s math.Point3) math.Point3 {
	denom := r.X*s.Y - r.Y*s.X
	t := ((q.X-p.X)*s.Y - (q.Y-p.Y)*s.X) / denom
	return math.P3(p.X+t*r.X, p.Y+t*r.Y, 0)
}

// TestNopHdmiCSG models the NopSCADlib HDMI socket metal shell (vitamins/pcb.scad hdmi(),
// hdmi_full). Its cross-section D() is the convex hull of two stacked rectangles ⇒ a keystone
// hexagon; the shell is difference(offset(t) D(), D()) extruded along the depth, plus a 1 mm
// solid end flange (offset(t) D()). Built here as a solid offset-keystone block with a BLIND
// bore cut that stops 1 mm short of one end — a thin-wall (t=0.5 mm) blind-pocket boolean where
// the cap face and the four bore walls collide at the cut limit. Dimensions in cm (mm/10).
//
// hdmi_full = [l=12, iw1=14, iw2=10, ih1=3, ih2=4.5, h=6.5, t=0.5] (mm).
func TestNopHdmiCSG(t *testing.T) {
	t.Parallel()
	// Keystone hexagon D(), CCW, in cm. From the hull of the wide top rect (±0.7, y∈[0.3,0.6])
	// and the narrower lower rect (±0.5, y∈[0.15,0.6]).
	inner := []math.Point3{
		math.P3(-0.7, 0.6, 0), math.P3(-0.7, 0.3, 0), math.P3(-0.5, 0.15, 0),
		math.P3(0.5, 0.15, 0), math.P3(0.7, 0.3, 0), math.P3(0.7, 0.6, 0),
	}
	const wallT = 0.05 // t = 0.5 mm
	const depth = 1.2  // l = 12 mm
	const capT = 0.1   // 1 mm solid flange/cap at the bottom end
	outer := offsetConvexCCW(inner, wallT)

	body := prismBody(outer, 0, depth, "hdmi-shell")
	bore := prismBody(inner, capT, depth+0.05, "hdmi-bore") // overshoot the open end, stop at capT
	body = cutOrFatal(t, body, bore, "hdmi bore")

	requireValidNopSolid(t, "hdmi", body)

	// Volume = outer·depth − inner·(depth−capT): full block minus the blind bore.
	want := nopPolygonArea(outer)*depth - nopPolygonArea(inner)*(depth-capT)
	if got := vol(body); stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("hdmi shell volume = %.6f cm^3, want ~%.6f (thin-wall blind-bore boolean)", got, want)
	}
}
