// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Self-intersection detection — the COPLANAR branch (M48 #2223 split of self_intersect.go). When two
// triangles lie in the same plane the Möller straddle test has no depth to measure, so they are
// projected onto their shared plane's orthonormal axes and tested by separating axes + Sutherland–
// Hodgman clipping; the overlap is classified by shared AREA (a ratio of the smaller triangle), and
// the witness is a point INSIDE the shared region so the caller's shared-boundary filter can tell
// legitimate edge contact from real interpenetration (#2074/#2077).

// coplanarOverlap projects both triangles onto their shared plane's dominant axes, tests 2D
// overlap by separating axes, and reports a witness INSIDE the shared region.
//
// The witness has to be truthful. Returning t1's centre instead (as this did until
// Oblikovati#2074) put the witness far from where the triangles actually met, and the caller
// filters legitimate contact by asking whether the witness lies on the two faces' shared
// boundary — a centre never does. Two coplanar faces meeting along a shared edge, which every
// sheet-metal wall makes where its end cap meets the sheet's side, were therefore reported as
// interpenetrating on every part that had one.
func coplanarOverlap(t1, t2 [3]math.Point3, n math.Vector3) (math.Point3, float64, bool) {
	u, v := planeAxes(n)
	a := projectTriangle(t1, u, v)
	b := projectTriangle(t2, u, v)
	if separated(a, b) || separated(b, a) {
		return math.Point3{}, 0, false
	}
	region, flat := clipToTriangle(t2, b, a)
	if len(region) < 3 || degenerateShare(flat, a, b) {
		return math.Point3{}, 0, false
	}
	// The 2D coordinates come from UNIT plane axes, so the shoelace area is a true area.
	return centroidOf(region), polygonArea2D(flat), true // centroidOf is shared with the sphere-cap mesher
}

// coplanarShareRatio is the least share of the smaller triangle two coplanar triangles must
// overlap by to count as interpenetrating. It is a RATIO of areas, so it carries no model
// scale. Contact along an edge overlaps by ~1e-16 of a triangle (rounding alone) and a real
// overlap by ~1e-2 or more, so every threshold between about 1e-12 and 1e-6 gives the same
// answer — see TestCoplanarShareThresholdHasAPlateau.
const coplanarShareRatio = 1e-9 // tol:numeric — a ratio of two areas, so it carries no model scale

// degenerateShare reports whether the clipped region is contact rather than overlap: two
// coplanar triangles meeting along an edge or at a corner share a region of no area, which is
// how every face meets its neighbour and is not an interpenetration.
func degenerateShare(region [][2]float64, a, b [3][2]float64) bool {
	smaller := stdmath.Min(triArea2D(a), triArea2D(b))
	if smaller <= 0 {
		return true // a degenerate triangle cannot overlap anything
	}
	return polygonArea2D(region)/smaller < coplanarShareRatio
}

// triArea2D is the projected triangle's area.
func triArea2D(t [3][2]float64) float64 {
	return polygonArea2D([][2]float64{t[0], t[1], t[2]})
}

// polygonArea2D is the shoelace area, unsigned so winding does not matter.
func polygonArea2D(p [][2]float64) float64 {
	sum := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		sum += p[i][0]*p[j][1] - p[j][0]*p[i][1]
	}
	return stdmath.Abs(sum) / 2
}

// clipToTriangle clips triangle t (with projection tp) against clip's three edge half-planes,
// returning the shared region's corners in 3D and their projections. It is Sutherland–Hodgman
// run on the projected coordinates while interpolating the 3D points, so the corners land on
// the real surface and the projections stay available to measure the region's area.
func clipToTriangle(t [3]math.Point3, tp, clip [3][2]float64) ([]math.Point3, [][2]float64) {
	poly := []math.Point3{t[0], t[1], t[2]}
	proj := [][2]float64{tp[0], tp[1], tp[2]}
	for i := 0; i < 3 && len(poly) > 0; i++ {
		j := (i + 1) % 3
		nx, ny := clip[j][1]-clip[i][1], clip[i][0]-clip[j][0]
		inside := func(p [2]float64) float64 {
			return nx*(p[0]-clip[i][0]) + ny*(p[1]-clip[i][1])
		}
		poly, proj = clipHalfPlane(poly, proj, inside, insideSign(clip, nx, ny, i))
	}
	return poly, proj
}

