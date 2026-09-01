// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SPIRIC OPEN-ARC canal ARM (A2 wave, OCCT blend/simple E6/E8/F1/F3): a torus SECTOR's
// meridian-cut rim is bounded between two corner vertices instead of closing into a full loop — the
// ARM counterpart to R4's torus-corner PATCH (fillet_torus_corner.go), which already solves the
// analytic sphere at each end. spiricRimSpine's per-station formula (fillet_spiric_spine.go) is
// closure-agnostic (proved by inspection: station(ψ) needs only the frame + trig constants, no loop
// assumption), so this file reuses it VERBATIM — the only new work is walking the edge's own BOUNDED
// ψ range instead of a full 2π loop, and resolving which of the two directions around the tube
// actually matches the picked edge (there is no free orientation choice here, unlike the closed
// engine's dir search: exactly one ψ-path from ψ0 to ψ1 passes through the edge's own interior point,
// and going the other way around the tube would leave the host face's angular span entirely).
//
// Dispatched from torusArmEdge's meridian branch AFTER spiricClosedRimArmEdge declines (an OPEN edge
// always fails isClosedCircleEdge, so the two engines are mutually exclusive by construction — do-no-
// harm): a closed spiric rim keeps its existing engine unchanged; an open one now gets this sibling
// instead of falling straight to the honest meridian reject.

// spiricArcAngle is p's tube angle ψ about the spine frame's origin D (the rim edge's own pre-fillet
// circle centre) — the same measurement spiricSeamAngle makes for the closed rim's seam vertex,
// generalized to any point (both edge endpoints AND an interior point, for the open arc's direction
// test). ok=false when p projects onto D itself (degenerate).
func spiricArcAngle(d math.Point3, spine spiricRimSpine, p math.Point3) (float64, bool) {
	rel := d.VectorTo(p)
	x := float64(rel.Dot(spine.mHat.AsVector()))
	z := float64(rel.Dot(spine.host.AxisDir.AsVector()))
	if stdmath.Hypot(x, z) == 0 {
		return 0, false
	}
	return stdmath.Atan2(z, x), true
}

// spiricArcSweep resolves the ONE ψ-path from ψ0 (edge start) to ψ1 (edge end) that actually matches
// the picked edge, by requiring ψ_mid (an interior point of the SAME edge) to fall strictly between
// them along that path — there is no free direction choice here (unlike the closed rim's outward-
// normal search): going the wrong way around the tube would trace an entirely different arc that does
// not contain ψ_mid. dir=+1 walks ψ increasing (CCW), −1 decreasing (CW); span is the total sweep
// (always in (0, 2π)). ok=false when NEITHER direction brackets ψ_mid (a degenerate/self-overlapping
// arc — do-no-harm).
func spiricArcSweep(psi0, psiMid, psi1 float64) (dir, span float64, ok bool) {
	const eps = 1e-9
	ccwSpan, ccwMid := wrapTwoPi(psi1-psi0), wrapTwoPi(psiMid-psi0)
	if ccwSpan > eps && ccwMid > eps && ccwMid < ccwSpan-eps {
		return 1, ccwSpan, true
	}
	cwSpan, cwMid := wrapTwoPi(psi0-psi1), wrapTwoPi(psi0-psiMid)
	if cwSpan > eps && cwMid > eps && cwMid < cwSpan-eps {
		return -1, cwSpan, true
	}
	return 0, 0, false
}

// spiricOpenArcStationsAt builds n+1 exact stations walking ψ from psi0 in direction dir over the
// resolved span (the mirror of spiricRimStationsAt, without the closed loop's seam re-evaluation —
// an open arc's two ends are already distinct edge vertices, nothing to re-close).
func spiricOpenArcStationsAt(spine spiricRimSpine, psi0, dir, span float64, n int) (ellipticRimStations, bool) {
	st := ellipticRimStations{
		centers:   make([]math.Point3, n+1),
		wallFeet:  make([]math.Point3, n+1),
		planeFeet: make([]math.Point3, n+1),
	}
	for k := 0; k <= n; k++ {
		psi := psi0 + dir*span*float64(k)/float64(n)
		c, tf, pf, ok := spine.station(psi)
		if !ok {
			return ellipticRimStations{}, false
		}
		st.centers[k], st.wallFeet[k], st.planeFeet[k] = c, tf, pf
	}
	return st, true
}

