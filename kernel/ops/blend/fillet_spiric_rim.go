// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SPIRIC closed-rim canal BAND (J3 convex, A4 concave): exact stations from the spiric spine
// (fillet_spiric_spine.go), lofted with geom.LoftCanalStations and packaged as the SAME closed-rim
// canal payload the elliptic band ships (armEllipticRim), so the proven host-agnostic rim rebuild —
// rails as the loft's own u-isocurves, wall-seam carry, concave winding flip — is reused verbatim,
// not re-derived. The stations wrap the TUBE (ψ around the minor circle) where the elliptic band's
// wrap the wall azimuth; everything downstream is direction-agnostic.
//
// Dispatched from torusArmEdge ONLY on its torusArmMeridian reject path, so every latitude rim, open
// arc, and non-torus pairing keeps its existing behavior byte-identically; a spiric decline restores
// the exact meridian reject message that shipped before this engine.

const (
	// spiricRimStationsMin/Max bound the adaptive station doubling around the tube — same policy as the
	// elliptic band: station columns are exact, the count only controls the loft between them.
	spiricRimStationsMin = 32
	spiricRimStationsMax = 512
	// spiricRimEnvelopeCoef scales the model-relative envelope bound (mirrors ellipticRimEnvelopeCoef;
	// see its comment for why 1e3·res.Weld() and not the raw weld).
	spiricRimEnvelopeCoef = 1e3
	// spiricRimArcSamples is the interior arc sampling per mid-interval for the envelope certificate.
	spiricRimArcSamples = 3
)

// spiricClosedRimArmEdge builds the spiric canal band for a CLOSED meridian-cut Torus∧Plane rim.
// handled=true ONLY when the band was built and certified; every decline returns false so the caller
// (torusArmEdge) keeps its existing meridian reject unchanged (do-no-harm).
func spiricClosedRimArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if p.varying() || !isClosedCircleEdge(e) {
		return edgeFillet{}, false
	}
	host, pl, hostFace, planeFace, ok := torusPlaneEdgeFaces(e)
	if !ok {
		return edgeFillet{}, false
	}
	spine, ok := newSpiricRimSpine(body, e, host, pl, hostFace, planeFace, p.r0, true) // CLOSED rim: require the full-loop existence guard
	if !ok {
		return edgeFillet{}, false
	}
	canal, ok := buildSpiricRimCanal(body, e, spine, pl, hostFace, planeFace)
	if !ok {
		return edgeFillet{}, false
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: canal.surf, armEllipticRim: canal}, true
}

// buildSpiricRimCanal resolves the station density, the walk direction, the fit gates and the
// envelope certificate, then packages the band as the closed-rim canal payload.
func buildSpiricRimCanal(body *topo.Body, e *topo.Edge, spine spiricRimSpine, pl geom.Plane, hostFace, planeFace *topo.Face) (*ellipticRimCanal, bool) {
	psi0, ok := spiricSeamAngle(e, spine)
	if !ok {
		return nil, false
	}
	res := tol.ForBody(body)
	st, surf, ok := resolveSpiricRimStations(spine, psi0, res)
	if !ok {
		return nil, false
	}
	if !capFaceContainsFeet(planeFace, pl, st.planeFeet) {
		return nil, false // the cap rail runs off the cap face — a clipped multi-piece band, not this engine
	}
	if !spiricWallSpanFits(spine, st, e, hostFace) {
		return nil, false // a tube foot ran past the host face's own angular span
	}
	return assembleSpiricRimCanal(spine, st, surf)
}

// spiricSeamAngle is the rim vertex's tube angle about the rim-circle centre — the station walk seams
// there so the rebuilt band's seam vertices land on the host face's own seam edge at the rim vertex.
func spiricSeamAngle(e *topo.Edge, spine spiricRimSpine) (float64, bool) {
	d, ok := rimCircleCenter(e)
	if !ok {
		return 0, false
	}
	rel := d.VectorTo(e.StartVertex().Point())
	x := float64(rel.Dot(spine.mHat.AsVector()))
	z := float64(rel.Dot(spine.host.AxisDir.AsVector()))
	if stdmath.Hypot(x, z) == 0 {
		return 0, false // rim vertex at the rim centre — degenerate
	}
	return stdmath.Atan2(z, x), true
}

