// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Station resolution for the EllipticalCylinder∧Cone pinched canal: the azimuth-span resolvers
// (closed rim / open arc), the exact station grid with its topology anchors, the fit gates that
// keep the band on its two host faces, and the measured envelope error the density loop bounds.

// closedEllipticConeSpan fixes the walk for a CLOSED rim: the full period starting and ending at
// the pinch. Only the pinch-AT-the-rim-vertex sub-case (C3 — the host seams meet the rim exactly
// at the tangency azimuth, so the rails simply close through that kept vertex with NO split) ships
// here.
//
// The OFF-seam sub-case (B7/C2: the wall/cone seam anchor azimuths land a small arc away from the
// pinch, so the rebuild must split each rail at its own anchor) is a DELIBERATE, MEASURED decline,
// not a missing feature stub: it was built (bindSplitRails/bindOneSplitRail in
// fillet_elliptic_cone_rebuild.go, still present for a future fix) and DRAWEXE-reconciled, and the
// reconciliation falsified it. Per-face live-oracle receipts (DRAWEXE 8.0.0, tolblend_simple/B7,
// r=75, SCALE1=5; wave-report-F.md carries the full session):
//   - our band measures 109109.2, DRAWEXE's own band (its OWN face split, `explode result f` +
//     `sprops result_i 1.e-9`) is TWO patches summing to 27640.1+293.511 = 27933.6 — a ~3.9x
//     overshoot. The other 4 faces (both host laterals + both untouched far caps) match DRAWEXE to
//     ≤0.04% (quadrature noise on the oblique wall, see perface_oracle_test.go), so the defect is
//     confined to the band surface itself, not the host retrim.
//   - DRAWEXE's own two band patches' V-knot ranges are [8.659,785.398] and [785.398,794.058]: they
//     jointly span exactly one 2π*R (R≈125, the wall radius) loop, split at azimuth≈4° — i.e. OCCT
//     also builds ONE continuous closed-loop canal, split into 2 NURBS patches purely for its own
//     surface bookkeeping, not because of a physically different (partial-loop) construction. A
//     back-of-envelope Fermi check (average cross-section arc-length × total rim circumference,
//     ~35 × 785 ≈ 27,500) lands within 2% of DRAWEXE's real number and nowhere near ours — so
//     27933.6, not 109109.2, is the physically correct band area.
//   - Per-station tangency certificates all pass (every column IS exactly r from both hosts) and the
//     azimuth sweep of the half-angle is smooth, unimodal, and bounded (~0 to 0.46 rad) — ruling out
//     a wrong-root/self-crossing spine. The one thing UNIQUE to this sub-case (not exercised by the
//     working C3 pinch-at-seam or B4/B8 open-arc paths, which never anchor-snap) is
//     ellipticConeStationGrid's snapAnchorOnGrid: it places the wall- and cone-seam anchors ~0.04
//     rad apart on a ~0.098 rad average grid step — two NEARLY-COINCIDENT v-parameter columns feeding
//     geom.LoftPinchedCanalStations' GLOBAL chord-length B-spline interpolation. That is a classic
//     ill-conditioned-interpolation setup (Runge-style overshoot between near-duplicate knots): the
//     data COLUMNS stay exactly right (hence the certificates pass) while the INTERPOLATED surface
//     between them can bulge well past the true envelope without folding — consistent with every
//     symptom measured. Untested hypothesis, not a proven fix: the next attempt should either merge
//     near-duplicate anchor columns or use a knot-insertion/local scheme robust to them, then
//     re-verify per-face against DRAWEXE before trusting it.
//
// Shipping the wrong body would be worse than declining it (it already reads FAIL(area), but the
// mesh IS watertight and would silently look "almost right"): honest refusal until the above is
// resolved.
func closedEllipticConeSpan(e *topo.Edge, spine ellipticConeRimSpine, pinchFoot math.Point3, res tol.Resolution) (ellipticConeRimSpan, bool) {
	rimV := e.StartVertex().Point()
	slack := ellipticRimEnvelopeCoef * res.Weld()
	if float64(rimV.DistanceTo(pinchFoot)) > slack {
		return ellipticConeRimSpan{}, false // off-seam pinch: measured-wrong, see doc comment above
	}
	uV, _ := spine.ec.ParamAt(rimV)
	return ellipticConeRimSpan{u0: uV, u1: uV + 2*stdmath.Pi, pinchStart: true, closed: true, pinch: rimV}, true
}

