// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED elliptic-rim canal BAND: stations from the closed-form spine
// (fillet_elliptic_rim_spine.go), lofted into the constant-radius canal BSpline with
// geom.LoftCanalStations — the SAME exact-station loft the cone canal arm ships (CN2), so the
// rational-quadratic cross-section, the chord-length v-parametrisation and the foot-at-radius assert
// are reused, not re-derived. The band CLOSES: station N repeats station 0 exactly, so the loft's two
// end cross-sections coincide and the topology carries the seam as one real edge (the wall's own seam
// azimuth), exactly as the analytic torus band does.
//
// Everything here is gated on `armEllipticRim != nil`, a payload ONLY this builder sets, so the
// analytic ruling path (F4), the circular-cylinder/cone/sphere/torus arms and the existing closed-rim
// bands (J1/I9/K1/S2/S5) cannot be reached by it — the maximally do-no-harm gate.

const (
	// ellipticRimStationsMin floors the ADAPTIVE station count around the rim; it doubles until the
	// MEASURED between-station envelope error is within bound. Every station column is EXACT on the
	// true envelope, so the count only controls the cubic v-interpolation between them.
	ellipticRimStationsMin = 32
	// ellipticRimStationsMax caps the doubling. Still over bound here means the rim is genuinely
	// unresolvable at this radius — honest-reject, never loft a band with a known gap.
	ellipticRimStationsMax = 512
	// ellipticRimEnvelopeCoef scales the model-relative envelope bound. The band's two RAILS are the
	// face boundaries the neighbours are re-trimmed onto, so topological watertightness is exact by
	// construction (one shared edge); this bounds the GEOMETRIC deviation of the lofted rail from the
	// true contact locus. 1e3·res.Weld() ≈ 1e-6 of the model size — four orders inside OCCT's own 1%
	// area gate, and reachable within the station cap for a closed rim (a bare res.Weld() would demand
	// O(1e4) stations on a 484-long elliptic directrix for no measurable gain).
	ellipticRimEnvelopeCoef = 1e3
	// ellipticRimArcSamples is the number of interior arc parameters probed per mid-interval when
	// measuring the envelope error (the two rails plus the arc midpoint).
	ellipticRimArcSamples = 3
)

// ellipticRimCanal is the built band: the canal surface, its two CLOSED rails (the wall contact locus
// and the plane contact locus, extracted as the loft's own u-isocurves so the face boundary and the
// face surface agree exactly), and the seam cross-section's on-arc midpoint.
type ellipticRimCanal struct {
	surf      geom.BSplineSurface
	wallRail  geom.Curve3
	planeRail geom.Curve3
	seamMid   math.Point3
	concave   bool
	r         float64
	// coneCap discriminates the EllipticalCylinder∧CONE pinched-canal vein (tolblend B4..C3,
	// fillet_elliptic_cone_canal.go): non-nil ONLY when that builder produced the payload, in
	// which case the rails/seamMid above are unused and ellipticClosedRimCanalBody routes to the
	// cone-cap rebuild instead of rebuildRim. Nil on every plane-cap band (J6/J8) — byte-invisible
	// to them.
	coneCap *ellipticConeCanal
}

// ellipticClosedRimArmEdge dispatches a CLOSED rim edge bounded by one geom.EllipticalCylinder wall
// and one plane cap to the canal band builder. handled=true ONLY when the band was built, so every
// decline falls through to the byte-identical curvedAdjacentError refusal (do-no-harm). The sibling of
// ellipticalCylinderArmEdge, which owns the OPEN straight-ruling edge of the same host (F4).
func ellipticClosedRimArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if p.varying() {
		return edgeFillet{}, false // constant radius only
	}
	if ef, handled := ellipticConeRimArmEdge(body, e, p); handled {
		return ef, true // tolblend B4..C3: EllipticalCylinder∧Cone pinched canal (closed or open arc)
	}
	if e.StartVertex() != e.EndVertex() {
		return edgeFillet{}, false // the plane-cap band below is CLOSED rim only
	}
	ec, pl, wallF, capF, ok := ellipticalCylinderPlaneEdge(e)
	if !ok {
		return edgeFillet{}, false
	}
	canal, ok := buildEllipticRimCanal(body, e, ec, pl, wallF, capF, p.r0)
	if !ok {
		return edgeFillet{}, false
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: canal.surf, armEllipticRim: canal}, true
}

