// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// panelSide builds one long-boundary Side (the A-side or B-side, derivation §1.4/§3.1) of a
// dual-host panel between two exact corner points: when det's host is INACTIVE over the whole span
// (still the plain, unmodified fillet-cylinder tangent line — the sliver's off-host side, derivation
// §1.3) a straight tangent-line rail G1 to the host plane; when ACTIVE, the boss's own dip-rim
// sub-arc between from/to, G0 to the host plane (never G1 — the base-rim crease, the T6 fold lesson
// obstacleSides already documents for the single-host rim). from/to MUST be the already-resolved
// exact corner points (a wing node, a setbackSection endpoint) — never re-derived from a station
// z here — so a panel's rail shares its corner with the neighbouring wing/seam rail to machine
// precision (the corner-weld invariant, ADR-0042); extractPanelLoop is the caller that resolves them.
// ok=false when det.host is not planar (an unexpected obstacle-detection shape) or the rim sub-arc
// fails to fit.
func panelSide(ef edgeFillet, det obstacleDetection, active bool, from, to math.Point3) (Side, bool) {
	hostPl, ok := det.host.Geometry().(geom.Plane)
	if !ok {
		return Side{}, false
	}
	if !active {
		return Side{Curve: geom.NewLineSegment(from, to), Adjacent: hostPl, Cont: G1}, true
	}
	rail, ok := panelRimSubArc(det, ef, from, to)
	if !ok {
		return Side{}, false
	}
	return Side{Curve: rail, Adjacent: hostPl, Cont: G0}, true
}

// panelRimSubArc fits an approximated B-spline through det's dip rim between the exact points
// from/to (obstacleRimArc's own least-squares idiom — geom.NewApproximatedBSplineCurve — applied to
// a SUB-range instead of the full dip band), pinned exactly at both ends. It does NOT filter the
// existing obstacleRimSamples=64 discretization (dipRimSamples): probed empirically on U4's host B
// (the oblique elliptical-cylinder boss), those samples run ~1.06 apart along the axis while a
// sliver span is only ~0.39 wide, so a plain filter would carry ZERO interior points — not enough to
// fit a faithful sub-arc. Instead this resamples the SAME analytic rim curve dipRimPointAtStation
// already crosses, at panelRimSubSamples intermediate stations between from and to — the sub-range
// equivalent of dipRimSamples at the density a narrow band needs. ok=false when a station has no rim
// crossing (a from/to pair outside det's own active band — the caller's precondition).
func panelRimSubArc(det obstacleDetection, ef edgeFillet, from, to math.Point3) (geom.BSplineCurve, bool) {
	zFrom, zTo := axisParam(ef, from), axisParam(ef, to)
	pts := []math.Point3{from}
	for i := 1; i < panelRimSubSamples; i++ {
		z := zFrom + (zTo-zFrom)*float64(i)/float64(panelRimSubSamples)
		p, ok := dipRimPointAtStation(det, ef, z)
		if !ok {
			return geom.BSplineCurve{}, false
		}
		pts = append(pts, p)
	}
	pts = append(pts, to)
	nctrl := min(8, len(pts))
	bs, err := geom.NewApproximatedBSplineCurve(pts, 3, nctrl, geom.FitCentripetal)
	return bs, err == nil
}

// panelRimSubSamples is the intermediate-station density panelRimSubArc resamples a rim sub-arc at
// (9 total points including the two pinned ends) — matching obstacleRimArc's own nctrl=min(8,...)
// control-point budget, so the fit has as many points as it can use. Measured: doubling this to 16
// moves the sliver area match by < 0.001pp (0.9691%->0.9692%), so the residual ~0.9-1.0% gap to the
// oracle is NOT a rim-fit sampling-density artifact — it is the G1 ribbon/Coons-fill's own faithfulness
// (comfortably inside the corpus's 1% gate, corpus.json "U4".deps, so left at the smaller value).
const panelRimSubSamples = 8
