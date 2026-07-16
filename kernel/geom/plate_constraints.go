// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// P3 of M6's degenerate-corner plate fill: turn the 4 rail SIDES of a corner into the
// PlateConstraint rows the P2 Duchon solver consumes. Each side is sampled along its Curve; at
// every sample a G0 (point-value) row is emitted, and — for a G1 side (Order 1) — two extra
// first-derivative rows (∂/∂u and ∂/∂v) that encode tangency to the side's Adjacent surface.
//
// The plate is solved PER COORDINATE (X, Y, Z are three independent scalar fields over the same
// (u,v) domain sharing one system matrix; PlateSolveMulti takes them as three RHS columns). So a
// G0 row contributes the foot's three coordinates and a G1 row the three components of each axis
// partial S_u, S_v. DiscretizeSides therefore returns the constraint SKELETON ([]PlateConstraint:
// U,V,Order only) PLUS a parallel [3][]float64 of the per-coordinate target values, lined up
// row-for-row, so P4a calls PlateSolveMulti(cs, vals[:]) directly. See .superpowers/sdd/
// plate-math-kit.md §3 for the watertight-critical G1-row recipe transcribed below.

// PlateSide is one rail of a degenerate corner: its 3D Curve, the Adjacent analytic surface the
// fill must be tangent to (needed only when Order==1), and the continuity Order (0 = G0-only
// rail, 1 = G1 rail carrying first-derivative tangency to Adjacent).
type PlateSide struct {
	Curve    Curve3
	Adjacent Surface
	Order    int
}

// plateRows accumulates the constraint skeleton and the three parallel coordinate-target value
// columns (X, Y, Z) so every emitted row appends to all four in lockstep.
type plateRows struct {
	cs   []PlateConstraint
	vals [3][]float64
}

// addG0 appends a G0 point-value row at (u,v) carrying the foot's three coordinates.
func (r *plateRows) addG0(u, v float64, foot math.Point3) {
	r.push(u, v, [2]int{0, 0}, foot.X, foot.Y, foot.Z)
}

// addDeriv appends a first-derivative row of the given order carrying an axis partial's three
// components (kit §3: the ∂/∂u row is S_u, the ∂/∂v row is S_v).
func (r *plateRows) addDeriv(u, v float64, order [2]int, s math.Vector3) {
	r.push(u, v, order, s.X, s.Y, s.Z)
}

func (r *plateRows) push(u, v float64, order [2]int, x, y, z float64) {
	r.cs = append(r.cs, PlateConstraint{U: u, V: v, Order: order})
	r.vals[0] = append(r.vals[0], x)
	r.vals[1] = append(r.vals[1], y)
	r.vals[2] = append(r.vals[2], z)
}

// DiscretizeSides samples the 4 corner rails into the plate solver's constraint rows: G0 point
// values everywhere, plus the tangency-encoding ∂/∂u,∂/∂v rows on the G1 (Order 1) sides. It
// returns the constraint skeleton and the parallel X/Y/Z target columns (feed the latter to
// PlateSolveMulti as vals[:]). Errors — so the caller falls back to coons4 — when fewer than 2
// samples are asked for, a rail projects to a near-degenerate strip in the average plane, a G1
// rail is near-perpendicular to Ω, or a G1 Adjacent surface has no well-defined normal at a foot.
//
// Example:
//
//	cs, vals, err := DiscretizeSides(sides, dom, 8)
//	fields, err := PlateSolveMulti(cs, vals[:]) // X, Y, Z share one matrix
func DiscretizeSides(sides [4]PlateSide, d PlateDomain, samples int) ([]PlateConstraint, [3][]float64, error) {
	if samples < 2 {
		return nil, [3][]float64{}, fmt.Errorf(
			"geom: DiscretizeSides needs samples >= 2 to resolve a rail direction, got %d", samples)
	}
	pts, err := allSampleWorldPoints(sides, samples)
	if err != nil {
		return nil, [3][]float64{}, err
	}
	res := ResolutionForPoints(pts)
	centroid := centroidOf(pts)
	var rows plateRows
	for _, side := range sides {
		if err := discretizeSide(side, d, curveParams(side.Curve, samples), res, centroid, &rows); err != nil {
			return nil, [3][]float64{}, err
		}
	}
	return rows.cs, rows.vals, nil
}