// buildEllipticRimCanal solves the spine, resolves the station density, lofts the band and gates it on
// the two host faces actually CARRYING it. ok=false (fall through to the flat refusal) when the spine
// declines, the ball is not tangent everywhere (r past the wall's evolute), the loft declines, the
// band SPILLS off the cap face, or it overruns the wall's own extent.
func buildEllipticRimCanal(body *topo.Body, e *topo.Edge, ec geom.EllipticalCylinder, pl geom.Plane, wallF, capF *topo.Face, r float64) (*ellipticRimCanal, bool) {
	spine, ok := newEllipticRimSpine(body, e, ec, pl, wallF, r)
	if !ok {
		return nil, false
	}
	u0, _ := ec.ParamAt(e.StartVertex().Point()) // seam the band on the wall's own seam azimuth
	res := opstol.ForBody(body)
	st, surf, ok := resolveEllipticRimStations(spine, u0, res)
	if !ok {
		return nil, false
	}
	if !ellipticRimFootsFit(spine, st, e, pl, wallF, capF) {
		return nil, false
	}
	return assembleEllipticRimCanal(spine, st, surf)
}

// ellipticRimStations is one resolved station set: the exact ball centres and their two exact feet,
// with station N repeating station 0 so the lofted band closes on itself.
type ellipticRimStations struct {
	centers, wallFeet, planeFeet []math.Point3
}

// resolveEllipticRimStations chooses the station density that bounds the between-station envelope
// error, starting at ellipticRimStationsMin and doubling. It also fixes the band's ORIENTATION: the
// first attempt walks u forward, and if the lofted surface's own normal opposes the band's
// material-outward direction the walk is reversed (which flips ∂/∂v, hence the normal) so the rebuild
// can add the band face un-reversed exactly as the analytic torus band is added.
func resolveEllipticRimStations(spine ellipticRimSpine, u0 float64, res opstol.Resolution) (ellipticRimStations, geom.BSplineSurface, bool) {
	dir, ok := ellipticRimWalkDirection(spine, u0, res)
	if !ok {
		return ellipticRimStations{}, geom.BSplineSurface{}, false
	}
	for n := ellipticRimStationsMin; n <= ellipticRimStationsMax; n *= 2 {
		st, ok := ellipticRimStationsAt(spine, u0, dir, n)
		if !ok {
			return ellipticRimStations{}, geom.BSplineSurface{}, false
		}
		surf, err := geom.LoftCanalStations(st.centers, st.wallFeet, st.planeFeet, spine.r, res.Weld())
		if err != nil {
			return ellipticRimStations{}, geom.BSplineSurface{}, false
		}
		if ellipticRimEnvelopeError(spine, st, surf) <= ellipticRimEnvelopeCoef*res.Weld() {
			return st, surf, true
		}
	}
	return ellipticRimStations{}, geom.BSplineSurface{}, false
}

// ellipticRimWalkDirection fixes the u-walk sense ONCE, off a coarse trial loft: +1 when walking u
// forward already gives the band a surface normal pointing OUT of the solid, −1 otherwise (reversing
// the walk reverses ∂/∂v and so the normal). Deciding it once — rather than flipping inside the
// refinement loop — keeps the refinement monotone and cannot oscillate.
func ellipticRimWalkDirection(spine ellipticRimSpine, u0 float64, res opstol.Resolution) (float64, bool) {
	st, ok := ellipticRimStationsAt(spine, u0, 1, ellipticRimStationsMin)
	if !ok {
		return 0, false
	}
	surf, err := geom.LoftCanalStations(st.centers, st.wallFeet, st.planeFeet, spine.r, res.Weld())
	if err != nil {
		return 0, false
	}
	flip, ok := ellipticRimBandOutward(spine, st, surf)
	if !ok {
		return 0, false
	}
	if flip {
		return -1, true
	}
	return 1, true
}

