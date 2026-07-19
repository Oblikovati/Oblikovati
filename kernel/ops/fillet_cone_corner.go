// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-host trihedral corner — the cone-host campaign Slice CN3 (cone-host-corner-derivation.md §3).
// Where three equal-radius fillets meet over a host CONE (apex A, axis â into the opening, half-angle α,
// material inside → s=+1) and two planes, the corner blend is an analytic geom.Sphere of the same radius
// r, exactly like the M5 cylinder-host and SP2 sphere-host corners. Its centre is the rolling ball
// tangent to the two planes (r inside each) AND to the host cone, which becomes tangency to the coaxial
// OFFSET cone A′ = A + s·(r/sinα)·â. That collapses to a quadratic in the plane-pair line parameter with
// the cos²α-SCALED coefficients (bounded for all α ∈ (0,π/2), unlike the 1/cos²α tan²α form). A MANDATORY
// nappe filter (w·â > 0) precedes the nearer-vertex pick — C8's second root is a genuine tangency on the
// WRONG nappe of A′ that nearer-vertex alone could pick. Dispatch is ordered AFTER sphereHostCorner and
// before solvePlanarBlend, so the cylinder/sphere/planar corner paths stay byte-identical
// (unreachable-by-construction for a 1-cone-host + 2-planes corner). Concave bore (s=−1), a near-cylinder
// / near-plane cone, a grazing solve, and any inconsistency honest-reject with the exact
// "corner face must be planar" string. This file solves the corner SPHERE + its tangent points; the
// arm↔sphere weld/retrim is CN4's concern (so C2/C6/C8/D1 move to flooring at the WELD stage, corpus 60).

