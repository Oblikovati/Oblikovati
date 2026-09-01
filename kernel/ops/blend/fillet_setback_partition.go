// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// partitionTolCoef is the chord-to-angle multiplier k (k≈2..4, ADR-0042) that turns a model-relative
// weld length into the angular closure tolerance ε_θ = k·Weld()/r_f — never a bare epsilon.
const partitionTolCoef = 4

// rimPartition is the three directed native-parameter spans of a footprint-rim partition — hostA
// (seam→cross1), band (cross1→cross2, the interference notch), hostB (cross2→seam) — each a positive
// angle summing to 2π (the closure invariant §D3 of m4-rim-partition-derivation.md). hostA/hostB drive the
// minor-vs-major choice each host sub-arc is emitted with, replacing the old scale-dependent midpoint guess.
type rimPartition struct {
	hostA float64
	band  float64
	hostB float64
}

// partitionFootprintRim computes the scale-invariant σ-partition of a boss footprint conic (rule (b),
// §D2): with σ(p)=(p−contact)·ê the fillet-band ruler, both crossings lie on σ=0, so the band is the
// between-crossings arc whose interior midpoint has σ>0 and the two host arcs are the complement F∖band
// split at the seam. The one σ-sign test is taken deep inside the notch (never near the contact line), so
// it cannot flip with footprint size — unlike the old local-midpoint host-arc test that dropped 242° of
// the large torus rim (m4-spike.md §CRITICAL). ok=false honest-rejects the degenerate cases (§pitfalls:
// seam under the fillet, coincident/engulfing crossings, non-conic footprint, failed closure).
func partitionFootprintRim(boss crossingBoss, cyl geom.Cylinder, seam, cross1, cross2 math.Point3) (rimPartition, bool) {
	contact, edgeward, ok := footEdgeward(boss, cyl)
	if !ok || sigma(contact, edgeward, seam) >= 0 { // seam must be host-side (σ<0); else §D4 reseat (unwired)
		return rimPartition{}, false
	}
	ts, ok0 := footprintNativeAngle(boss.footEdge, seam)
	t1, ok1 := footprintNativeAngle(boss.footEdge, cross1)
	t2, ok2 := footprintNativeAngle(boss.footEdge, cross2)
	bandMinor, ok3 := bandIsMinorArc(boss.footEdge, contact, edgeward, cross1, cross2)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return rimPartition{}, false
	}
	return spansFromCuts(ts, t1, t2, bandMinor, partitionAngularTol(boss.footEdge, seam, cross1, cross2))
}

// spansFromCuts turns the three native cut angles (seam, cross1, cross2) into the ordered rimPartition.
// The band spans the (cross1,cross2) interval that is minor iff bandMinor; hostA/hostB are the complement
// F∖band, split at the seam, so they cannot re-cover the band. The closure guard (§D3) rejects any coding
// slip or degenerate (a span ≤ tol, or a sum ≠ 2π — e.g. a seam that fell inside the band interval).
func spansFromCuts(ts, t1, t2 float64, bandMinor bool, tol float64) (rimPartition, bool) {
	d12 := positiveAngleSpan(t2 - t1)               // forward span cross1→cross2
	bandForward := bandMinor == (d12 <= stdmath.Pi) // band traverses cross1→cross2 forward
	var p rimPartition
	if bandForward {
		p = rimPartition{hostA: positiveAngleSpan(t1 - ts), band: d12, hostB: positiveAngleSpan(ts - t2)}
	} else {
		p = rimPartition{hostA: positiveAngleSpan(ts - t1), band: 2*stdmath.Pi - d12, hostB: positiveAngleSpan(t2 - ts)}
	}
	if !partitionCloses(p, tol) {
		return rimPartition{}, false
	}
	return p, true
}

// partitionCloses is the checkable partition invariant: every directed span is strictly positive (no
// degenerate/zero-width arc) and the three sum to 2π within the angular tol — the malformed 118° slit
// (Δ-sum 237°) fails it loudly, the correct 241.6+57+61.6=360° passes (§D3).
func partitionCloses(p rimPartition, tol float64) bool {
	if p.hostA <= tol || p.band <= tol || p.hostB <= tol {
		return false
	}
	return stdmath.Abs(p.hostA+p.band+p.hostB-2*stdmath.Pi) <= tol
}

