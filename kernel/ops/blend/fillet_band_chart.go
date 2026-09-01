// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The fillet band's OWN metric chart, and the ideal patch box drawn in it.
//
// A constant-radius blend on a straight edge sweeps a cylinder; the band the fillet actually keeps is
// a rectangle in that cylinder's (u, v) chart — u the distance along the axis from the c0 section, v
// the section angle measured from host A's contact toward host B's. cylinderFace emits exactly the
// four sides of that rectangle: the A-tangent line (v = 0), the two end sections (u = 0, u = uMax) and
// the B-tangent line (v = vMax).
//
// This file owns the chart alone — no obstacles, no topology. The imprint walk
// (fillet_band_imprint.go) draws the obstacles' cuts in it, and the router
// (fillet_band_imprint_faces.go) turns the surviving region's boundary back into loops. Splitting the
// chart out keeps the walk free of trigonometry and the chart free of body faces.

// bandChart maps between 3D and the fillet band's own metric chart. It is built from the fillet's own
// solved geometry (the two section centres and their tangent points), so its four box corners are the
// tangent points themselves — bit for bit, via bandBoxCorner — and a rebuilt loop welds to whatever the
// untouched neighbours still carry.
type bandChart struct {
	origin math.Point3  // the c0 section centre: (u, v) = (0, 0) is its host-A tangent point
	axis   math.Vector3 // unit, c0 centre -> c1 centre
	e0, e1 math.Vector3 // orthonormal section frame; e0 points at host A's contact, e1 at v = +π/2
	radius float64
	uMax   float64 // |c1 centre − c0 centre|
	vMax   float64 // the section sweep from host A's contact to host B's, in (0, π)
	// ta0, ta1, tb0, tb1 are the fillet's own tangent points at the four box corners, kept verbatim so
	// a rebuilt loop's corners are IDENTICAL to the ones the untouched faces still carry.
	ta0, ta1, tb0, tb1 math.Point3
}

// newBandChart builds the chart of a constant-radius blend from its two solved sections. ok=false when
// the sections are degenerate (coincident centres, a tangent point off the section circle, or a sweep
// that is not a proper arc), i.e. when there is no rectangle to draw the imprint in.
func newBandChart(ef edgeFillet) (bandChart, bool) {
	c := bandChart{origin: ef.c0.cen, radius: ef.cyl.Radius,
		ta0: ef.c0.ta, ta1: ef.c1.ta, tb0: ef.c0.tb, tb1: ef.c1.tb}
	span := ef.c0.cen.VectorTo(ef.c1.cen)
	c.uMax = float64(span.Length())
	if c.uMax <= 0 || c.radius <= 0 {
		return bandChart{}, false
	}
	c.axis = span.Scale(math.Scalar(1 / c.uMax))
	if !bandChartFrame(&c, ef) {
		return bandChart{}, false
	}
	return c, true
}

// bandChartFrame fixes the section frame so v runs from host A's contact (v = 0) to host B's (v =
// vMax) the short way round, and checks both tangent points really sit on the section circle.
func bandChartFrame(c *bandChart, ef edgeFillet) bool {
	ra := ef.c0.cen.VectorTo(ef.c0.ta)
	rb := ef.c0.cen.VectorTo(ef.c0.tb)
	if !bandRadialOK(ra, c.radius) || !bandRadialOK(rb, c.radius) {
		return false
	}
	c.e0 = ra.Scale(math.Scalar(1 / c.radius))
	c.e1 = c.axis.Cross(c.e0)
	v := stdmath.Atan2(float64(rb.Dot(c.e1)), float64(rb.Dot(c.e0)))
	if v < 0 {
		c.e1, v = c.e1.Negate(), -v
	}
	c.vMax = v
	return v > bandChartMinSweep && v < stdmath.Pi
}

// bandChartMinSweep is the smallest section sweep the chart accepts. Below it the "rectangle" is a
// sliver and the imprint's cell classification has no interior to probe; such a blend is declined
// rather than imprinted on a degenerate box.
const bandChartMinSweep = 1e-6

// bandRadialOK reports whether a tangent point's radial really has the band's radius (relative, so it
// is scale-invariant per ADR-0042).
func bandRadialOK(r math.Vector3, radius float64) bool {
	return stdmath.Abs(float64(r.Length())-radius) <= 1e-9*radius
}

// bandPointAt is the band point at chart coordinates (u, v).
func (c bandChart) bandPointAt(u, v float64) math.Point3 {
	radial := c.e0.Scale(math.Scalar(c.radius * stdmath.Cos(v))).Add(c.e1.Scale(math.Scalar(c.radius * stdmath.Sin(v))))
	return c.origin.TranslateBy(c.axis.Scale(math.Scalar(u)).Add(radial))
}

// bandBoxCorner is bandPointAt with the four ideal-box corners answered by the fillet's OWN tangent
// points instead of by trigonometry. Everything the imprint emits at a box corner therefore welds to
// the untouched neighbour faces bit for bit, which is what keeps a face the walk does not rebuild
// byte-identical to today.
func (c bandChart) bandBoxCorner(u, v float64) math.Point3 {
	atU0, atUMax := u == 0, u == c.uMax
	switch {
	case v == 0 && atU0:
		return c.ta0
	case v == 0 && atUMax:
		return c.ta1
	case v == c.vMax && atU0:
		return c.tb0
	case v == c.vMax && atUMax:
		return c.tb1
	}
	return c.bandPointAt(u, v)
}

// bandAxialCut is the chart u of the plane through q whose normal is the band axis — where a plane
// PERPENDICULAR to the axis cuts the band, as a section circle.
func (c bandChart) bandAxialCut(q math.Point3) float64 {
	return float64(c.origin.VectorTo(q).Dot(c.axis))
}

// bandRulingCuts is the chart v (0, 1 or 2 values) at which a plane PARALLEL to the band axis cuts it,
// as rulings: the plane through q with unit normal n meets the section circle where
// cos(v − φ) = d/r, with φ the normal's own angle in the section frame.
func (c bandChart) bandRulingCuts(q math.Point3, n math.Vector3) []float64 {
	d := float64(c.origin.VectorTo(q).Dot(n)) / c.radius
	if stdmath.Abs(d) > 1 {
		return nil
	}
	phi := stdmath.Atan2(float64(n.Dot(c.e1)), float64(n.Dot(c.e0)))
	a := stdmath.Acos(stdmath.Max(-1, stdmath.Min(1, d)))
	return []float64{phi + a, phi - a}
}