// resolveSpiricOpenArcStations doubles the station density until the measured between-station
// envelope error (spiricRimEnvelopeError, unchanged — it reads only the spine/stations/surface, never
// assumes closure) is within the model-relative bound — the open-arc analogue of
// resolveSpiricRimStations, with no orientation search (spiricArcSweep already fixed the one valid
// direction; there is nothing left to flip).
func resolveSpiricOpenArcStations(spine spiricRimSpine, psi0, dir, span float64, res tol.Resolution) (ellipticRimStations, geom.BSplineSurface, bool) {
	for n := spiricRimStationsMin; n <= spiricRimStationsMax; n *= 2 {
		st, ok := spiricOpenArcStationsAt(spine, psi0, dir, span, n)
		if !ok {
			return ellipticRimStations{}, geom.BSplineSurface{}, false
		}
		surf, err := geom.LoftCanalStations(st.centers, st.wallFeet, st.planeFeet, spine.r, res.Weld())
		if err != nil {
			return ellipticRimStations{}, geom.BSplineSurface{}, false
		}
		if spiricRimEnvelopeError(spine, st, surf) <= spiricRimEnvelopeCoef*res.Weld() {
			return st, surf, true
		}
	}
	return ellipticRimStations{}, geom.BSplineSurface{}, false
}

// spiricOpenArcArmEdge builds the spiric canal band for an OPEN meridian-cut Torus∧Plane rim ARC
// (E6/E8/F1/F3). handled=true ONLY when the band was built and certified; every decline returns false
// so the caller (torusArmEdge) keeps its existing meridian reject unchanged (do-no-harm) — the sibling
// of spiricClosedRimArmEdge, mutually exclusive with it by the isClosedCircleEdge/!isClosedCircleEdge
// split. Sets ONLY armSurface (a plain geom.BSplineSurface, no closed-rim payload): the arm's two ends
// are ordinary edge vertices — whatever corner/cap machinery already handles any other curved-host
// arm's endpoints (torusArmFillet's single-torus E7 arm included) is what resolves them here too, not
// a bespoke closed-loop seam.
func spiricOpenArcArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if p.varying() || isClosedCircleEdge(e) {
		return edgeFillet{}, false
	}
	host, pl, hostFace, planeFace, ok := torusPlaneEdgeFaces(e)
	if !ok {
		return edgeFillet{}, false
	}
	spine, ok := newSpiricRimSpine(body, e, host, pl, hostFace, planeFace, p.r0, false) // OPEN arc: only the visited stations need to exist
	if !ok {
		return edgeFillet{}, false
	}
	psi0, dir, span, ok := spiricEdgeArcSweep(e, spine)
	if !ok {
		return edgeFillet{}, false
	}
	_, surf, ok := resolveSpiricOpenArcStations(spine, psi0, dir, span, tol.ForBody(body))
	if !ok {
		return edgeFillet{}, false
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: surf}, true
}

// spiricEdgeArcSweep resolves the edge's own bounded ψ range: the start-vertex angle ψ0 and the
// (dir, span) spiricArcSweep found by bracketing the edge's interior midpoint — split out of
// spiricOpenArcArmEdge so it stays within funlen.
func spiricEdgeArcSweep(e *topo.Edge, spine spiricRimSpine) (psi0, dir, span float64, ok bool) {
	d, ok := rimCircleCenter(e)
	if !ok {
		return 0, 0, 0, false
	}
	psi0, ok0 := spiricArcAngle(d, spine, e.StartVertex().Point())
	psi1, ok1 := spiricArcAngle(d, spine, e.EndVertex().Point())
	psiMid, okMid := spiricArcAngle(d, spine, edgeMidpoint(e))
	if !ok0 || !ok1 || !okMid {
		return 0, 0, 0, false
	}
	dir, span, ok = spiricArcSweep(psi0, psiMid, psi1)
	return psi0, dir, span, ok
}
