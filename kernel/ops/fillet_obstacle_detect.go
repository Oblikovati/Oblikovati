// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// boundaryLine2 is the receded fillet boundary in the host plane: a point and a UNIT direction.
// It is deliberately unexported and local to this detector — the kernel's general-purpose
// geom.Line2d has no signed-distance query, and this slice only ever needs the perp-dot test below.
type boundaryLine2 struct {
	origin math.Point2
	dir    math.Vector2 // must be unit length
}

// signedDist returns the signed distance from p to the boundary line: +ve on the host side,
// -ve on the fillet side (the side dir's left-hand perpendicular points away from).
func (b boundaryLine2) signedDist(p math.Point2) float64 {
	return b.dir.Cross(b.origin.VectorTo(p))
}

// crossing is one intersection of the obstacle rim (as a sampled polyline) with the receded boundary
// line: the rim index just before it, the intersection point in host-plane 2D, and — once analyticNode
// has re-solved it ON the rim curve — that curve's own parameter there. T/onRim is what lets the two
// rim segments the node splits be TRIMMED sub-arcs of the rim's own conic (rimNodeTrimsOf) instead of
// straight truncated chords; a crossing that was only ever bracketed on the polyline has no such
// parameter and keeps onRim false.
type crossing struct {
	I     int         // rim polyline index whose segment [I, I+1] crosses the boundary
	P     math.Point2 // the crossing point
	T     float64     // the rim curve's parameter at P, valid only when onRim
	onRim bool        // P == rim.PointAt(T): the crossing was solved on the curve, not lerped on a chord
}

// rimCrossings returns the boundary crossings of the closed rim polyline, ordered as they appear along
// the rim. A crossing is a SIGN CHANGE of the signed distance to the boundary line larger than the
// model weld — so a vertex merely grazing the boundary (|d| ≤ weld on both sides) is NOT a crossing
// (the tangency guard, spec §Numerical pitfalls).
func rimCrossings(rim []math.Point2, b boundaryLine2, res Resolution) []crossing {
	tol := res.Weld()
	var out []crossing
	n := len(rim)
	for i := range n {
		a, c := rim[i], rim[(i+1)%n]
		da, dc := b.signedDist(a), b.signedDist(c)
		if da > tol && dc < -tol || da < -tol && dc > tol {
			out = append(out, crossing{I: i, P: lerpAtZero(a, c, da, dc)})
		}
	}
	return out
}

// lerpAtZero returns the point on segment a→c where the signed distance crosses zero.
func lerpAtZero(a, c math.Point2, da, dc float64) math.Point2 {
	t := da / (da - dc)
	return a.Lerp(c, t) // Point2.Lerp: stable single-eval, exact at t=0/1 (#1654)
}

// analyticNode re-solves a bracketed boundary node on the obstacle's EXACT rim curve. rimCrossings can
// only BRACKET a node: it runs on the obstacleRimSamples-chord polyline, so what it returns is the
// CHORD's zero of the signed distance — a full sagitta inside the CURVE's. Measured across the corpus's
// eleven obstacle-detecting cases, the node sat 3.05e-03 … 3.74e-02 off the exact closed-form rim∩band
// crossing (U4's boss-B node landed 3.815e-03 short of its exact ±√44, which alone cost each U4 sliver
// face ~0.9 % against DRAWEXE). Rim sample i IS the rim curve at parameter i/n (sampleHoleRim), so the
// crossing segment [I, I+1] is exactly the parameter bracket [I/n, (I+1)/n]; bisecting the signed
// distance there ON THE CURVE — with bisectRimParam, the same analytic solver dipRimPointAtStation
// already uses for the section endpoints — lands the node on the rim at the parametric floor (measured
// ≤1.8e-14 against the closed form, from ~1e-2). The polyline index .I is untouched, since every
// downstream consumer's dip range is expressed in polyline indices (mergeObstacleRim's
// "dip = nodes[0].I+1..nodes[1].I") and the refined point stays inside its own bracket. It also RECORDS
// the parameter it solved at (.T/.onRim): the node's two adjacent rim segments are then trimmed sub-arcs
// of the rim's own conic (rimNodeTrimsOf) rather than straight chords, which is the same solve read for
// geometry instead of for a point.
//
// It works in the host plane's own 2D frame (the frame rimCrossings works in), so the result drops
// straight into crossing.P and packDetection's lift is unchanged. It KEEPS the chord's lerped point when
// the curve does not strictly straddle the boundary across the sample's own parameter bracket — a sample
// sitting exactly on the line, or a rim whose chord sign change is not the curve's — so the refinement
// can only sharpen a node the polyline already found, never move it to a different root.
//
// It is applied on the OBSTACLE path only (analyticNodeDetections) and NOT to the runout-imprint path
// that shares bandCrossings: there the node pair is consumed solely by bandLineFromNodes to REBUILD the
// band line, and a chord-lerped point sits on that line exactly by construction (signedDist == 0 to the
// last bit) where a curve-solved one sits on it only to the bisection floor. That path then re-solves its
// own crossing analytically anyway (lineCircleRoots / solveConicLevel), so it has nothing to gain here
// and a rounding-level tilt of its own band line to lose.
func analyticNode(c crossing, rim geom.Curve3, n int,
	flat func(math.Point3) math.Point2, b boundaryLine2) crossing {
	f := func(t float64) float64 { return b.signedDist(flat(rim.PointAt(t))) }
	t0, t1 := float64(c.I)/float64(n), float64(c.I+1)/float64(n)
	f0, f1 := f(t0), f(t1)
	if f0 == 0 || f1 == 0 || (f0 < 0) == (f1 < 0) {
		return c
	}
	t := bisectRimParam(f, t0, t1)
	return crossing{I: c.I, P: flat(rim.PointAt(t)), T: t, onRim: true}
}

