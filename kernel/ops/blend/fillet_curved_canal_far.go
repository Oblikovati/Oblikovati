// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The canal arms' GEOMETRIC far termini (M6' C4 W3c/F1, derivation: .superpowers/sdd/
// canal-far-runout-derivation.md). W4 found the reused "far end = host-loop crossing" identity
// (armRulingEnd) invalid on the REAL N7 body: each arm's far end is a smooth interior RUNOUT lying on
// NO host loop (measured gap 50u). The truth (DRAWEXE-verified): each arm ends where its terminating
// face F_far cuts the fillet band — each host rail ends at rail∩F_far and the terminal boundary is
// armSurface∩F_far. For a plane F_far ⊥ the spine (s_4, s_10) that section is the radius-r cross-section
// arc (farCrossSectionArc verbatim); for a plane parallel to the torus axis (s_5) it is the classical
// SPIRIC section (geom.SpiricArc, exact on both surfaces). This file replaces the loop-crossing outer
// trim with the closed-form section for canal arms; the single-ball path (armHostContactRail /
// armRulingEnd) is UNTOUCHED. A bare-face fixture (no wired edge, so no F_far) falls back to the reused
// loop-crossing rails, so the W2 fixture tests stay byte-identical.

// canalArmRailsAndTerminal builds one canal arm's two host contact rails (each oriented outer→tHost)
// and its terminal cross-section (the arm face's far boundary curve), at the arm's reflected weld wi.
// On the REAL body (arm.edge wired) the outer ends and terminal are the geometric section against the
// arm's terminating face F_far (canalFarTerminal); when F_far is unavailable (a bare-face fixture, or a
// terminating face the closed forms do not cover) it falls back to the reused loop-crossing rails +
// farCrossSectionArc. Both host rails and the terminal SHARE their feet by construction — the terminal's
// endpoints ARE the two rails' outer ends — which is the shared-edge identity the loop welds on.
func canalArmRailsAndTerminal(arm edgeFillet, set armSetback, wi cornerWeld, res opstol.Resolution) (endSeg, endSeg, endSeg, bool) {
	if h0, h1, term, ok := canalFarTerminal(arm, set, wi, res); ok {
		return h0, h1, term, true
	}
	h0, h1, ok := canalArmHostRails(arm, set, wi, res)
	if !ok {
		return endSeg{}, endSeg{}, endSeg{}, false
	}
	// Same honest-reject gate as canalTerminalSection's arc branch: the reused loop-crossing rails' outer
	// ends must be a genuine radius-r cross-section before farCrossSectionArc snaps an arc through them
	// (inert for the W2/bare-face fixtures, whose feet ARE a ⊥-spine cross-section).
	if !feetOnFarCrossSection(set.arm, wi.radius, h0.from, h1.from, res.Weld()*wi.radius) {
		return endSeg{}, endSeg{}, endSeg{}, false
	}
	far, ok := farCrossSectionArc(set.arm, wi.radius, h0.from, h1.from)
	return h0, h1, far, ok
}

// canalFarTerminal builds the arm's host rails + terminal section against its terminating face F_far
// (derivation §3-4). Returns ok=false — so canalArmRailsAndTerminal reuses the loop-crossing path —
// when the arm carries no wired edge, F_far is not identifiable, a host rail's section cannot be built,
// or the terminal section declines. The two rails' outer feet feed the terminal so their endpoints match.
func canalFarTerminal(arm edgeFillet, set armSetback, wi cornerWeld, res opstol.Resolution) (endSeg, endSeg, endSeg, bool) {
	ffar, ok := canalFarFace(arm, wi)
	if !ok {
		return endSeg{}, endSeg{}, endSeg{}, false
	}
	tol := res.Weld() * wi.radius
	h0, ok0 := canalFarRail(arm.a, set, set.railDir0, ffar, wi, res, tol)
	h1, ok1 := canalFarRail(arm.b, set, set.railDir1, ffar, wi, res, tol)
	if !ok0 || !ok1 {
		return endSeg{}, endSeg{}, endSeg{}, false
	}
	term, ok := canalTerminalSection(set.arm, wi.radius, ffar, h0, h1, tol)
	if !ok {
		return endSeg{}, endSeg{}, endSeg{}, false
	}
	return h0, h1, term, true
}

// canalFarFace identifies the arm's terminating face F_far (derivation §1): the original body face at
// the arm edge's FAR vertex that is neither host and is transverse to the edge tangent there. Exactly
// one PLANE must survive the filter; zero, ambiguous, or a non-plane candidate → false (honest-decline
// to the loop-crossing path, never a snapped section).
func canalFarFace(arm edgeFillet, wi cornerWeld) (geom.Plane, bool) {
	if arm.edge == nil {
		return geom.Plane{}, false
	}
	far := farEndVertex(arm.edge, wi.center)
	tan, ok := edgeTangentAt(arm.edge, far)
	if !ok {
		return geom.Plane{}, false
	}
	return transverseNonHostPlane(far, arm.a, arm.b, tan)
}