// allSampleWorldPoints evaluates every rail sample once up front — the corner-extent point set
// that fixes the model-relative Resolution and the world centroid used to orient the transverse
// tangents. It also validates that each side carries a Curve, and a G1 side an Adjacent surface.
func allSampleWorldPoints(sides [4]PlateSide, samples int) ([]math.Point3, error) {
	var pts []math.Point3
	for i, side := range sides {
		if side.Curve == nil {
			return nil, fmt.Errorf("geom: DiscretizeSides side %d has a nil Curve; all 4 rails are required", i)
		}
		if side.Order == 1 && side.Adjacent == nil {
			return nil, fmt.Errorf(
				"geom: DiscretizeSides side %d is G1 (Order 1) but has a nil Adjacent surface; a G1 rail needs a tangency surface", i)
		}
		for _, t := range curveParams(side.Curve, samples) {
			pts = append(pts, side.Curve.PointAt(t))
		}
	}
	return pts, nil
}

// curveParams returns the `samples` parameter values evenly spanning the curve's domain,
// endpoints inclusive (so adjacent sides share their shared corner point). Requires samples>=2.
func curveParams(c Curve3, samples int) []float64 {
	lo, hi := c.Domain()
	ts := make([]float64, samples)
	for k := range ts {
		ts[k] = lo + (hi-lo)*float64(k)/float64(samples-1)
	}
	return ts
}

// discretizeSide emits one side's rows: it first rejects a degenerate projected strip, then at
// each sample emits the G0 row and, for a G1 side, the two tangency-encoding derivative rows.
func discretizeSide(side PlateSide, d PlateDomain, ts []float64, res Resolution, centroid math.Point3, rows *plateRows) error {
	if err := checkStripNonDegenerate(side, d, ts, res); err != nil {
		return err
	}
	for _, t := range ts {
		foot := side.Curve.PointAt(t)
		u, v := d.Project(foot)
		rows.addG0(u, v, foot)
		if side.Order != 1 {
			continue
		}
		if err := addG1Rows(side, d, res, centroid, foot, t, rows); err != nil {
			return err
		}
	}
	return nil
}

// checkStripNonDegenerate rejects a rail that projects to a near-degenerate strip in Ω: a rail
// (near-)perpendicular to the average plane collapses to a point, carrying no in-plane direction
// for the solver. The error carries the measured domain arc-length and the expected minimum.
func checkStripNonDegenerate(side PlateSide, d PlateDomain, ts []float64, res Resolution) error {
	arc := sideDomainArcLength(side, d, ts)
	min := res.Weld()
	if arc <= min {
		return fmt.Errorf(
			"geom: DiscretizeSides rail projects to a degenerate strip in the average plane "+
				"(domain arc-length %.6g <= minimum %.6g = weld); the rail has no in-plane extent", arc, min)
	}
	return nil
}

// sideDomainArcLength sums the projected polyline length of the rail in Ω (Σ|Δ(u,v)|).
func sideDomainArcLength(side PlateSide, d PlateDomain, ts []float64) float64 {
	total := 0.0
	pu, pv := d.Project(side.Curve.PointAt(ts[0]))
	for _, t := range ts[1:] {
		u, v := d.Project(side.Curve.PointAt(t))
		total += stdmath.Hypot(u-pu, v-pv)
		pu, pv = u, v
	}
	return total
}

// addG1Rows appends the ∂/∂u and ∂/∂v tangency rows at one G1 foot (kit §3).
func addG1Rows(side PlateSide, d PlateDomain, res Resolution, centroid, foot math.Point3, t float64, rows *plateRows) error {
	su, sv, err := g1TangentTargets(side, d, res, centroid, foot, side.Curve.TangentAt(t))
	if err != nil {
		return err
	}
	u, v := d.Project(foot)
	rows.addDeriv(u, v, [2]int{1, 0}, su)
	rows.addDeriv(u, v, [2]int{0, 1}, sv)
	return nil
}