// openEllipticConeSpan fixes the walk for an OPEN arc rim: from the non-pinched end (which must
// run out onto a planar side face — verified later by the rebuild) to the pinched end, which
// must coincide with the arc's end vertex (B4/B8: the tangency azimuth IS the arc boundary).
func openEllipticConeSpan(e *topo.Edge, spine ellipticConeRimSpine, lo, hi float64, res tol.Resolution) (ellipticConeRimSpan, bool) {
	slack := ellipticRimEnvelopeCoef * res.Weld()
	chordEnd := func(u float64) float64 {
		st, ok := spine.solveStation(u)
		if !ok {
			return stdmath.Inf(1)
		}
		return 2 * spine.r * stdmath.Sin(spine.sectionHalfAngle(st))
	}
	switch {
	case chordEnd(hi) <= slack:
		return openSpanTo(e, spine, lo, hi, res)
	case chordEnd(lo) <= slack:
		return openSpanTo(e, spine, hi, lo, res)
	}
	return ellipticConeRimSpan{}, false // interior pinch on an open arc — a different construction
}

// openSpanTo packages the open walk u0→u1 (u1 pinched) with the pinch snapped to the arc's end
// VERTEX point at that azimuth, so the band welds onto the existing solid vertex exactly.
func openSpanTo(e *topo.Edge, spine ellipticConeRimSpine, u0, u1 float64, res tol.Resolution) (ellipticConeRimSpan, bool) {
	v, ok := arcEndVertexAt(e, spine, u1, res)
	if !ok {
		return ellipticConeRimSpan{}, false
	}
	return ellipticConeRimSpan{u0: u0, u1: u1, pinchStart: false, closed: false, pinch: v}, true
}

// arcEndVertexAt returns the rim endpoint vertex whose point sits at azimuth u (the pinched
// end), within the band's geometric slack.
func arcEndVertexAt(e *topo.Edge, spine ellipticConeRimSpine, u float64, res tol.Resolution) (math.Point3, bool) {
	st, ok := spine.solveStation(u)
	if !ok {
		return math.Point3{}, false
	}
	slack := ellipticRimEnvelopeCoef * res.Weld()
	for _, v := range []math.Point3{e.StartVertex().Point(), e.EndVertex().Point()} {
		if float64(v.DistanceTo(st.wallFoot)) <= slack {
			return v, true
		}
	}
	return math.Point3{}, false
}

// resolveEllipticConeCanal builds the n-interval station grid, gates every station, lofts the
// pinched band, and fixes its outward orientation (reversing the walk when the trial normal points
// into the material).
func resolveEllipticConeCanal(e *topo.Edge, spine ellipticConeRimSpine, span ellipticConeRimSpan, wallF, coneF *topo.Face, n int, res tol.Resolution) (*ellipticConeCanal, bool) {
	canal, ok := loftEllipticConeSpan(e, spine, span, wallF, coneF, n, res)
	if !ok {
		return nil, false
	}
	flip, ok := ellipticConeBandOutward(spine, canal)
	if !ok {
		return nil, false
	}
	if !flip {
		return canal, true
	}
	rev := reversedEllipticConeSpan(span)
	return loftEllipticConeSpan(e, spine, rev, wallF, coneF, n, res)
}

// reversedEllipticConeSpan swaps the walk direction (u0↔u1), moving the pinch flags with it.
func reversedEllipticConeSpan(s ellipticConeRimSpan) ellipticConeRimSpan {
	r := s
	r.u0, r.u1 = s.u1, s.u0
	if !s.closed {
		r.pinchStart = true // the open walk now starts at the pinched end
	}
	return r
}

