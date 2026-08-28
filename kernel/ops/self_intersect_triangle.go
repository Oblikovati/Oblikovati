// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Self-intersection detection — the NARROW PHASE (M48 #2223 split of self_intersect.go). The Möller
// interval test for one triangle pair: T1 must straddle T2's plane and vice versa, their intervals on
// the planes' intersection line must overlap, and the crossing must be deeper than the faceting
// allowance. It also classifies the hit (crossStraddle vs crossCoplanar vs crossNone) and hands the
// coplanar branch to self_intersect_coplanar.go. Every comparison is in TRUE length units against a
// tolerance scaled by the smaller triangle (#2075).

// crossingThickness is how deep the shallower of the two triangles pokes through the other's
// plane — the honest size of a straddling crossing, and what the faceting allowance is compared
// against (#2077).
func crossingThickness(a, b [3]math.Point3) float64 {
	return stdmath.Min(pokeDepth(a, b), pokeDepth(b, a))
}

// pokeDepth is how far t reaches past the plane of other, on the side it reaches less far.
func pokeDepth(t, other [3]math.Point3) float64 {
	n, d := triPlaneEq(other)
	scale := float64(n.Length())
	if scale == 0 {
		return 0 // a degenerate triangle has no plane to poke through
	}
	hi, lo := stdmath.Inf(-1), stdmath.Inf(1)
	for _, p := range t {
		s := (float64(n.Dot(p.AsVector())) - d) / scale
		hi, lo = stdmath.Max(hi, s), stdmath.Min(lo, s)
	}
	return stdmath.Min(hi, -lo)
}

// selfIntersectEps keeps grazing contact from separating two coplanar triangles that in fact
// overlap. It guards ONLY the 2D separating-axis test in coplanarOverlap, whose coordinates come
// from the ORTHONORMAL axes planeAxes now returns and so are true lengths. They were not until
// #2077 — see planeAxes — so this really was a scaled residual for the whole of #2075's life.
const selfIntersectEps = 1e-9 // tol:numeric — a true length on unit plane axes, not a scaled residual

// crossRatio is how deep, and how far along, two triangles must cross before the crossing counts
// as interpenetration rather than contact. It multiplies the smaller triangle's own size, so it
// carries no model scale (ADR-0042) — see Oblikovati#2075.
const crossRatio = 1e-9 // tol:numeric — dimensionless; scaled by the local triangle size at use

// trianglesIntersect is the Möller interval test: T1 must straddle T2's plane
// and vice versa, and their intervals on the planes' intersection line must
// overlap. Coplanar triangles are handled by 2D overlap on their shared plane.
//
// Every comparison is made in TRUE length units, against a tolerance scaled by the smaller
// triangle. Until Oblikovati#2075 the plane distances were left unnormalised (n·p−d, inflated by
// |n| ≈ 2·area) and the interval was projected on the unnormalised n1×n2, so a fixed 1e-9 meant a
// different thing for every pair of triangles — and for small ones it meant nothing at all. Two
// faces that merely TOUCH along a line then read as interpenetrating, because the overlap interval
// is a single point whose inflated length still cleared the threshold. That is what made a mitered
// sheet-metal corner report a self-intersection where the gap cut abuts the wall it cuts.
func trianglesIntersect(t1, t2 [3]math.Point3) (math.Point3, bool) {
	p, kind := triangleCrossing(t1, t2, 0)
	return p, kind != crossNone
}

// crossKind says WHICH branch of the Möller test produced a hit. The two need different evidence:
// a straddling crossing is judged by how DEEP it is, a coplanar one by how much AREA it shares —
// a coplanar pair has no depth at all, so a depth gate would silently erase every coplanar
// overlap (#2077).
type crossKind int

const (
	crossNone crossKind = iota
	crossStraddle
	crossCoplanar
)

// triangleCrossing is trianglesIntersect with the branch reported, and with a crossing smaller
// than allow discarded as faceting noise (see faceMeshDeviation). allow is a LENGTH: the straddling
// branch compares it against the crossing's depth, the coplanar branch against the square root of
// the shared area, because those are the two branches' natural measures. allow = 0 makes the test
// exact, which is what a pair of planar faces gets.
func triangleCrossing(t1, t2 [3]math.Point3, allow float64) (math.Point3, crossKind) {
	eps := crossRatio * stdmath.Min(triScale(t1), triScale(t2))
	n2, d2 := triPlaneEq(t2)
	s1, straddles1 := signedDistances(t1, n2, d2, eps)
	if !straddles1 {
		return coplanarCrossing(t1, t2, n2, s1, eps, allow)
	}
	n1, d1 := triPlaneEq(t1)
	s2, straddles2 := signedDistances(t2, n1, d1, eps)
	if !straddles2 {
		return math.Point3{}, crossNone
	}
	// Cross the UNIT normals, not the raw ones. |n| is ~2·area, so n1×n2 shrinks as the fourth
	// power of the model and underflows the zero-length guard on small parts — which would drop a
	// real crossing rather than report it. The unit cross product has magnitude sin θ, which
	// depends only on the angle between the planes.
	line, err := unitCross(n1, n2)
	if err != nil {
		return math.Point3{}, crossNone // planes too near parallel to define an intersection line
	}
	p, hit := intervalOverlap(t1, s1, t2, s2, line, eps)
	if !hit || crossingThickness(t1, t2) <= allow {
		return math.Point3{}, crossNone
	}
	return p, crossStraddle
}

