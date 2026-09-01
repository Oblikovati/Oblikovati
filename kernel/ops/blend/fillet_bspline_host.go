// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// B-spline-host fillet arm (wave G): a picked edge bordered by at least one
// geom.BSplineSurface host (the other host a plane or a second B-spline) gets a NUMERIC
// rolling-ball canal band — stations marched by geom.MarchCanalEdgeStations (OCCT
// BlendFunc_ConstRad's section condition + BRepBlend_Walking's warm continuation, ported
// faithfully) and lofted by geom.LoftCanalStations, exactly as the elliptic rim canal
// (fillet_elliptic_rim_canal.go) does for its closed-form spine. Everything downstream is
// gated on the bsplineHostArmSurface WRAPPER type, which only this builder constructs, so
// no existing arm, weld or rim path can be diverted here (the armEllipticRim gating
// pattern — do-no-harm).

// bsplineHostCanal is the built band payload: the canal surface, its two foot rails (the
// loft's own u-isocurves, so face boundary and face surface agree exactly), the stations
// (for the end-trim bracketing), and the classification the body builders dispatch on.
type bsplineHostCanal struct {
	surf     geom.BSplineSurface
	stations []geom.CanalEdgeStation
	railA    geom.Curve3 // u=0 isocurve: the foot locus on host A (the edge's first face)
	railB    geom.Curve3 // u=1 isocurve: the foot locus on host B (the edge's second face)
	seamMid  math.Point3 // station-0 on-arc midpoint (the closed-rim seam cross-section)
	r        float64
	concave  bool
	closed   bool
	// plan is the march's anchor plan: stations [plan.iEdge0, plan.iEdge1] anchor on the
	// picked edge itself; indices outside are prolong stations riding the hosts' natural
	// extension so the cap trim is contained (prolong-then-trim, OCCT
	// BRepBlend_SurfRstLineBuilder.cxx).
	plan  bsplineHostAnchorPlan
	hostA *topo.Face
	hostB *topo.Face
}

// bsplineHostArmSurface carries the canal payload through edgeFillet.armSurface (whose
// struct is frozen this wave): it IS the canal B-spline via embedding, so every generic
// consumer sees a plain surface, while the wave-G body builders key on the concrete type.
// The wrapper never reaches a built face — body builders tessellate.Unwrap to the inner BSplineSurface.
type bsplineHostArmSurface struct {
	geom.BSplineSurface
	canal *bsplineHostCanal
}

// bsplineHostCanalOf returns the wave-G canal payload of an edge fillet, or nil — the one
// dispatch key for the wave-G body builders.
func bsplineHostCanalOf(ef edgeFillet) *bsplineHostCanal {
	if w, ok := ef.armSurface.(bsplineHostArmSurface); ok {
		return w.canal
	}
	return nil
}

// bsplineHostPair reports an edge bounded by one B-spline host and one {plane | B-spline}
// host, in FACE ORDER (a = e.Faces()[0], b = e.Faces()[1]) so the rails' A/B naming always
// matches ef.a/ef.b.
func bsplineHostPair(e *topo.Edge) (a, b *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, false
	}
	nBSpline := 0
	for _, f := range faces {
		switch f.Geometry().(type) {
		case geom.BSplineSurface:
			nBSpline++
		case geom.Plane:
		default:
			return nil, nil, false // any other host mix keeps its existing path untouched
		}
	}
	if nBSpline == 0 {
		return nil, nil, false
	}
	return faces[0], faces[1], true
}

// bsplineHostArmEdge is the wave-G dispatch arm: it fires ONLY for a constant-radius pick
// on a B-spline-host edge pair and returns handled=false on any build decline, so the
// caller falls through to the byte-identical existing refusal (do-no-harm, the
// ellipticClosedRimArmEdge contract).
func bsplineHostArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if p.varying() {
		return edgeFillet{}, false
	}
	aF, bF, ok := bsplineHostPair(e)
	if !ok {
		return edgeFillet{}, false
	}
	canal, ok := buildBsplineHostCanal(body, e, aF, bF, p.r0)
	if !ok {
		return edgeFillet{}, false
	}
	return edgeFillet{
		a: aF, b: bF, edge: e,
		armSurface: bsplineHostArmSurface{BSplineSurface: canal.surf, canal: canal},
		armConcave: canal.concave,
	}, true
}

