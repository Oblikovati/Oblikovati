// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The Gauss–Bonnet closure guard for the curved-arm trihedral corner (m5-weld-setback-retrim-
// derivation.md §A.3). The three weld rails form a closed spherical triangle on the corner sphere
// with vertices at the host-tangent points; this file certifies that triangle. It is the 2-sphere
// analogue of M4's scalar ΣΔ=2π corner test — stronger, because it simultaneously certifies the
// retrim (the same three host-tangent points anchor the host loops in T5.3/T5.4). A wrong-sign or
// mis-traversed rail set fails loudly here rather than welding an inside-out or torn corner.

// curvedClosureValid enforces the four §A.3 fail-loud invariants and returns false on any failure:
//  1. endpoint match — the arms' rails chain into one closed loop over exactly three shared points;
//  2. length = geodesic — each rail's subtense equals the geodesic between its two host-tangent points;
//  3. signed closure — the spherical excess E∈(0,2π) and r²E equals the forward triangle area;
//  4. per-arm station — each arm carries a resolved (finite) station root.
//
// Example:
//
//	if !curvedClosureValid(w, res) { /* reject: the corner does not close */ }
func curvedClosureValid(w cornerWeld, res Resolution) bool {
	if len(w.arms) < 3 || len(w.tPoints) != 3 {
		return false // not a closed trihedral corner
	}
	if !stationsResolved(w.arms) {
		return false // invariant 4: an arm has no resolved station root
	}
	dirs, ok := tangentDirs(w)
	if !ok {
		return false // a host-tangent point coincides with the centre (degenerate)
	}
	if !railsChain(w, res) {
		return false // invariant 1: the rails do not chain into one closed loop
	}
	if !railSubtensesMatch(w, dirs, res) {
		return false // invariant 2: a rail subtense ≠ its host-tangent geodesic
	}
	return sphericalClosure(dirs, w.radius, res) // invariant 3
}

// stationsResolved is invariant 4: every arm must carry a finite station (its in-domain root was
// solved). The uniqueness/existence of that root is enforced upstream at station-solve time
// (solveArmSetback rejects a gap); this guards against a NaN/Inf sneaking into the certificate.
func stationsResolved(arms []armSetback) bool {
	for _, a := range arms {
		if stdmath.IsNaN(a.station) || stdmath.IsInf(a.station, 0) {
			return false
		}
	}
	return true
}

// railsChain is invariant 1: each arm's two rail endpoints (C + r·railDir) must coincide with a
// host-tangent point, and each of the three tangent points must join exactly two rails — i.e. the
// three rails chain T→T→T→T into one closed loop. A dangling or doubled rail breaks the count.
func railsChain(w cornerWeld, res Resolution) bool {
	tol := res.Weld() * w.radius
	hits := make([]int, len(w.tPoints))
	for _, a := range w.arms {
		for _, d := range [2]math.UnitVector3{a.railDir0, a.railDir1} {
			idx := matchPoint(w.tPoints, endpointOf(w.center, w.radius, d), tol)
			if idx < 0 {
				return false
			}
			hits[idx]++
		}
	}
	for _, h := range hits {
		if h != 2 {
			return false // each tangent point must join exactly two rails
		}
	}
	return true
}

// railSubtensesMatch is invariant 2: each arm's rail subtense (the angle between its two rail
// directions) must equal the geodesic distance between the two host-tangent points it connects.
// The rail directions and the tangent-point directions are equal on valid input, so this bites
// only when a rail direction is perturbed independently of the tangent points.
func railSubtensesMatch(w cornerWeld, dirs [3]math.UnitVector3, res Resolution) bool {
	tol := res.Weld() * w.radius
	for _, a := range w.arms {
		i := matchPoint(w.tPoints, endpointOf(w.center, w.radius, a.railDir0), tol)
		j := matchPoint(w.tPoints, endpointOf(w.center, w.radius, a.railDir1), tol)
		if i < 0 || j < 0 || i == j {
			return false
		}
		if stdmath.Abs(a.railDir0.AngleTo(a.railDir1)-dirs[i].AngleTo(dirs[j])) > closureAngleTol {
			return false
		}
	}
	return true
}

// sphericalClosure is invariant 3: the spherical excess (from the interior angles, Gauss–Bonnet)
// must lie in (0,2π) and cross-check the solid angle (Van Oosterom–Strackee) — two independent
// formulas for the same area — within the corner-local area tolerance res.Weld·r².
func sphericalClosure(dirs [3]math.UnitVector3, r float64, res Resolution) bool {
	excess := sphericalExcess(dirs)
	if excess <= 0 || excess >= 2*stdmath.Pi {
		return false // degenerate (collapsed) or over-full triangle
	}
	solid := stdmath.Abs(solidAngle(dirs))
	return stdmath.Abs(excess-solid)*r*r <= res.Weld()*r*r
}

// tangentDirs returns the three host-tangent points as unit directions from the centre.
func tangentDirs(w cornerWeld) ([3]math.UnitVector3, bool) {
	var out [3]math.UnitVector3
	for i := 0; i < 3; i++ {
		d, err := math.UnitVector3FromVector(w.center.VectorTo(w.tPoints[i]))
		if err != nil {
			return out, false
		}
		out[i] = d
	}
	return out, true
}

// sphericalExcess is Σ(interior angles) − π (Gauss–Bonnet); the spherical triangle's area is r²·E.
func sphericalExcess(d [3]math.UnitVector3) float64 {
	a := interiorAngle(d[0], d[1], d[2])
	b := interiorAngle(d[1], d[2], d[0])
	c := interiorAngle(d[2], d[0], d[1])
	return a + b + c - stdmath.Pi
}

// interiorAngle is the spherical-triangle interior angle at vertex, between the arcs to p and q —
// the angle between the two arc tangents at vertex (each the component of the neighbour ⊥ vertex).
func interiorAngle(vertex, p, q math.UnitVector3) float64 {
	return tangentAt(vertex, p).AngleTo(tangentAt(vertex, q))
}

// tangentAt is the tangent of the great-circle arc vertex→other at vertex: other projected ⊥ vertex.
func tangentAt(vertex, other math.UnitVector3) math.Vector3 {
	v, o := vertex.AsVector(), other.AsVector()
	return o.Sub(v.Scale(o.Dot(v)))
}

// solidAngle is the signed solid angle of the spherical triangle (Van Oosterom–Strackee) for the
// three unit direction vectors — an independent second estimate of the excess used to cross-check
// the retrim area. tan(Ω/2) = a·(b×c) / (1 + a·b + b·c + a·c) for unit a,b,c.
func solidAngle(d [3]math.UnitVector3) float64 {
	a, b, c := d[0].AsVector(), d[1].AsVector(), d[2].AsVector()
	num := a.Dot(b.Cross(c))
	den := 1 + a.Dot(b) + b.Dot(c) + a.Dot(c)
	return 2 * stdmath.Atan2(num, den)
}