// coplanarCrossing is the branch taken when t1 does not straddle t2's plane: either the two lie in
// that plane and overlap in area, or they do not meet at all.
//
// A coplanar pair has no depth, so the faceting allowance applies to its AREA. The existing ratio
// filter (#2074) compares the overlap to the smaller TRIANGLE, which a sliver passes on a
// meaningless absolute overlap: a torus tessellated against a plane it touches produced overlaps
// down to 1e-16 cm2 that way (#2077). Requiring the overlap to span more than allow on a side
// discards those without weakening the ratio test that catches edge contact.
func coplanarCrossing(t1, t2 [3]math.Point3, n2 math.Vector3, s1 [3]float64, eps, allow float64) (math.Point3, crossKind) {
	if !triCoplanar(s1, eps) {
		return math.Point3{}, crossNone
	}
	p, area, hit := coplanarOverlap(t1, t2, n2)
	if !hit || area <= allow*allow {
		return math.Point3{}, crossNone
	}
	return p, crossCoplanar
}

// parallelSinRatio is how far from parallel two planes must be before their intersection line is
// worth computing. It is |sin θ| — dimensionless — so it says nothing about the model's size.
const parallelSinRatio = 1e-12 // tol:numeric — a sine, not a length

// unitCross returns the unit direction of the two planes' intersection line. It divides by the
// normals' magnitudes explicitly rather than calling UnitVector3FromVector, whose zero-length guard
// is an ABSOLUTE 1e-9: |n| is ~2·area, so on a part measured in microns every face normal looks
// like a zero vector to that guard and a real crossing is silently dropped (#2075).
func unitCross(n1, n2 math.Vector3) (math.Vector3, error) {
	m1, m2 := float64(n1.Length()), float64(n2.Length())
	if m1 == 0 || m2 == 0 {
		return math.Vector3{}, fmt.Errorf("degenerate triangle normal: |n1|=%g |n2|=%g, want both > 0", m1, m2)
	}
	c := n1.Cross(n2)
	sin := float64(c.Length()) / (m1 * m2)
	if sin < parallelSinRatio {
		return math.Vector3{}, fmt.Errorf("planes are parallel to within |sin| = %g, want >= %g", sin, parallelSinRatio)
	}
	return c.Scale(math.Scalar(1 / (m1 * m2 * sin))), nil
}

// triScale is a triangle's characteristic length — its longest edge. It turns the dimensionless
// crossRatio into a length tolerance that follows the geometry instead of the model's units.
func triScale(t [3]math.Point3) float64 {
	longest := 0.0
	for i := range 3 {
		if d := float64(t[i].DistanceTo(t[(i+1)%3])); d > longest {
			longest = d
		}
	}
	return longest
}

// triPlaneEq returns the triangle's (unnormalized) plane normal and offset.
func triPlaneEq(t [3]math.Point3) (math.Vector3, float64) {
	n := t[0].VectorTo(t[1]).Cross(t[0].VectorTo(t[2]))
	return n, float64(n.Dot(t[0].AsVector()))
}

// signedDistances returns each vertex's TRUE perpendicular distance to the plane and whether the
// triangle genuinely straddles it (signs on both sides beyond eps). Normalising by |n| is what
// makes eps a length the caller can reason about (#2075).
func signedDistances(t [3]math.Point3, n math.Vector3, d, eps float64) ([3]float64, bool) {
	var s [3]float64
	scale := float64(n.Length())
	if scale == 0 {
		return s, false // a degenerate triangle has no plane to straddle
	}
	pos, neg := false, false
	for i, p := range t {
		s[i] = (float64(n.Dot(p.AsVector())) - d) / scale
		if s[i] > eps {
			pos = true
		}
		if s[i] < -eps {
			neg = true
		}
	}
	return s, pos && neg
}

func triCoplanar(s [3]float64, eps float64) bool {
	for _, v := range s {
		if v > eps || v < -eps {
			return false
		}
	}
	return true
}

// intervalOverlap projects both triangles' plane crossings onto the planes' intersection line —
// a UNIT direction, so the interval is a true arc length — and reports a witness point only when
// the intervals share more than eps of it. Faces that touch along a line share a single POINT of
// that line, so requiring positive length is what separates contact from interpenetration (#2075).
func intervalOverlap(t1 [3]math.Point3, s1 [3]float64, t2 [3]math.Point3, s2 [3]float64, line math.Vector3, eps float64) (math.Point3, bool) {
	a0, a1, pa := crossingInterval(t1, s1, line)
	b0, b1, _ := crossingInterval(t2, s2, line)
	lo, hi := max(a0, b0), min(a1, b1)
	if lo+eps >= hi {
		return math.Point3{}, false
	}
	return pa((lo + hi) / 2), true
}

// crossingInterval finds where the triangle's edges cross the other plane
// (s changes sign), projected on the intersection line; pa maps a line
// parameter back to a 3D witness point.
func crossingInterval(t [3]math.Point3, s [3]float64, line math.Vector3) (float64, float64, func(float64) math.Point3) {
	var pts []math.Point3
	for i := range 3 {
		j := (i + 1) % 3
		if (s[i] > 0) == (s[j] > 0) || s[i] == s[j] {
			continue
		}
		f := math.Scalar(s[i] / (s[i] - s[j]))
		pts = append(pts, t[i].TranslateBy(t[i].VectorTo(t[j]).Scale(f)))
	}
	if len(pts) < 2 {
		pts = append(pts, t[0], t[0]) // degenerate grazing — empty interval
	}
	u0 := float64(line.Dot(pts[0].AsVector()))
	u1 := float64(line.Dot(pts[1].AsVector()))
	p0, p1 := pts[0], pts[1]
	if u0 > u1 {
		u0, u1, p0, p1 = u1, u0, p1, p0
	}
	at := func(u float64) math.Point3 {
		if u1 == u0 {
			return p0
		}
		f := math.Scalar((u - u0) / (u1 - u0))
		return p0.TranslateBy(p0.VectorTo(p1).Scale(f))
	}
	return u0, u1, at
}