// bsplineHostSeedCentre is the two-plane closed-form ball-centre seed at the station-0
// anchor: E ∓ r(n̂A+n̂B)/(1+n̂A·n̂B) with material-outward host normals (− into the material
// for a convex pick, + into the void for a concave one). ok=false when the hosts are
// near-tangent at the anchor (the 1/(1+n̂·n̂) blowup) or a normal is unreadable.
func bsplineHostSeedCentre(e *topo.Edge, aF, bF *topo.Face, at math.Point3, r float64) (math.Point3, bool) {
	na, okA := outwardHostNormalAt(aF, at)
	nb, okB := outwardHostNormalAt(bF, at)
	if !okA || !okB {
		return math.Point3{}, false
	}
	denom := 1 + float64(na.Dot(nb))
	if denom < 1e-6 {
		return math.Point3{}, false // near-tangent hosts: no corner to round (seed denominator blowup)
	}
	sign := -1.0 // convex: centre in the material
	if ClassifyEdgeConvexity(e) == EdgeConcave {
		sign = 1.0 // concave: centre in the void; the fillet ADDS the weld ring
	}
	return at.TranslateBy(na.Add(nb).Scale(math.Scalar(sign * r / denom))), true
}

// outwardHostNormalAt is the material-outward unit normal of a host face at (the closest
// surface point to) p — the raw surface normal flipped by the face's Reversed flag.
func outwardHostNormalAt(f *topo.Face, p math.Point3) (math.Vector3, bool) {
	u, v, _ := geom.ClosestPointOnSurface(f.Geometry(), p)
	n := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	un, err := math.UnitVector3FromVector(n)
	if err != nil {
		return math.Vector3{}, false
	}
	return un.AsVector(), true
}

// bsplineHostPeriod detects a u-closed host (a swept pipe wall) and returns its u-period
// for the march's universal-cover lift; 0 for an open host. Closure is measured at the
// v-midline against the model weld.
func bsplineHostPeriod(f *topo.Face, weld float64) float64 {
	bs, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return 0
	}
	ulo, uhi := bs.UDomain()
	vlo, vhi := bs.VDomain()
	vm := (vlo + vhi) / 2
	if float64(bs.PointAt(ulo, vm).DistanceTo(bs.PointAt(uhi, vm))) > weld {
		return 0
	}
	return uhi - ulo
}

// bsplineHostMarchHost packages one host face for the geom march.
func bsplineHostMarchHost(f *topo.Face, weld float64) geom.CanalMarchHost {
	return geom.CanalMarchHost{Surf: f.Geometry(), PeriodU: bsplineHostPeriod(f, weld)}
}

// bsplineHostSeamMid is the on-arc midpoint of the station-0 cross-section (the closed-rim
// seam arc's interior point). ok=false when the two feet are antipodal about the centre.
func bsplineHostSeamMid(st geom.CanalEdgeStation, r float64) (math.Point3, bool) {
	bis, err := math.UnitVector3FromVector(
		st.Center.VectorTo(st.FootA.P).Add(st.Center.VectorTo(st.FootB.P)))
	if err != nil {
		return math.Point3{}, false
	}
	return st.Center.TranslateBy(bis.AsVector().Scale(math.Scalar(r))), true
}

// bsplineHostStationV returns the loft's v-parameters of the stations — the same
// centre-chord parameterization the loft itself uses (spineChordParams), so the end-trim
// bracketing addresses the surface at exactly the station columns.
func bsplineHostStationV(stations []geom.CanalEdgeStation) []float64 {
	centers := make([]math.Point3, len(stations))
	for i, st := range stations {
		centers[i] = st.Center
	}
	return spineChordParams(centers)
}

// bsplineHostBandOutwardFlip reports whether the lofted band's surface normal opposes the
// material-outward direction at the mid station, i.e. the station walk must be reversed —
// the ellipticRimBandOutward decision generalized to any host pair: outward is −side·(Q−C)
// with side = +1 concave (ball in the void) / −1 convex (ball in the material).
func bsplineHostBandOutwardFlip(stations []geom.CanalEdgeStation, surf geom.BSplineSurface, concave bool) (flip, ok bool) {
	j := len(stations) / 2
	vp := bsplineHostStationV(stations)
	q := surf.PointAt(0.5, vp[j])
	side := -1.0
	if concave {
		side = 1.0
	}
	want, err := math.UnitVector3FromVector(stations[j].Center.VectorTo(q).Scale(math.Scalar(-side)))
	if err != nil {
		return false, false
	}
	dot := float64(surf.NormalAt(0.5, vp[j]).Dot(want.AsVector()))
	if stdmath.Abs(dot) < 1e-9 {
		return false, false // degenerate normal at the probe station — decline
	}
	return dot < 0, true
}