// ellipticRimStationsAt builds n+1 uniformly-spaced stations around the rim starting at the seam
// azimuth u0 and walking in direction dir, with the LAST station evaluated at u0 itself so it is
// bit-identical to the first and the band closes with no gap. It declines when any station is not
// tangent to the wall — the r-past-the-evolute failure a constant-radius fillet must refuse.
func ellipticRimStationsAt(spine ellipticRimSpine, u0, dir float64, n int) (ellipticRimStations, bool) {
	st := ellipticRimStations{
		centers:   make([]math.Point3, n+1),
		wallFeet:  make([]math.Point3, n+1),
		planeFeet: make([]math.Point3, n+1),
	}
	tol := spine.r * spineTangencyCoef
	for k := 0; k <= n; k++ {
		u := u0 + dir*2*stdmath.Pi*float64(k)/float64(n)
		if k == n {
			u = u0 // close exactly on the seam station
		}
		c, wf, pf, ok := spine.station(u)
		if !ok || spine.tangencyError(c) > tol {
			return ellipticRimStations{}, false
		}
		st.centers[k], st.wallFeet[k], st.planeFeet[k] = c, wf, pf
	}
	return st, true
}

// spineTangencyCoef is the per-station tangency band as a fraction of r: |dist(C, wall) − r| must stay
// inside it. The distance comes from a numeric point inversion, so the band absorbs its convergence
// noise while still catching the real failure (the ball centre past the wall's evolute, where the true
// distance collapses well below r — a gross, not marginal, violation).
const spineTangencyCoef = 1e-6

// ellipticRimBandOutward reports whether the lofted band's own surface normal already points to the
// SOLID's outside, and if not that the station walk must be flipped. The band's outward direction is
// −side·(Q − C)/r: on a CONVEX rim the ball is inside the material so the surface faces away from the
// centre; on a CONCAVE rim the ball is in the void so it faces toward it.
//
// The decision is a dot of two UNIT vectors, so the degeneracy floor is dimensionless and needs no model
// scale (ADR-0042). Normalising `want` is what makes that true: it is C→Q, of magnitude ~r, so dotting it
// raw against the unit normal gave a LENGTH-scaled quantity whose effective threshold drifted with model
// scale — a 1e-9 floor meant 1e-9 of a radian on a unit part but 1e-10 of one on a 10× part.
func ellipticRimBandOutward(spine ellipticRimSpine, st ellipticRimStations, surf geom.BSplineSurface) (flip bool, ok bool) {
	j := len(st.centers) / 2
	vp := spineChordParams(st.centers)
	q := surf.PointAt(0.5, vp[j])
	want, err := math.UnitVector3FromVector(st.centers[j].VectorTo(q).Scale(-spine.side))
	if err != nil {
		return false, false // the probe point coincides with the ball centre — no outward direction
	}
	n := surf.NormalAt(0.5, vp[j])
	dot := float64(n.Dot(want.AsVector()))
	if stdmath.Abs(dot) < ellipticRimAxisTiltTol {
		return false, false // degenerate normal at the probe station — decline
	}
	return dot < 0, true
}

// ellipticRimEnvelopeError is the between-station error the loft must bound, measured at every
// interval MIDPOINT (station columns are exact by construction, so the error peaks there): how far the
// wall rail strays OFF the wall, how far the plane rail strays OFF the plane, and how far the arc
// midpoint strays from ball distance r about the exact centre of the station the wall rail lands on.
func ellipticRimEnvelopeError(spine ellipticRimSpine, st ellipticRimStations, surf geom.BSplineSurface) float64 {
	vp := spineChordParams(st.centers)
	worst := 0.0
	for j := 0; j+1 < len(st.centers); j++ {
		worst = stdmath.Max(worst, ellipticRimIntervalError(spine, surf, 0.5*(vp[j]+vp[j+1])))
	}
	return worst
}