// insideSign reports which side of the clip edge the triangle's own third corner is on, so the
// clip works for either winding.
func insideSign(clip [3][2]float64, nx, ny float64, i int) float64 {
	third := clip[(i+2)%3]
	if nx*(third[0]-clip[i][0])+ny*(third[1]-clip[i][1]) < 0 {
		return -1
	}
	return 1
}

// clipHalfPlane keeps the part of the polygon on the inside of one half-plane, interpolating
// both the 3D point and its projection at each crossing.
func clipHalfPlane(poly []math.Point3, proj [][2]float64, dist func([2]float64) float64,
	sign float64) ([]math.Point3, [][2]float64) {
	var outP []math.Point3
	var outQ [][2]float64
	for i := range poly {
		j := (i + 1) % len(poly)
		di, dj := sign*dist(proj[i]), sign*dist(proj[j])
		if di >= 0 {
			outP, outQ = append(outP, poly[i]), append(outQ, proj[i])
		}
		if (di >= 0) == (dj >= 0) || di == dj {
			continue
		}
		f := di / (di - dj)
		outP = append(outP, poly[i].TranslateBy(poly[i].VectorTo(poly[j]).Scale(math.Scalar(f))))
		outQ = append(outQ, [2]float64{
			proj[i][0] + f*(proj[j][0]-proj[i][0]),
			proj[i][1] + f*(proj[j][1]-proj[i][1]),
		})
	}
	return outP, outQ
}

// planeAxes picks two in-plane axes for the dominant-normal projection.
// planeAxes returns an ORTHONORMAL pair spanning the plane with normal n, so coordinates projected
// on them are true lengths and 2D areas are true areas.
//
// They used to be the raw cross products n×ref and n×(n×ref), whose magnitudes are |n| and |n|² —
// so the projected space carried the triangle's own area as a scale factor, and carried a DIFFERENT
// one on each axis. #2075 normalised every other comparison in this file and left this branch
// alone, which is why the note on selfIntersectEps claimed unit axes that did not exist. Anything
// measured in that space (a length against selfIntersectEps, an area against the faceting
// allowance) meant a different thing for every pair of triangles (#2077).
func planeAxes(n math.Vector3) (math.Vector3, math.Vector3) {
	ref := math.V3(1, 0, 0)
	if stdmath.Abs(float64(n.X)) > stdmath.Abs(float64(n.Y)) && stdmath.Abs(float64(n.X)) > stdmath.Abs(float64(n.Z)) {
		ref = math.V3(0, 1, 0)
	}
	u, v := n.Cross(ref), n.Cross(n.Cross(ref))
	// Divide by the magnitudes explicitly. UnitVector3FromVector guards zero length with an ABSOLUTE
	// 1e-9, and |n| ~ 2·area underflows that on small parts — the trap #2075 fell into.
	lu, lv := float64(u.Length()), float64(v.Length())
	if lu == 0 || lv == 0 {
		return math.Vector3{}, math.Vector3{} // degenerate triangle: everything projects to a point
	}
	return u.Scale(math.Scalar(1 / lu)), v.Scale(math.Scalar(1 / lv))
}

func projectTriangle(t [3]math.Point3, u, v math.Vector3) [3][2]float64 {
	var out [3][2]float64
	for i, p := range t {
		out[i] = [2]float64{float64(u.Dot(p.AsVector())), float64(v.Dot(p.AsVector()))}
	}
	return out
}

// separated reports whether any edge normal of a separates the projections of
// a and b — disjoint intervals on the axis mean no overlap (plain SAT, winding
// independent).
func separated(a, b [3][2]float64) bool {
	for i := range 3 {
		j := (i + 1) % 3
		nx, ny := a[j][1]-a[i][1], a[i][0]-a[j][0] // edge normal
		aMin, aMax := axisInterval(a, nx, ny)
		bMin, bMax := axisInterval(b, nx, ny)
		if aMax < bMin-selfIntersectEps || bMax < aMin-selfIntersectEps {
			return true
		}
	}
	return false
}

// axisInterval projects the triangle onto the (nx, ny) axis.
func axisInterval(t [3][2]float64, nx, ny float64) (float64, float64) {
	lo, hi := 1e308, -1e308
	for _, p := range t {
		v := nx*p[0] + ny*p[1]
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}
