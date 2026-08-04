// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The line-vs-offset-torus tangency quartic and its root-selection protocol (fillet_torus_corner.go
// owns the corner-blend orchestration around this; realQuarticRoots, in quartic_real_roots.go,
// owns the polynomial-only Ferrari solve). Split out per SRP: this file is "given a line and an
// offset torus, find the admissible tangent point(s)"; it knows nothing about corner blends, host
// recognition, or tangent-point placement on the OTHER two faces.

// torusQuarticGraceBand is the point-space separation (× res.Weld()) below which two otherwise-
// distinct real roots are a GRAZING near-tangency of the offset line to the offset torus — an
// ill-conditioned configuration whose root pick is noise-dominated, not a genuine reflected-root
// pair (sibling of sphereRootsSeparated's grazing band, curvedCornerBandK reused for consistency
// with the rest of the R4 corner family).
const torusQuarticGraceBand = curvedCornerBandK

// torusCornerCenterOnLine solves C(t) = p0 + t·d for the torus host tangency (distance ρ from the
// core circle), non-dimensionalized by the corner's own resolution scale (ADR-0042 applied to
// polynomial conditioning: u, d, Rm, ρ are all divided by L = res.Size() before the quartic
// coefficients are built — t itself needs NO rescaling, since t is the dimensionless line
// parameter and distance-to-core-circle is homogeneous of degree 1 in every length input, so
// scaling every length input by 1/L scales F(t) by 1/L⁴ without moving its roots). Root selection
// is the three-tier protocol: (a) keep only real roots with G(t) ≥ 0 (the ONE extraneous-root class
// the single squaring introduces); (b) a lone survivor wins; (c) more than one survivor is
// adjudicated by the SAME station-domain witness the cylinder-host corner uses
// (cornerCylinderArms/rootStationsInDomain) — built even though none of the four corpus fixtures
// exercise it, per the N7 lesson that a silent nearer-vertex pick among ambiguous roots is exactly
// the bug class this project has been burned by before.
func torusCornerCenterOnLine(v *topo.Vertex, tor geom.Torus, p0 math.Point3, d math.Vector3, rho, r float64, res Resolution) (math.Point3, bool) {
	cands, ok := torusPhysicalCandidates(tor, p0, d, rho, res)
	if !ok {
		return math.Point3{}, false
	}
	switch len(cands) {
	case 0:
		return math.Point3{}, false
	case 1:
		return cands[0], true
	default:
		// Built here (not inside torusCornerTiebreak) so the tiebreak's OWN logic — the part this
		// wave adds — is unit-testable with hand-built arms, independent of v.Edges() traversal
		// (dependency injection over global lookup, CLAUDE.md).
		arms := cornerCylinderArms(v, r, res)
		return torusCornerTiebreak(arms, v.Point(), r, res, cands)
	}
}

// torusPhysicalCandidates returns the DISTINCT, physically-admissible candidate centres on the
// line: real quartic roots with G(t) ≥ 0, deduplicated and grazing-checked in POINT SPACE (never by
// thresholding the algebraic discriminant — the point-space gap has a clean geometric scale, the
// discriminant does not). ok=false when any two SURVIVING candidates are within the grazing band —
// an ill-conditioned near-tangency, honest-rejected rather than guessed.
func torusPhysicalCandidates(tor geom.Torus, p0 math.Point3, d math.Vector3, rho float64, res Resolution) ([]math.Point3, bool) {
	roots := torusTangencyRoots(tor, p0, d, rho, res)
	var pts []math.Point3
	for _, t := range roots {
		pts = append(pts, p0.TranslateBy(d.Scale(t)))
	}
	if !torusCandidatesWellSeparated(pts, res) {
		return nil, false // grazing: two roots too close in point space to trust either
	}
	return pts, true
}

// torusCandidatesWellSeparated reports whether every pair of candidate points is EITHER
// (near-)coincident (the same physical root, already deduplicated by realQuarticRoots upstream —
// unreachable here in practice, kept as a defensive no-op) or separated by at least the grazing
// band; a pair strictly between those — close but not the same point — is the grazing degeneracy.
func torusCandidatesWellSeparated(pts []math.Point3, res Resolution) bool {
	band := torusQuarticGraceBand * res.Weld()
	for i := range pts {
		for j := i + 1; j < len(pts); j++ {
			if d := float64(pts[i].DistanceTo(pts[j])); d > 0 && d < band {
				return false
			}
		}
	}
	return true
}