// footprintArcBySpan emits the footprint sub-arc from→to whose native span is `span`: the minor arc when
// span<π, else the major arc. The minor/major choice is DERIVED from the σ-partition's exact native span
// (partitionFootprintRim), never from a local midpoint side-test, so it is scale-invariant — the small
// cylinder (host=minor) and the large torus (host=major) both fall out of the same rule.
func footprintArcBySpan(footEdge *topo.Edge, from, to math.Point3, span float64) (geom.Curve3, bool) {
	if span > stdmath.Pi {
		return footprintMajorArc(footEdge, from, to)
	}
	return footprintSubArc(footEdge, from, to)
}

// bandIsMinorArc reports whether the interference band is the MINOR arc between the two crossings: true
// iff the minor arc's midpoint is on the fillet-band side (σ>0). That midpoint sits deep in the notch
// (σ margin ≈ r_f(1−cos(Δ_band/2)), three orders above weld tol on T1), so the sign never flips near L.
func bandIsMinorArc(footEdge *topo.Edge, contact math.Point3, edgeward math.Vector3, cross1, cross2 math.Point3) (bool, bool) {
	minor, ok := footprintSubArc(footEdge, cross1, cross2)
	if !ok {
		return false, false
	}
	return sigma(contact, edgeward, minor.PointAt(0.5)) > 0, true
}

// sigma is the signed offset to the fillet contact line L: σ(p)=(p−contact)·ê, positive on the band/edge
// side, negative on the free host side — the same ruler the setback stations solve σ=0 on.
func sigma(contact math.Point3, edgeward math.Vector3, p math.Point3) float64 {
	return contact.VectorTo(p).Dot(edgeward)
}

// footprintNativeAngle is a point's native angular parameter on the footprint conic: atan2 in the
// circle's own frame for a geom.Circle/geom.Arc3d, or the geom.EllipseFull parameter (NOT the point's
// atan2 — eccentric-vs-true-anomaly would misorder cut points on a high-eccentricity ellipse, §pitfalls).
// Consecutive native intervals tile [0,2π) by construction, which is what buys the partition its closure.
func footprintNativeAngle(footEdge *topo.Edge, p math.Point3) (float64, bool) {
	switch g := footEdge.Geometry().(type) {
	case geom.Circle:
		return circleNativeAngle(g.Center, g.RefDir, g.Normal, p), true
	case geom.Arc3d:
		return circleNativeAngle(g.Center, g.RefDir, g.Normal, p), true
	case geom.EllipseFull:
		return ellipseAngleOf(g, p), true
	default:
		return 0, false
	}
}

// circleNativeAngle is atan2(d·binormal, d·RefDir) with d=p−center — the angle of p in the circle's
// (RefDir, Normal×RefDir) frame, the same parametrization geom.Circle/Arc3d.PointAt evaluates.
func circleNativeAngle(center math.Point3, refDir, normal math.UnitVector3, p math.Point3) float64 {
	d := center.VectorTo(p)
	binormal := normal.Cross(refDir)
	return stdmath.Atan2(d.Dot(binormal), d.Dot(refDir.AsVector()))
}

// footprintRadius is the footprint conic's characteristic radius (the circle radius, or an ellipse's
// major radius) — the length ε_θ is scaled by to convert a weld tolerance into an angular one.
func footprintRadius(footEdge *topo.Edge) (float64, bool) {
	if e, ok := footEdge.Geometry().(geom.EllipseFull); ok {
		return e.MajorRadius, true
	}
	_, r, ok := footprintConic(footEdge)
	return r, ok
}

// partitionAngularTol is the model-relative angular closure tolerance ε_θ = k·Weld()/r_f (ADR-0042):
// the weld length over the pts' resolution, divided by the footprint radius (chord→angle). A non-conic or
// zero-radius footprint yields 0 (the closure guard then admits only an exact-to-ulp partition).
func partitionAngularTol(footEdge *topo.Edge, pts ...math.Point3) float64 {
	rf, ok := footprintRadius(footEdge)
	if !ok || rf <= 0 {
		return 0
	}
	return partitionTolCoef * opstol.ForPoints(pts).Weld() / rf
}

// positiveAngleSpan reduces a raw angle difference into [0, 2π) — the forward native span between two
// conic parameters, the ruler the three consecutive rim sub-arcs are measured with.
func positiveAngleSpan(delta float64) float64 {
	a := stdmath.Mod(delta, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}