// farEndVertex is the arm edge's endpoint farther from the corner centre c — the far runout terminus
// (the near endpoint is the trihedral corner).
func farEndVertex(e *topo.Edge, c math.Point3) *topo.Vertex {
	s, t := e.StartVertex(), e.EndVertex()
	if s.Point().DistanceTo(c) >= t.Point().DistanceTo(c) {
		return s
	}
	return t
}

// edgeTangentAt returns the unit tangent of edge e at the endpoint v (the domain end matching v), the
// direction the F_far transversality test measures against. Declines when the tangent is degenerate.
func edgeTangentAt(e *topo.Edge, v *topo.Vertex) (math.UnitVector3, bool) {
	lo, hi := e.Geometry().Domain()
	at := lo
	if e.EndVertex().ID() == v.ID() {
		at = hi
	}
	u, err := math.UnitVector3FromVector(e.Geometry().TangentAt(at))
	if err != nil {
		return math.UnitVector3{}, false
	}
	return u, true
}

// transverseNonHostPlane returns the unique original face at v that is a plane, is neither host a/b, and
// is transverse to the edge tangent tan (|tan·n| above the scale-free sinFloor). It is F_far only when
// exactly one such plane exists; zero or several → false (the far end is not a simple planar runout).
func transverseNonHostPlane(v *topo.Vertex, a, b *topo.Face, tan math.UnitVector3) (geom.Plane, bool) {
	var found geom.Plane
	n := 0
	for _, f := range facesAround(v) {
		if f.ID() == a.ID() || f.ID() == b.ID() {
			continue
		}
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(float64(tan.AsVector().Dot(pl.Normal()))) <= sinFloor {
			continue
		}
		found, n = pl, n+1
	}
	return found, n == 1
}

// facesAround gathers the distinct faces touching vertex v (via its edges), the original-body faces the
// far vertex is a corner of.
func facesAround(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var faces []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				faces = append(faces, f)
			}
		}
	}
	return faces
}

// canalFarRail builds one host contact rail whose OUTER end is the geometric section rail∩F_far
// (derivation §3), the canal replacement for armHostContactRail's loop-crossing terminator. A cylinder
// arm's ruling meets F_far in a line∩plane point; a torus arm's WALL rail is curvedHostArc verbatim
// (already ending at the far-vertex azimuth); its CAP rail is that arc extended past the far azimuth by
// δ_cap (extendCapRail) because the wider wall crosses F_far at a smaller azimuth than the cap circle.
func canalFarRail(host *topo.Face, set armSetback, railDir math.UnitVector3, ffar geom.Plane, wi cornerWeld, res opstol.Resolution, tol float64) (endSeg, bool) {
	tHost := endpointOf(wi.center, wi.radius, railDir)
	switch s := set.arm.(type) {
	case geom.Cylinder:
		return cylinderFarRail(s, tHost, ffar)
	case geom.Torus:
		return torusFarRail(host, s, tHost, ffar, wi, res, tol)
	}
	return endSeg{}, false
}

// cylinderFarRail terminates a cylinder arm's straight ruling (fixed point tHost, direction = arm axis)
// at its intersection with F_far (line∩plane, derivation §3). Declines when the ruling is (near-)parallel
// to F_far — |â·n̂| ≤ sinFloor, the scale-free sine floor: the ruling grazes the plane and the crossing is
// not robust (the arm never cleanly terminates on it). Both â (arm axis) and n̂ (F_far normal, UAxis×VAxis)
// are unit, so LinePlaneIntersection's direction·normal test IS |â·n̂| — passing sinFloor as its tol makes
// it the derivation's sine-floor guard. Inert on-corpus (|â·n̂|=1, the ruling meets F_far square).
func cylinderFarRail(cyl geom.Cylinder, tHost math.Point3, ffar geom.Plane) (endSeg, bool) {
	line, err := geom.NewLine(tHost, cyl.AxisDir.AsVector())
	if err != nil {
		return endSeg{}, false
	}
	outer, ok := geom.LinePlaneIntersection(line, ffar, sinFloor)
	if !ok {
		return endSeg{}, false
	}
	return endSeg{from: outer, to: tHost}, true
}

// torusFarRail builds a torus arm's contact arc (curvedHostArc, whose PointAt(1) is the setback tangent
// point tHost). The wall rail (cylinder host) already ends at the far-vertex azimuth and is kept
// verbatim; the CAP rail (plane host) is extended past that azimuth by δ_cap. Declines when the arc
// misses tHost.
func torusFarRail(host *topo.Face, tor geom.Torus, tHost math.Point3, ffar geom.Plane, wi cornerWeld, res opstol.Resolution, tol float64) (endSeg, bool) {
	arc, ok := curvedHostArc(host.Geometry(), tor, wi, res)
	if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
		return endSeg{}, false
	}
	if _, isCap := host.Geometry().(geom.Plane); isCap {
		arc, ok = extendCapRail(arc, tor, ffar)
		if !ok {
			return endSeg{}, false
		}
	}
	return endSeg{from: arc.PointAt(0), to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, true
}