// obstacleNodes returns the two rim crossing indices bracketing the dip past the boundary, or
// ok=false when the rim does not genuinely cross twice (tangential touch or no dip → honest-reject,
// ADR-3). Exactly two crossings is the single-dip case this slice handles; >2 (a rim weaving across)
// is a tracked follow-up and also returns ok=false here so the caller honest-rejects rather than
// mis-building.
func obstacleNodes(rim []math.Point2, b boundaryLine2, res Resolution) ([2]crossing, bool) {
	cs := rimCrossings(rim, b, res)
	if len(cs) != 2 {
		return [2]crossing{}, false
	}
	return [2]crossing{cs[0], cs[1]}, true
}

// dipArcOrder returns the two crossings ordered so dipsPast's forward arc (c0→c1, wrapping through the
// array end when needed) is the SHORTER of the two candidate arcs the crossings split the closed rim
// into. A genuine mid-span obstacle is a LOCAL excursion — a short arc relative to the whole rim — while
// the complementary arc is the footprint's bulk sitting comfortably on the host side (ADR-4's
// single-dip case). Rim sample 0 (the curve's t=0 seam point, sampleHoleRim) is an arbitrary reference
// with no relation to the boundary line, so naively trusting the crossings' ascending array-index order
// — which, by construction, can never wrap through index 0, so the arc it selects always EXCLUDES index
// 0 — silently tests the wrong (majority, bulge) arc whenever the seam happens to fall INSIDE the true
// short dip arc instead. That is exactly what an oblique/non-uniformly-sampled elliptical footprint can
// do (#2007 U3: the true dip was an 11-of-64-sample arc containing index 0; ascending order handed
// dipsPast the 53-sample bulge arc instead, which is uniformly host-side and so can never test true
// regardless of which sample within it is examined — the defect is the ARC choice, not the sample).
// Ties (exactly half the rim either way) keep ascending order; model-relative geometry needs no
// tolerance here since arc length is an exact integer count, not a measured quantity.
func dipArcOrder(nodes [2]crossing, n int) (c0, c1 crossing) {
	arcLen := (nodes[1].I - nodes[0].I + n) % n
	// TODO(#2008): the exact-tie case 2*arcLen == n (crossings diametrically opposite,
	// reachable since obstacleRimSamples is even) falls back to ascending order here — the
	// pre-fix behavior that mis-picked U3's wrap dip. Harmless for the current corpus, but a
	// future opposite-crossing footprint with its true dip on the wrapping arc would reproduce
	// the #2007 defect. Resolve the tie by the signed-distance excursion, not arc length, when a
	// case demands it.
	if 2*arcLen <= n {
		return nodes[0], nodes[1]
	}
	return nodes[1], nodes[0]
}

// dipsPast reports whether the GIVEN forward arc (c0→c1, wrapping the array when c0.I > c1.I — the
// caller decides which of the two candidate arcs to hand in, dipArcOrder above) genuinely dips PAST the
// boundary into the fillet band, vs. bulging away. It tests the arc's GEOMETRIC EXTREMAL sample — the
// deepest excursion (max −side*signedDist) among every sample strictly between c0 and c1 — rather than a
// single INDEX-midpoint sample: an INDEX-midpoint only coincides with the arc's geometric middle when
// the rim is sampled at uniform arc length, which a non-uniformly-sampled (oblique/elongated elliptical)
// footprint is not — a single fixed-index sample can land where local sampling noise or curve shape
// gives a misleading value even though the SAME arc genuinely dips elsewhere. This function trusts
// whatever arc it is handed (the diamond wrap-around case below deliberately calls it with both
// orderings to pin that trust) — it does not itself decide which of the two candidate arcs is the
// obstacle; that choice is dipArcOrder's job (#2007 U3: a wrong arc CHOICE, not a wrong SAMPLE, was the
// root cause — a uniformly-signed arc gives the same verdict for every sample in it, extremal or not).
// side is +1 when the fillet band is on the negative-signed-distance side of the boundary (signedDist:
// host +ve, fillet -ve). A genuine dip has its deepest sample in the fillet band, so
// side*signedDist(extremal) is NEGATIVE — hence the `< 0` test (a bulge keeps every sample on the host
// side, so even the least-positive sample gives a positive product → false).
func dipsPast(rim []math.Point2, c0, c1 crossing, b boundaryLine2, side float64) bool {
	n := len(rim)
	arcLen := (c1.I - c0.I + n) % n
	extremal := side * b.signedDist(rim[(c0.I+1)%n])
	for off := 2; off <= arcLen; off++ {
		if d := side * b.signedDist(rim[(c0.I+off)%n]); d < extremal {
			extremal = d
		}
	}
	return extremal < 0
}