// coneHostCorner recognises the CN3 cone corner: exactly one host-cone face and two planar host faces.
// Returns the cone geometry, the cone FACE (its material-outward normal fixes the convex sign), and the
// two plane faces. ok=false for any other host mix — so solveBlend keeps the planar path / reject.
// Sibling of sphereHostCorner.
func coneHostCorner(faces []*topo.Face) (geom.Cone, *topo.Face, [2]*topo.Face, bool) {
	if len(faces) != 3 {
		return geom.Cone{}, nil, [2]*topo.Face{}, false
	}
	var co geom.Cone
	var coneFace *topo.Face
	var planes [2]*topo.Face
	nCo, nPl := 0, 0
	for _, f := range faces {
		if c, isCo := f.Geometry().(geom.Cone); isCo {
			co, coneFace, nCo = c, f, nCo+1
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl && nPl < 2 {
			planes[nPl], nPl = f, nPl+1
		}
	}
	return co, coneFace, planes, nCo == 1 && nPl == 2
}

// solveConeBlend solves the analytic sphere corner for a cone-host trihedral corner
// (cone-host-corner-derivation.md §3). Returns the "corner face must be planar" reject (do-no-harm) when
// no equal-r ball fits (concave bore, a near-cylinder / near-plane cone, the plane-pair line missing or
// grazing the offset cone, a wrong-nappe/absent tangency, or an inconsistent centre) — so a declined cone
// corner errors exactly as before. Mirrors solveSphereBlend for the cone host.
func solveConeBlend(v *topo.Vertex, faces []*topo.Face, co geom.Cone, coneFace *topo.Face, planes [2]*topo.Face, r float64) (*cornerBlend, error) {
	res := coneCornerResolution(v, co, planes)
	c, ok := coneHostCornerCenter(co, coneFace, planes, r, v, res)
	if !ok || !coneCornerConsistent(c, co, planes, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	corner, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: corner, tan: coneCornerTangents(faces, co, c)}, nil
}

// coneHostCornerCenter solves the ball centre tangent to the two planes and the host cone. The two plane
// constraints pin c to a line (planePairLine); the host tangency c on the offset cone A′ becomes the
// cos²α-scaled quadratic qa·t²+qb·t+qc = 0 in the line parameter (coneQuadCoeffs), whose roots pass the
// nappe filter then the nearer-vertex pick (coneCornerParam). ok=false on a concave bore (material
// outside), a near-cylinder / near-plane cone (α bands), parallel planes (no line), or the line
// clearing/grazing the offset cone. coneCornerRoot then reduces the reflected pair to the material-outward
// seed c0 (no cylinder arms witness a cone-host corner — derivation §"Reflected roots").
func coneHostCornerCenter(co geom.Cone, coneFace *topo.Face, planes [2]*topo.Face, r float64, v *topo.Vertex, res Resolution) (math.Point3, bool) {
	if sgn, ok := coneCornerMaterialSign(co, coneFace); !ok || sgn <= 0 {
		return math.Point3{}, false // concave bore (material outside): A′ = A − r/sinα·â is a follow-on slice
	}
	p0, d, ok := planePairLine(planes, r, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	if !coneAlphaInBand(co, p0, sinA, cosA, res) {
		return math.Point3{}, false // near-cylinder (r/sinα blows up) or near-plane cone (α→π/2)
	}
	ahat := co.AxisDir.AsVector()
	apexPrime := co.Apex.TranslateBy(ahat.Scale(r / sinA)) // A′ = A + s·(r/sinα)·â, s = +1 convex-external
	u := apexPrime.VectorTo(p0)                            // u = p₀ − A′
	qa, qb, qc := coneQuadCoeffs(u, d, ahat, cosA*cosA)
	t, ok := coneCornerParam(qa, qb, qc, u, d, ahat, p0, v.Point(), res)
	if !ok {
		return math.Point3{}, false
	}
	return coneCornerRoot(v, r, res, p0.TranslateBy(d.Scale(t)))
}

// coneCornerMaterialSign is the apex-safe host material-side test that fixes whether this slice may solve
// the corner. Because a cone's outward normal cos α·êr − sin α·â dotted with its own radial êr is ±cos α
// INDEPENDENT of azimuth, the sign is read from a single OFF-AXIS face sample A + â + êref (never at the
// corner vertex, which for C8 IS the apex where the radial is 0/0 — derivation §"Apex singularity"):
// s = n̂·êr > 0 ⇒ material INSIDE the cone (convex-external, this slice) ⇒ solve; s ≤ 0 ⇒ material OUTSIDE
// (concave bore) ⇒ reject. Sibling of coneHostMaterialSign, but evaluated without a picked edge so the
// centre solve does not depend on the corner's edge wiring.
func coneCornerMaterialSign(co geom.Cone, coneFace *topo.Face) (float64, bool) {
	probe := co.Apex.TranslateBy(co.AxisDir.AsVector().Add(co.Ref.AsVector())) // off-axis face sample, never the apex
	n, ok := outwardFaceNormal(coneFace, probe)
	if !ok {
		return 0, false
	}
	radial, err := coneRadialDir(co, probe)
	if err != nil {
		return 0, false
	}
	return float64(n.Dot(radial)), true
}

// coneAlphaInBand guards the two α-limit degeneracies as MODEL-relative length bands (ADR-0042), the
// length L = |A − p₀| standing in for the model scale (never |A − v|, which collapses to 0 at C8's apex
// vertex): sin α ≥ k·res.Weld()/L (else the apex shift r/sin α blows up — a true cylinder host takes the
// M5 path) and cos α ≥ k·res.Weld()/L (else a near-plane cone, α→π/2). Reuses the arm's coneAlphaBandCoef.
func coneAlphaInBand(co geom.Cone, p0 math.Point3, sinA, cosA float64, res Resolution) bool {
	l := stdmath.Max(float64(co.Apex.DistanceTo(p0)), res.Weld())
	aband := coneAlphaBandCoef * res.Weld() / l
	return sinA >= aband && cosA >= aband
}

// coneQuadCoeffs are the cos²α-SCALED coefficients of the host-tangency quadratic qa·t²+qb·t+qc = 0
// (cone-host-corner-derivation.md §3): c on the offset cone ⇔ |w|²cos²α = (w·â)² with w(t) = u + t·d.
// Scaling by cos²α (rather than the equivalent tan²α form) keeps every coefficient bounded for ALL
// α ∈ (0,π/2); qa may legitimately be NEGATIVE (line ∥ axis ⇒ qa = −sin²α·|d|², C8) or ~zero.
func coneQuadCoeffs(u, d, ahat math.Vector3, cos2A float64) (qa, qb, qc float64) {
	ud, ua := float64(u.Dot(d)), float64(u.Dot(ahat))
	dd, da := float64(d.Dot(d)), float64(d.Dot(ahat))
	uu := float64(u.Dot(u))
	qa = cos2A*dd - da*da
	qb = 2 * (cos2A*ud - ua*da)
	qc = cos2A*uu - ua*ua
	return qa, qb, qc
}

// coneCornerParam solves the host-tangency quadratic for the line parameter and picks the corner ball's
// root: it enumerates the real roots (coneRealRoots), keeps only those on the OPENING-side nappe of A′
// (w(t)·â > 0 — the mandatory nappe filter, load-bearing for C8), THEN takes the one nearer the corner
// vertex. ok=false when no root survives the nappe filter (all tangencies on the far nappe) — an honest
// reject rather than a wrong-nappe centre.
func coneCornerParam(qa, qb, qc float64, u, d, ahat math.Vector3, p0, vertex math.Point3, res Resolution) (float64, bool) {
	roots, ok := coneRealRoots(qa, qb, qc, d, res)
	if !ok {
		return 0, false
	}
	band := curvedCornerBandK * res.Weld()
	var kept []float64
	for _, t := range roots {
		if float64(u.Add(d.Scale(t)).Dot(ahat)) > band { // nappe filter: w(t)·â > 0 (opening-side nappe of A′)
			kept = append(kept, t)
		}
	}
	return nearerKeptRoot(kept, p0, d, vertex)
}

// coneRealRoots returns every real root of the cone host-tangency quadratic. qa may be NEGATIVE (C8) or
// ~zero: when |qa| is below the dimensionless axis floor (line on the offset cone's ruling-direction
// cone) it falls back to the single linear root t = −qc/qb, rejecting when |qb| is also below band. A
// negative discriminant (the line clears the offset cone) or two roots whose POINT-space separation falls
// below the grazing band (curvedCornerBandK·res.Weld()) honest-rejects rather than emit an
// ill-conditioned centre (cone-host-corner-derivation.md §"Grazing/coalescing roots").
func coneRealRoots(qa, qb, qc float64, d math.Vector3, res Resolution) ([]float64, bool) {
	dd := float64(d.Dot(d))
	if stdmath.Abs(qa) < curvedCornerAxisTiny*dd {
		if stdmath.Abs(qb) < curvedCornerBandK*res.Weld()*stdmath.Sqrt(dd) {
			return nil, false // line on the offset cone (both leading coefficients vanish) — degenerate
		}
		return []float64{-qc / qb}, true
	}
	disc := qb*qb - 4*qa*qc
	if disc < 0 {
		return nil, false // the offset line clears the offset cone: no real tangency
	}
	root := stdmath.Sqrt(disc)
	t1, t2 := (-qb-root)/(2*qa), (-qb+root)/(2*qa)
	if stdmath.Abs(t1-t2)*float64(d.Length()) < curvedCornerBandK*res.Weld() {
		return nil, false // grazing/coalescing roots: the corner is a geometric degeneracy
	}
	return []float64{t1, t2}, true
}

// nearerKeptRoot returns the nappe-surviving root whose point p₀+t·d lies closest to the corner vertex
// (the physical ball sits in the wedge adjacent to v). ok=false when no root survived the nappe filter.
func nearerKeptRoot(ts []float64, p0 math.Point3, d math.Vector3, vertex math.Point3) (float64, bool) {
	if len(ts) == 0 {
		return 0, false
	}
	best := ts[0]
	bestD := float64(p0.TranslateBy(d.Scale(best)).DistanceTo(vertex))
	for _, t := range ts[1:] {
		if dv := float64(p0.TranslateBy(d.Scale(t)).DistanceTo(vertex)); dv < bestD {
			best, bestD = t, dv
		}
	}
	return best, true
}

// coneCornerRoot reduces the reflected-root family to the material-outward seed c0. A cone-host corner
// bounds only Cone∧Plane and Plane∧Plane edges — never a Plane∧Cylinder LINE arm — so the reflected-root
// station witness (cornerCylinderArms) is empty and the selector keeps the legacy c0, which the oracle
// confirms is correct in all four cases (cone-host-corner-derivation.md §"Reflected roots"). Any cylinder
// arm present is out of scope: honest-reject rather than pick a possibly-wrong root. Sibling of
// sphereCornerRoot; also the certification that cornerCylinderArms does NOT claim the Cone∧Plane ruling
// edge (it matches on the cylinder FACE type via cylinderPlaneEdge, so a cone host gives it no arms).
func coneCornerRoot(v *topo.Vertex, r float64, res Resolution, c0 math.Point3) (math.Point3, bool) {
	if len(cornerCylinderArms(v, r, res)) != 0 {
		return math.Point3{}, false
	}
	return c0, true
}

// coneCornerConsistent verifies the solved centre truly sits r from both planes (two-sided |dist| = r,
// per the N7 reflected-root lesson) and r from the host cone wall (the exact signed cone distance
// coneSignedDistance = r). A magnitude failure makes solveConeBlend return the do-no-harm reject rather
// than emit a bad corner. Sibling of sphereCornerConsistent.
func coneCornerConsistent(c math.Point3, co geom.Cone, planes [2]*topo.Face, r float64, res Resolution) bool {
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if stdmath.Abs(stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n)))-r) > res.Weld() {
			return false // not at distance r from this plane (either side)
		}
	}
	return stdmath.Abs(coneSignedDistance(co, c)-r) < res.Weld()
}