// ellipticRimIntervalError measures the three deviations above at one v.
func ellipticRimIntervalError(spine ellipticRimSpine, surf geom.BSplineSurface, v float64) float64 {
	qWall := surf.PointAt(0, v)
	uStar, _, foot := geom.ClosestPointOnSurface(spine.ec, qWall)
	err := float64(foot.DistanceTo(qWall))
	qPlane := surf.PointAt(1, v)
	err = stdmath.Max(err, stdmath.Abs(float64(math.P3(0, 0, 0).VectorTo(qPlane).Dot(spine.nPl.AsVector()))-spine.cPl))
	c, _, _, ok := spine.station(uStar)
	if !ok {
		return stdmath.Inf(1)
	}
	for k := 1; k < ellipticRimArcSamples; k++ {
		q := surf.PointAt(float64(k)/float64(ellipticRimArcSamples), v)
		err = stdmath.Max(err, stdmath.Abs(float64(q.DistanceTo(c))-spine.r))
	}
	return err
}

// assembleEllipticRimCanal packages the band: its two rails read back as the loft's OWN u-isocurves
// (so the face boundary edges and the face surface are the same geometry by construction, not two
// independent fits), and the seam cross-section's on-arc midpoint — the exact radius-r arc point
// halfway between the two feet, which the rebuild threads through geom.Arc3dByThreePoints.
func assembleEllipticRimCanal(spine ellipticRimSpine, st ellipticRimStations, surf geom.BSplineSurface) (*ellipticRimCanal, bool) {
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
		return nil, false // the two feet are antipodal about the centre — no arc midpoint
	}
	return &ellipticRimCanal{
		surf: surf, wallRail: wallRail, planeRail: planeRail,
		seamMid: c0.TranslateBy(bis.AsVector().Scale(spine.r)),
		concave: spine.side > 0, r: spine.r,
	}, true
}

// ellipticClosedRimCanalArm is assembleCurvedArmBody's dispatch classifier: exactly ONE pick carrying
// the elliptic-rim canal PAYLOAD. Nothing else can set that payload, so no existing weld can be
// diverted here (do-no-harm).
func ellipticClosedRimCanalArm(fils []edgeFillet) (edgeFillet, bool) {
	if len(fils) != 1 || fils[0].armEllipticRim == nil {
		return edgeFillet{}, false
	}
	return fils[0], true
}

// ellipticClosedRimCanalBody welds the closed elliptic rim into its canal band through the SAME
// host-agnostic rim rebuild the analytic torus bands use (rebuildRim): the wall and the cap are
// re-trimmed onto the two rails and the band is inserted between them. An empty reason means the body
// is the weld; a non-empty one names the obstruction and the body is nil (never a partial body).
func ellipticClosedRimCanalBody(body *topo.Body, ef edgeFillet) (*topo.Body, string) {
	canal := ef.armEllipticRim
	e := ef.edge
	if canal.coneCap != nil {
		return ellipticConeCanalBody(body, e, canal.coneCap) // cone-cap pinched canal (tolblend B4..C3)
	}
	wallF, capF, ok := rimBandHosts(e)
	if !ok {
		return nil, fmt.Sprintf("elliptic rim canal: edge %d must border one elliptic wall and one cap plane", e.ID())
	}
	rimV := e.StartVertex()
	seamEdge, bottomV := wallSeam(wallF, e, rimV)
	if seamEdge == nil {
		return nil, fmt.Sprintf("elliptic rim canal: wall %T has no seam edge at the rim vertex to recede", wallF.Geometry())
	}
	rf := &rimFillet{
		cyl: wallF, cap: capF, rimEdge: e, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: canal.wallRail, capTan: canal.planeRail, band: canal.surf, r: canal.r,
		seamMid: canal.seamMid, concave: canal.concave,
	}
	b, err := rebuildRim(body, rf, canal.concave)
	if err != nil {
		return nil, fmt.Sprintf("elliptic rim canal rebuild declined: %v", err)
	}
	return b, ""
}