// loftEllipticConeSpan solves the exact stations over the span and lofts them.
func loftEllipticConeSpan(e *topo.Edge, spine ellipticConeRimSpine, span ellipticConeRimSpan, wallF, coneF *topo.Face, n int, res tol.Resolution) (*ellipticConeCanal, bool) {
	grid := ellipticConeStationGrid(span, n)
	sts, ok := solveGatedStations(e, spine, span, grid, wallF, coneF, res)
	if !ok {
		return nil, false
	}
	centers, wallFeet, coneFeet := stationRows(sts, span)
	pinchEnd := span.closed || !span.pinchStart
	loft, err := geom.LoftPinchedCanalStations(centers, wallFeet, coneFeet, spine.r, res.Weld(), span.pinchStart, pinchEnd)
	if err != nil {
		return nil, false
	}
	return &ellipticConeCanal{spine: spine, loft: loft, stations: sts, closed: span.closed,
		pinch: span.pinch, wallF: wallF, coneF: coneF}, true
}

// ellipticConeStationGrid is the uniform azimuth grid spanning the walk.
func ellipticConeStationGrid(span ellipticConeRimSpan, n int) []float64 {
	grid := make([]float64, n+1)
	for k := 0; k <= n; k++ {
		grid[k] = span.u0 + (span.u1-span.u0)*float64(k)/float64(n)
	}
	return grid
}

// solveGatedStations solves every grid azimuth and applies the per-station certificates and the
// on-face fit gates. The pinched end stations are exempt from the certificate (their synthesized
// limit is checked by the loft) but still fit-gated.
func solveGatedStations(e *topo.Edge, spine ellipticConeRimSpine, span ellipticConeRimSpan, grid []float64, wallF, coneF *topo.Face, res tol.Resolution) ([]ellipticConeStation, bool) {
	gates, ok := newEllipticConeFitGates(e, wallF, coneF, res)
	if !ok {
		return nil, false
	}
	sts := make([]ellipticConeStation, len(grid))
	for k, u := range grid {
		st, sok := spine.solveStation(u)
		if !sok || !gates.admit(spine, u, st) {
			return nil, false
		}
		if !pinchIndex(span, k, len(grid)) && spine.stationCertificateError(st) > spineTangencyCoef*spine.r {
			return nil, false
		}
		sts[k] = st
	}
	return sts, true
}

// pinchIndex reports whether grid index k is a pinched end of the span.
func pinchIndex(span ellipticConeRimSpan, k, n int) bool {
	if span.closed {
		return k == 0 || k == n-1
	}
	if span.pinchStart {
		return k == 0
	}
	return k == n-1
}

// stationRows extracts the loft rows, substituting the synthesized pinch columns: both feet AT
// the pinch point so the band closes exactly on the solid's vertex/rail junction.
func stationRows(sts []ellipticConeStation, span ellipticConeRimSpan) (centers, wallFeet, coneFeet []math.Point3) {
	n := len(sts)
	centers = make([]math.Point3, n)
	wallFeet = make([]math.Point3, n)
	coneFeet = make([]math.Point3, n)
	for k, st := range sts {
		centers[k], wallFeet[k], coneFeet[k] = st.center, st.wallFoot, st.coneFoot
		if pinchIndex(span, k, n) {
			wallFeet[k], coneFeet[k] = span.pinch, span.pinch
		}
	}
	return centers, wallFeet, coneFeet
}

// ellipticConeBandOutward probes whether the lofted band's normal already points out of the
// solid at the mid station (the concave band faces its ball centre) — flip=true reverses the
// walk. The dimensionless dot floor mirrors ellipticRimBandOutward (ADR-0042).
func ellipticConeBandOutward(spine ellipticConeRimSpine, canal *ellipticConeCanal) (flip, ok bool) {
	j := len(canal.stations) / 2
	q := canal.loft.Surf.PointAt(0.5, canal.loft.VParams[j])
	want, err := math.UnitVector3FromVector(canal.stations[j].center.VectorTo(q).Scale(-spine.side))
	if err != nil {
		return false, false
	}
	dot := float64(canal.loft.Surf.NormalAt(0.5, canal.loft.VParams[j]).Dot(want.AsVector()))
	if stdmath.Abs(dot) < ellipticRimAxisTiltTol {
		return false, false
	}
	return dot < 0, true
}