// resolveSpiricRimStations mirrors resolveEllipticRimStations: fix the walk direction once off a
// coarse trial loft, then double the station count until the measured between-station envelope error
// is inside the bound (or honest-decline at the cap).
func resolveSpiricRimStations(spine spiricRimSpine, psi0 float64, res tol.Resolution) (ellipticRimStations, geom.BSplineSurface, bool) {
	dir, ok := spiricRimWalkDirection(spine, psi0, res)
	if !ok {
		return ellipticRimStations{}, geom.BSplineSurface{}, false
	}
	for n := spiricRimStationsMin; n <= spiricRimStationsMax; n *= 2 {
		st, ok := spiricRimStationsAt(spine, psi0, dir, n)
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

// spiricRimWalkDirection fixes the ψ-walk sense once, off a coarse trial loft, so the lofted band's
// surface normal points OUT of the solid and the rebuild can add the face un-reversed — the mirror of
// ellipticRimWalkDirection.
func spiricRimWalkDirection(spine spiricRimSpine, psi0 float64, res tol.Resolution) (float64, bool) {
	st, ok := spiricRimStationsAt(spine, psi0, 1, spiricRimStationsMin)
	if !ok {
		return 0, false
	}
	surf, err := geom.LoftCanalStations(st.centers, st.wallFeet, st.planeFeet, spine.r, res.Weld())
	if err != nil {
		return 0, false
	}
	flip, ok := spiricBandOutward(spine, st, surf)
	if !ok {
		return 0, false
	}
	if flip {
		return -1, true
	}
	return 1, true
}

// spiricRimStationsAt builds n+1 stations around the tube from the seam angle psi0, the last
// re-evaluated AT psi0 so the loft closes bit-identically on its seam (the elliptic band's closure
// invariant, and the discretizeEdge shared-station rule's loft-side analogue).
func spiricRimStationsAt(spine spiricRimSpine, psi0, dir float64, n int) (ellipticRimStations, bool) {
	st := ellipticRimStations{
		centers:   make([]math.Point3, n+1),
		wallFeet:  make([]math.Point3, n+1),
		planeFeet: make([]math.Point3, n+1),
	}
	for k := 0; k <= n; k++ {
		psi := psi0 + dir*2*stdmath.Pi*float64(k)/float64(n)
		if k == n {
			psi = psi0
		}
		c, tf, pf, ok := spine.station(psi)
		if !ok {
			return ellipticRimStations{}, false
		}
		st.centers[k], st.wallFeet[k], st.planeFeet[k] = c, tf, pf
	}
	return st, true
}

// spiricBandOutward mirrors ellipticRimBandOutward: the band's outward direction at the probe station
// is −σ·(Q − C)/r (convex: away from the material-side centre; concave: toward the void-side centre).
func spiricBandOutward(spine spiricRimSpine, st ellipticRimStations, surf geom.BSplineSurface) (flip bool, ok bool) {
	j := len(st.centers) / 2
	vp := spineChordParams(st.centers)
	q := surf.PointAt(0.5, vp[j])
	want, err := math.UnitVector3FromVector(st.centers[j].VectorTo(q).Scale(math.Scalar(-spine.side)))
	if err != nil {
		return false, false
	}
	n := surf.NormalAt(0.5, vp[j])
	dot := float64(n.Dot(want.AsVector()))
	if stdmath.Abs(dot) < ellipticRimAxisTiltTol {
		return false, false
	}
	return dot < 0, true
}

// spiricRimEnvelopeError is the between-station certificate, measured at every interval midpoint:
// the wall rail's distance OFF the host torus, the plane rail's distance OFF the cap plane, and the
// interior arc samples' deviation from ball radius r about the EXACT station centre at the tube angle
// the wall rail actually lands on — the MaxBallDev-style interior certificate, declared from the
// REQUEST geometry (host torus + cap plane + r), never from the loft itself.
func spiricRimEnvelopeError(spine spiricRimSpine, st ellipticRimStations, surf geom.BSplineSurface) float64 {
	vp := spineChordParams(st.centers)
	worst := 0.0
	for j := 0; j+1 < len(st.centers); j++ {
		worst = stdmath.Max(worst, spiricRimIntervalError(spine, surf, 0.5*(vp[j]+vp[j+1])))
	}
	return worst
}

// spiricRimIntervalError measures the three envelope deviations at one loft parameter v.
func spiricRimIntervalError(spine spiricRimSpine, surf geom.BSplineSurface, v float64) float64 {
	qWall := surf.PointAt(0, v)
	uStar, vStar := spine.host.ParamAt(qWall)
	eWall := float64(spine.host.PointAt(uStar, vStar).DistanceTo(qWall))
	qPlane := surf.PointAt(1, v)
	nV := spine.nHat.AsVector()
	ePlane := stdmath.Abs(float64(spine.host.Center.VectorTo(qPlane).Dot(nV)) - spine.capD)
	err := stdmath.Max(eWall, ePlane)
	c, _, _, ok := spine.station(vStar)
	if !ok {
		return stdmath.Inf(1)
	}
	eArc := 0.0
	for k := 1; k < spiricRimArcSamples; k++ {
		q := surf.PointAt(float64(k)/float64(spiricRimArcSamples), v)
		eArc = stdmath.Max(eArc, stdmath.Abs(float64(q.DistanceTo(c))-spine.r))
	}
	return stdmath.Max(err, eArc)
}

// spiricWallSpanFits gates every tube foot inside the host face's own angular span: the feet sit a
// small azimuth off the cap meridian (atan(|w|/ξ)) on the μ = sign(w) side — which is the host-face
// side by construction — and must not run past the face's FAR boundary circle, whose span is measured
// walking the SAME μ direction. ok=false also when the host face has no readable far boundary (a
// configuration this engine does not model — decline, do not guess).
func spiricWallSpanFits(spine spiricRimSpine, st ellipticRimStations, e *topo.Edge, hostFace *topo.Face) bool {
	farU, ok := spiricHostFarAzimuth(e, spine, hostFace)
	if !ok {
		return false
	}
	mu := 1.0
	if spine.w < 0 {
		mu = -1
	}
	span := stdmath.Mod(mu*farU+2*stdmath.Pi, 2*stdmath.Pi) // cap → far walked in the feet's own direction
	slack := spiricSpanSlack(spine)
	for _, f := range st.wallFeet {
		du := mu * spiricAzimuthOf(spine, f)
		if du < -slack || du > span-slack {
			return false
		}
	}
	return true
}

// spiricSpanSlack is the angular slack allowed past the cap azimuth on the rim side — the feet at the
// seam sit exactly ON the cap meridian only when w = 0; a |w| offset legitimately puts every foot a
// touch past it, never more than atan(|w| / ξ_min).
func spiricSpanSlack(spine spiricRimSpine) float64 {
	xiMin := spine.host.MajorRadius - spine.b // ξ at the inner turn is at least R − b − |w| — use the tube-centre bound
	return stdmath.Atan2(stdmath.Abs(spine.w), stdmath.Max(xiMin, spine.r)) + 1e-9
}

// spiricAzimuthOf is p's azimuth about the host axis measured from m̂ (the cap meridian), signed by
// the n̂ side, in (−π, π].
func spiricAzimuthOf(spine spiricRimSpine, p math.Point3) float64 {
	d := perpComponent(spine.host.Center.VectorTo(p), spine.host.AxisDir)
	return stdmath.Atan2(float64(d.Dot(spine.nHat.AsVector())), float64(d.Dot(spine.mHat.AsVector())))
}

// spiricHostFarAzimuth is the signed azimuth (from the cap meridian) of the host face's far boundary
// circle — the OTHER closed circular edge on the host face. ok=false when there is none.
func spiricHostFarAzimuth(e *topo.Edge, spine spiricRimSpine, hostFace *topo.Face) (float64, bool) {
	for _, he := range hostFace.Edges() {
		if he == e || he.StartVertex() != he.EndVertex() {
			continue
		}
		c, ok := rimCircleCenter(he)
		if !ok {
			continue
		}
		if u := spiricAzimuthOf(spine, c); stdmath.Abs(u) > spiricSpanSlack(spine) {
			return u, true
		}
	}
	return 0, false
}

// assembleSpiricRimCanal packages the band exactly as the elliptic band does: rails as the loft's OWN
// u-isocurves, the seam cross-section's on-arc midpoint, and the concave flag from σ.
func assembleSpiricRimCanal(spine spiricRimSpine, st ellipticRimStations, surf geom.BSplineSurface) (*ellipticRimCanal, bool) {
	wallRail, err := geom.SurfaceIsoCurve(surf, true, 0)
	if err != nil {
		return nil, false
	}
	planeRail, err := geom.SurfaceIsoCurve(surf, true, 1)
	if err != nil {
		return nil, false
	}
	c0 := st.centers[0]
	bis, err := math.UnitVector3FromVector(c0.VectorTo(st.wallFeet[0]).Add(c0.VectorTo(st.planeFeet[0])))
	if err != nil {
		return nil, false // antipodal feet about the centre — no arc midpoint
	}
	return &ellipticRimCanal{
		surf: surf, wallRail: wallRail, planeRail: planeRail,
		seamMid: c0.TranslateBy(bis.AsVector().Scale(math.Scalar(spine.r))),
		concave: spine.side > 0, r: spine.r,
	}, true
}