// coneSignedDistance is the exact signed distance from point c to the host cone wall, POSITIVE INSIDE the
// cone: dist = (w₀·â)·sin α − |w₀ − (w₀·â)â|·cos α with w₀ = c − A (the axial reach times sin α minus the
// radial reach times cos α — the projection of w₀ onto the inward wall normal). A valid corner centre
// sits at dist = r.
func coneSignedDistance(co geom.Cone, c math.Point3) float64 {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	ahat := co.AxisDir.AsVector()
	w := co.Apex.VectorTo(c)
	axial := float64(w.Dot(ahat))
	perp := float64(w.Sub(ahat.Scale(axial)).Length())
	return axial*sinA - perp*cosA
}

// coneCornerTangents places the ball's tangent point on each host face, keyed by face id: on a plane it is
// the perpendicular foot of the centre (planeFootPoint, valid either side); on the host cone it is the
// meridian foot T (coneTangentPoint) — the degenerate host-tangency pinch vertex where the corner ball
// touches the cone. Sibling of sphereCornerTangents.
func coneCornerTangents(faces []*topo.Face, co geom.Cone, c math.Point3) map[uint64]math.Point3 {
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); ok {
			tan[f.ID()] = planeFootPoint(f, c)
			continue
		}
		tan[f.ID()] = coneTangentPoint(co, c)
	}
	return tan
}