// g1TangentTargets builds the axis partials S_u, S_v that encode G1 tangency to Adjacent at a
// rail foot (kit §3). Both the along-rail tangent A=t/ρ and the transverse tangent τ lie in
// Adjacent's tangent plane, so S_u×S_v ∥ n̂ (exact G1); the axis-aligned pair is the domain
// rotation {d̂,ŵ}→{u,v} of {A,τ} (S_u = d̂ᵘA + ŵᵘτ, with ŵ = (−d̂ᵛ, d̂ᵘ)).
func g1TangentTargets(side PlateSide, d PlateDomain, res Resolution, centroid, foot math.Point3, tangent math.Vector3) (su, sv math.Vector3, err error) {
	dhu, dhv, rho, err := railDomainFrame(tangent, d, res, foot)
	if err != nil {
		return math.Vector3{}, math.Vector3{}, err
	}
	a := tangent.Scale(1 / rho) // along-rail world tangent A = t/ρ (auto-consistent with G0)
	normal, err := footNormal(side.Adjacent, foot)
	if err != nil {
		return math.Vector3{}, math.Vector3{}, err
	}
	tau, err := transverseTangent(a, normal, foot, centroid)
	if err != nil {
		return math.Vector3{}, math.Vector3{}, err
	}
	su = a.Scale(dhu).Add(tau.Scale(-dhv)) // ŵᵘ = −d̂ᵛ
	sv = a.Scale(dhv).Add(tau.Scale(dhu))  // ŵᵛ =  d̂ᵘ
	return su, sv, nil
}

// railDomainFrame decomposes the world rail tangent into the average-plane chart: the unit
// domain direction d̂ = (dhu,dhv) and the domain speed ρ = |projected tangent|. It rejects a
// rail near-perpendicular to Ω (ρ ≤ weld), where A = t/ρ would blow up (kit §3.3).
func railDomainFrame(tangent math.Vector3, d PlateDomain, res Resolution, foot math.Point3) (dhu, dhv, rho float64, err error) {
	du := tangent.Dot(d.U)
	dv := tangent.Dot(d.V)
	rho = stdmath.Hypot(du, dv)
	// ρ is a domain SPEED (d(u,v)/dt of the curve's own parameterization), not an arc-length —
	// so this guard trips at a threshold that shifts with how the rail happens to be
	// parameterized. checkStripNonDegenerate's domain ARC-LENGTH gate (parameterization-
	// invariant) is the authoritative degeneracy check; this one only protects the A=t/ρ
	// division from blowing up at a near-perpendicular rail.
	if rho <= res.Weld() {
		return 0, 0, 0, fmt.Errorf(
			"geom: DiscretizeSides G1 rail near-perpendicular to the average plane at foot %v "+
				"(domain speed %.6g <= weld %.6g); a G1 rail needs in-plane extent", foot, rho, res.Weld())
	}
	return du / rho, dv / rho, rho, nil
}

// footNormal returns Adjacent's unit normal at the rail foot, inverting the point onto the
// surface with the interface's own ParamAt (exact for the analytic surfaces). Errors when the
// normal is degenerate (a pole/apex): no tangent plane means no defined G1 tangency.
func footNormal(s Surface, foot math.Point3) (math.Vector3, error) {
	u, v := s.ParamAt(foot)
	n := s.NormalAt(u, v)
	if n.LengthSquared() == 0 {
		return math.Vector3{}, fmt.Errorf(
			"geom: DiscretizeSides adjacent surface normal degenerate at foot %v (ParamAt→(%.6g,%.6g)); "+
				"a G1 rail needs a well-defined tangent plane", foot, u, v)
	}
	return unitOrZero(n), nil
}

// transverseTangent builds τ = |A|·unit(n̂×Â), the in-tangent-plane direction transverse to the
// rail (kit §3.5). Its DIRECTION is watertight-critical; its magnitude |A| is the least-distorting
// (geometry-derived, not tuned) choice. Oriented into the patch interior (toward the world
// centroid). Errors if the rail tangent is parallel to n̂ (rail not in the tangent plane).
func transverseTangent(a, normal math.Vector3, foot, centroid math.Point3) (math.Vector3, error) {
	dir := unitOrZero(normal.Cross(unitOrZero(a)))
	if dir.LengthSquared() == 0 {
		return math.Vector3{}, fmt.Errorf(
			"geom: DiscretizeSides rail tangent %v is parallel to the adjacent normal %v at foot %v; "+
				"the rail must lie in the surface's tangent plane", a, normal, foot)
	}
	tau := dir.Scale(a.Length())
	if tau.Dot(foot.VectorTo(centroid)) < 0 {
		tau = tau.Negate()
	}
	return tau, nil
}