// extendCapRail extends a torus cap rail's far end past the far-vertex azimuth by
// δ_cap = asin(d/ρ) − asin(d/(ρ+r)) (derivation §3): the cap section (circle radius ρ ∩ F_far) lands
// BEYOND the wall ruling (radius ρ+r) because the wider wall crosses the axis-parallel F_far at a
// smaller azimuth. d is the torus-axis→F_far distance, ρ the MajorRadius, r the MinorRadius; every term
// is a closed form of the torus + F_far, no tuned constant. Only the far end moves — the near (tHost)
// end at angle StartAngle+SweepAngle is unchanged. Declines when F_far is secant to the cap circle.
func extendCapRail(arc geom.Arc3d, tor geom.Torus, ffar geom.Plane) (geom.Arc3d, bool) {
	d, ok := capPlaneAxisDistance(tor, ffar)
	if !ok {
		return geom.Arc3d{}, false // degenerate F_far normal — skip the extension rather than move by δ=0
	}
	rho, wallR := tor.MajorRadius, tor.MajorRadius+tor.MinorRadius
	if d >= rho {
		return geom.Arc3d{}, false
	}
	delta := stdmath.Asin(d/rho) - stdmath.Asin(d/wallR)
	dir := signOf(arc.SweepAngle) // extend the far end AWAY from the near end at angle StartAngle+SweepAngle
	ext, err := geom.NewArc3d(arc.Center, arc.Normal.AsVector(), arc.RefDir.AsVector(), arc.Radius,
		arc.StartAngle-dir*delta, arc.SweepAngle+dir*delta)
	if err != nil {
		return geom.Arc3d{}, false
	}
	return ext, true
}

// capPlaneAxisDistance is the perpendicular distance from the torus axis to the axis-parallel plane
// F_far — |(ffar.Origin − Center)·n̂_F|, with n̂_F ⊥ the axis in the oblique case. Declines (false) on a
// degenerate F_far normal so extendCapRail skips the extension outright, rather than reading a silent
// zero distance (δ_cap = 0) that would leave the cap rail un-extended without anyone noticing.
func capPlaneAxisDistance(tor geom.Torus, ffar geom.Plane) (float64, bool) {
	n, err := math.UnitVector3FromVector(ffar.Normal())
	if err != nil {
		return 0, false
	}
	return stdmath.Abs(float64(tor.Center.VectorTo(ffar.Origin).Dot(n.AsVector()))), true
}

// canalTerminalSection builds the arm's terminal boundary between the two host feet (h0.from, h1.from):
// the radius-r cross-section ARC when F_far is ⊥ the spine (farCrossSectionArc — s_4/s_10, derivation
// §4), or the SPIRIC section of the torus by the axis-parallel plane when it is oblique (s_5 —
// geom.SpiricArc, exact on both surfaces, NOT a circular arc). The arc branch is GATED by
// feetOnFarCrossSection: farCrossSectionArc 3-point-fits SOME arc through ANY two feet and never checks
// they are a true radius-r cross-section about the far ball centre, so an oblique F_far (feet at unequal
// distances from m_far — e.g. a future torus arm whose terminating plane is neither ∥-axis nor ⊥-spine)
// would silently snap a wrong surface; the gate declines it to the honest-reject fallback instead.
func canalTerminalSection(armSurf geom.Surface, r float64, ffar geom.Plane, h0, h1 endSeg, tol float64) (endSeg, bool) {
	if tor, ok := armSurf.(geom.Torus); ok && planeParallelToAxis(tor, ffar) {
		return spiricTerminalSection(tor, ffar, h0.from, h1.from, tol)
	}
	if cyl, ok := armSurf.(geom.Cylinder); ok && !planePerpToDir(ffar, cyl.AxisDir) {
		return endSeg{}, false // oblique cylinder terminus — not a radius-r cross-section, out of closed-form scope
	}
	if !feetOnFarCrossSection(armSurf, r, h0.from, h1.from, tol) {
		return endSeg{}, false // feet are NOT a radius-r cross-section about m_far (oblique F_far) — never snap an arc
	}
	return farCrossSectionArc(armSurf, r, h0.from, h1.from)
}