// coneTangentPoint is the point where the corner ball at centre c touches the host cone — the meridian
// foot T = A + (w₀·ĝ)·ĝ with ĝ = cos α·â + sin α·ê the meridian ruling direction through c (ê the outward
// radial unit of w₀ = c − A). Exact identity (used by the tests): T − c has axial component r·sin α toward
// the apex, i.e. (T − c)·â = −r·sin α. Falls back to c when c lies on the axis (degenerate — unreachable
// once coneCornerConsistent holds, since a valid centre is off-axis by construction).
func coneTangentPoint(co geom.Cone, c math.Point3) math.Point3 {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	ahat := co.AxisDir.AsVector()
	w := co.Apex.VectorTo(c)
	axial := float64(w.Dot(ahat))
	radial, err := math.UnitVector3FromVector(w.Sub(ahat.Scale(axial)))
	if err != nil {
		return c
	}
	g := ahat.Scale(cosA).Add(radial.AsVector().Scale(sinA)) // ĝ = cos α·â + sin α·ê
	return co.Apex.TranslateBy(g.Scale(float64(w.Dot(g))))
}

// coneCornerResolution builds the model-relative weld tolerance for the cone corner from its own geometry
// (the vertex, the cone apex, and the two plane origins) — ADR-0042, so the tangency checks scale with the
// model rather than a bare 1e-6. Sibling of sphereCornerResolution.
func coneCornerResolution(v *topo.Vertex, co geom.Cone, planes [2]*topo.Face) Resolution {
	return ResolutionForPoints([]math.Point3{
		v.Point(), co.Apex,
		planes[0].Geometry().(geom.Plane).Origin,
		planes[1].Geometry().(geom.Plane).Origin,
	})
}