// torusTangencyRoots builds the non-dimensionalized quartic coefficients and returns its real
// roots (unfiltered by physical validity — torusPhysicalCandidates applies the G(t)≥0 filter, since
// it needs G(t) itself for the grazing/dedup logic too).
func torusTangencyRoots(tor geom.Torus, p0 math.Point3, d math.Vector3, rho float64, res Resolution) []float64 {
	L := stdmath.Max(stdmath.Max(res.Size(), tor.MinorRadius), 1)
	u := tor.Center.VectorTo(p0)
	c0, c1, c2, c3, c4 := torusQuarticCoeffs(u.Scale(1/L), d.Scale(1/L), tor.AxisDir.AsVector(), tor.MajorRadius/L, rho/L)
	roots := realQuarticRoots(c0, c1, c2, c3, c4)
	return torusFilterByG(roots, u, d, tor.MajorRadius, rho)
}

// torusFilterByG keeps only roots t with G(t) = Q(t)+Rm²−ρ² ≥ −band, the single extraneous-root
// class the offset torus's one squaring introduces (a negative G means the pre-squaring equation
// required radial(t) = −K(t) < 0, not a real distance — G does not involve axial/â at all, only
// Q(t)). Evaluated in the ORIGINAL (unscaled) units against the Newton-polished root, so the filter
// is exact regardless of the solve's internal non-dimensionalization. band is a small NEGATIVE-side
// slack (res-scaled) so an exactly-tangent (G≈0) root survives float round-off instead of being
// spuriously dropped.
func torusFilterByG(roots []float64, u, d math.Vector3, rm, rho float64) []float64 {
	band := torusQuarticGraceBand * (rm*rm + rho*rho) * 1e-12 // dimensionally length⁴, matching G's units
	var kept []float64
	for _, t := range roots {
		w := u.Add(d.Scale(t))
		q := float64(w.Dot(w))
		g := q + rm*rm - rho*rho
		if g >= -band {
			kept = append(kept, t)
		}
	}
	return kept
}

// torusQuarticCoeffs builds F(t) = (Q(t)+Rm²−ρ²)² − 4Rm²(Q(t)−axial(t)²)'s coefficients from the
// line (u,d) and torus (â,Rm,ρ) — ALL ALREADY NON-DIMENSIONALIZED by the caller (torusTangencyRoots)
// — exactly the derivation's closed form: g(t)=Q(t)+Rm²−ρ² and h(t)=Q(t)−axial(t)² are quadratics
// in t; c4=g2², c3=2g1g2, c2=g1²+2g0g2−4Rm²h2, c1=2g0g1−4Rm²h1, c0=g0²−4Rm²h0.
func torusQuarticCoeffs(u, d, ahat math.Vector3, rm, rho float64) (c0, c1, c2, c3, c4 float64) {
	q2, q1, q0 := float64(d.Dot(d)), 2*float64(u.Dot(d)), float64(u.Dot(u))
	a1, a0 := float64(d.Dot(ahat)), float64(u.Dot(ahat))
	g2, g1, g0 := q2, q1, q0+rm*rm-rho*rho
	h2, h1, h0 := q2-a1*a1, q1-2*a0*a1, q0-a0*a0
	c4 = g2 * g2
	c3 = 2 * g1 * g2
	c2 = g1*g1 + 2*g0*g2 - 4*rm*rm*h2
	c1 = 2*g0*g1 - 4*rm*rm*h1
	c0 = g0*g0 - 4*rm*rm*h0
	return c0, c1, c2, c3, c4
}

// torusCornerTiebreak adjudicates more than one physically-admissible candidate with the SAME
// station-domain witness the cylinder-host corner's selectCornerRoot uses (cornerCylinderArms +
// rootStationsInDomain): each candidate must sit in-domain on EVERY straight (cylinder) arm built
// at the corner. No arms present keeps the legacy nearer-vertex pick (mirrors sphereCornerRoot /
// coneCornerRoot's "no witness available" fallback); more than one (or zero) in-domain candidate is
// an honest reject rather than a guess.
func torusCornerTiebreak(arms []cornerArm, vp math.Point3, r float64, res Resolution, cands []math.Point3) (math.Point3, bool) {
	if len(arms) == 0 {
		return nearestOf(vp, cands), true
	}
	var chosen math.Point3
	n := 0
	scale := stdmath.Max(res.Size(), r)
	for _, c := range cands {
		if rootStationsInDomain(arms, vp, c, scale, res) {
			chosen, n = c, n+1
		}
	}
	if n != 1 {
		return math.Point3{}, false // neither/both in-domain: ambiguous — honest-reject (do-no-harm)
	}
	return chosen, true
}

// nearestOf returns whichever candidate lies nearest to vp.
func nearestOf(vp math.Point3, cands []math.Point3) math.Point3 {
	best := cands[0]
	bestD := vp.DistanceTo(best)
	for _, c := range cands[1:] {
		if d := vp.DistanceTo(c); d < bestD {
			best, bestD = c, d
		}
	}
	return best
}