// feetOnFarCrossSection reports whether BOTH terminal feet lie on the radius-r cross-section circle about
// the arm's far ball centre m_far (the spine point at the runout) — the precondition farCrossSectionArc
// silently assumes but never verifies. For a true cross-section (F_far ⊥ the spine) both feet share the
// SAME ball centre and sit at distance r from it; an OBLIQUE F_far puts the second foot at a different
// spine point (unequal distance), so |dist(m_far, footᵢ) − r| exceeds tol and the arc branch declines
// rather than snapping an arc through feet that are not a cross-section (mutation-proven: 6.33 vs r=5).
func feetOnFarCrossSection(arm geom.Surface, r float64, foot0, foot1 math.Point3, tol float64) bool {
	mFar, ok := armBallCenter(arm, foot0)
	if !ok {
		return false
	}
	d0 := stdmath.Abs(float64(mFar.DistanceTo(foot0)) - r)
	d1 := stdmath.Abs(float64(mFar.DistanceTo(foot1)) - r)
	return d0 <= tol && d1 <= tol
}

// planePerpToDir reports whether the plane pl is perpendicular to direction d (its normal is parallel to
// d, |n̂·d̂| ≈ 1) — the ⊥ case where the arm's terminal cross-section is a radius-r circular arc.
func planePerpToDir(pl geom.Plane, d math.UnitVector3) bool {
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return false
	}
	return stdmath.Abs(float64(n.AsVector().Dot(d.AsVector()))) > 1-sinFloor
}

// planeParallelToAxis reports whether ffar's normal is perpendicular to the torus axis (the plane runs
// parallel to the axis) — the oblique/spiric case, as opposed to a cap plane ⊥ the axis.
func planeParallelToAxis(tor geom.Torus, ffar geom.Plane) bool {
	n, err := math.UnitVector3FromVector(ffar.Normal())
	if err != nil {
		return false
	}
	return stdmath.Abs(float64(n.AsVector().Dot(tor.AxisDir.AsVector()))) < sinFloor
}

// spiricTerminalSection wraps the torus∩F_far spiric section (torusPlaneSection) as the arm's terminal
// endSeg from the wall foot to the cap foot. arc=false marks it as a non-circular curve, so the loop
// assembler reverses it via ReverseCurve3 (not the three-point arc reconstruction).
func spiricTerminalSection(tor geom.Torus, ffar geom.Plane, wallFoot, capFoot math.Point3, tol float64) (endSeg, bool) {
	sa, ok := torusPlaneSection(tor, ffar, wallFoot, capFoot, tol)
	if !ok {
		return endSeg{}, false
	}
	return endSeg{from: wallFoot, to: capFoot, curve: sa, mid: sa.PointAt(0.5)}, true
}

// torusPlaneSection is the spiric section of tor by the axis-parallel plane ffar, restricted to the tube
// arc from foot a (tube angle va) to foot b (vb), on the branch that passes through the two feet
// (derivation §4). Its endpoints are verified against the feet (the shared-edge identity — the feet are
// the host rails' outer ends); a branch/endpoint mismatch → false rather than a curve off the feet.
func torusPlaneSection(tor geom.Torus, ffar geom.Plane, a, b math.Point3, tol float64) (geom.SpiricArc, bool) {
	phi, m, k, c := geom.TorusSectionCoeffs(tor, ffar)
	va, ok0 := tubeAngleOf(tor, a, tol)
	vb, ok1 := tubeAngleOf(tor, b, tol)
	if !ok0 || !ok1 {
		return geom.SpiricArc{}, false
	}
	for _, branch := range [2]float64{-1, 1} {
		sa := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: branch, V0: va, V1: vb}
		if float64(sa.PointAt(0).DistanceTo(a)) <= tol && float64(sa.PointAt(1).DistanceTo(b)) <= tol {
			return sa, true
		}
	}
	return geom.SpiricArc{}, false
}

// tubeAngleOf is the torus tube angle v of a point p ON the torus: sin v = (p−Center)·axis / r,
// cos v = (|in-plane (p−Center)| − ρ) / r, so v = atan2(sin v, cos v). Declines (false) when p is not on
// the tube — its distance from the spine circle, r·hypot(sin v, cos v), differs from r by more than tol —
// so a caller never reads a tube angle off a foot that does not lie on the torus (which would place the
// spiric branch's endpoints on the wrong meridian).
func tubeAngleOf(tor geom.Torus, p math.Point3, tol float64) (float64, bool) {
	axis := tor.AxisDir.AsVector()
	d := tor.Center.VectorTo(p)
	along := float64(d.Dot(axis))
	inPlane := d.Sub(axis.Scale(along))
	sv := along / tor.MinorRadius
	cv := (float64(inPlane.Length()) - tor.MajorRadius) / tor.MinorRadius
	if stdmath.Abs(stdmath.Hypot(sv, cv)-1)*tor.MinorRadius > tol { // p off the tube (dist to spine ≠ r)
		return 0, false
	}
	return stdmath.Atan2(sv, cv), true
}
