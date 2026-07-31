// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED-FORM rolling-ball spine of a rim where a geom.EllipticalCylinder wall meets a
// geom.Cone cap (OCCT blend/tolblend_simple B4/B7/B8/C2/C3: a circle extruded obliquely — the
// elementarised right elliptical cylinder — glued to a cone whose slope EQUALS the extrusion
// tilt). The construction extends the plane-cap elliptic rim spine (fillet_elliptic_rim_spine.go):
// the wall is still a translation surface, so its unit normal N̂(u) is constant along every ruling
// and a ball tangent to the wall on the material side has its centre on the offset ruling
//
//	C(v) = P(u,0) + v·â + side·σw·r·N̂(u).
//
// The cap is now a CONE, not a plane: the outward offset of a cone by r is the coaxial cone with
// the same half-angle and the apex receded by r/sin(α) along the axis, so "C at distance r from
// the cone" is |C−A′|² = ((C−A′)·ĉ)²/cos²α — a QUADRATIC in v instead of the plane's linear
// solve, still exact. Both feet follow in closed form (the wall foot is the ruling point, the
// cone foot the perpendicular drop C − side·σc·r·n̂cone).
//
// The family's defining degeneracy: when the extrusion tilt equals the cone slope, the wall
// ruling at ONE azimuth u* passes through the cone apex and lies ON the cone — the two hosts are
// tangent along it, the quadratic collapses (both leading coefficients → 0), and the physical
// limit station is the rim point itself with a ZERO-width section (the teardrop pinch the
// pinched-canal loft carries; see kernel/geom/canal_pinched_loft.go). solveStation returns that
// limit exactly rather than failing.

// ellipticConeRimSpine binds the closed-form spine to one EllipticalCylinder∧Cone rim.
type ellipticConeRimSpine struct {
	ec   geom.EllipticalCylinder
	cone geom.Cone
	r    float64
	// side is +1 for a CONCAVE rim (ball rolls in the reentrant void, the fillet adds material) —
	// the only side this vein supports; a convex EllipticalCylinder∧Cone rim declines honestly.
	side float64
	// sigW/sigC are +1 when the host's GEOMETRIC outward normal is its material-outward normal
	// (derived by probing the SOLID, never from Reversed flags — the imported oblique-extrusion
	// orientation defect the plane-cap spine already documents).
	sigW, sigC float64
	nRim       math.UnitVector3 // rim plane unit normal (the rim is a circular arc on both hosts)
	cRim       float64          // rim plane offset: n̂Rim·X = cRim on the rim
	denRim     float64          // n̂Rim·â — ruling-vs-rim-plane tilt (≠ 0, guarded at bind time)
}

// ellipticConeStation is one exact spine station: the ball centre and its two host feet, plus the
// wall parameter v of the wall foot (the retrim gates read it).
type ellipticConeStation struct {
	center, wallFoot, coneFoot math.Point3
	v                          float64
}

// newEllipticConeRimSpine binds the spine, deriving both material signs and the rim side
// GEOMETRICALLY off the solid. ok=false — every caller falls through to the byte-identical flat
// refusal — when the rim curve is not a circular arc, the cap plane of the rim is parallel to the
// wall rulings, a material side is undecidable, or the rim is not cleanly concave.
func newEllipticConeRimSpine(body *topo.Body, e *topo.Edge, ec geom.EllipticalCylinder, cone geom.Cone, wallF, coneF *topo.Face, r float64) (ellipticConeRimSpine, bool) {
	arc, isArc := e.Geometry().(geom.Arc3d)
	if !isArc {
		return ellipticConeRimSpine{}, false // the cone-cap vein's rims are circular arcs (B4..C3)
	}
	nRim := arc.Normal
	den := float64(nRim.AsVector().Dot(ec.AxisDir.AsVector()))
	if stdmath.Abs(den) < ellipticRimAxisTiltTol {
		return ellipticConeRimSpine{}, false // rim plane ∥ the rulings — no rim station solve
	}
	sigW, ok := ellipticWallMaterialSign(body, e, ec, wallF, r)
	if !ok {
		return ellipticConeRimSpine{}, false
	}
	sigC, ok := coneCapMaterialSign(body, e, cone, coneF, r)
	if !ok {
		return ellipticConeRimSpine{}, false
	}
	s := ellipticConeRimSpine{
		ec: ec, cone: cone, r: r, side: 1, sigW: sigW, sigC: sigC,
		nRim: nRim, cRim: float64(math.P3(0, 0, 0).VectorTo(arc.Center).Dot(nRim.AsVector())), denRim: den,
	}
	if !ellipticConeRimIsConcave(body, e, s) {
		return ellipticConeRimSpine{}, false // convex (or unclean) EllipticalCylinder∧Cone rim — unsupported
	}
	return s, true
}

// coneCapMaterialSign returns +1 when the cone's geometric outward normal is material-outward,
// −1 when the material lies outside the cone. It probes the solid a quarter radius either side of
// a point stepped from the rim midpoint along the cone toward its FAR boundary (the cap's other
// closed edge), clear of the rim's own corner — the cone sibling of ellipticWallMaterialSign.
func coneCapMaterialSign(body *topo.Body, e *topo.Edge, cone geom.Cone, coneF *topo.Face, r float64) (float64, bool) {
	_, farV := wallSeam(coneF, e, e.StartVertex())
	if farV == nil {
		return 0, false
	}
	mid := edgeMidpoint(e)
	u, vRim := cone.ParamAt(mid)
	_, vFar := cone.ParamAt(farV.Point())
	along := ellipticRimProbeFraction * r * stdmath.Copysign(1, vFar-vRim)
	probe := cone.PointAt(u, vRim+along)
	step := cone.NormalAt(u, vRim+along).Scale(ellipticRimProbeFraction * r)
	in := PointInsideBody(body, probe.TranslateBy(step.Scale(-1)))
	out := PointInsideBody(body, probe.TranslateBy(step))
	if in == out {
		return 0, false // undecidable: cap thinner than the probe, or a degenerate pick
	}
	if in {
		return 1, true
	}
	return -1, true
}

// ellipticConeRimIsConcave runs the mixed-quadrant probe (ellipticRimConvexitySide's
// discriminator) with both hosts' MATERIAL-outward normals at the rim midpoint: the two mixed
// quadrants ±(nB−nA) are material exactly when the rim is concave (the hosts' union is the
// solid). false for a convex or unclean dihedral — the caller declines.
func ellipticConeRimIsConcave(body *topo.Body, e *topo.Edge, s ellipticConeRimSpine) bool {
	mid := edgeMidpoint(e)
	uw, vw := s.ec.ParamAt(mid)
	uc, vc := s.cone.ParamAt(mid)
	nA := s.ec.NormalAt(uw, vw).Scale(s.sigW)
	nB := s.cone.NormalAt(uc, vc).Scale(s.sigC)
	mixed, err := math.UnitVector3FromVector(nB.Sub(nA))
	if err != nil {
		return false // hosts tangent at the rim midpoint — no dihedral to classify
	}
	step := mixed.AsVector().Scale(ellipticRimProbeFraction * s.r)
	one := PointInsideBody(body, mid.TranslateBy(step))
	other := PointInsideBody(body, mid.TranslateBy(step.Scale(-1)))
	return one && other // both mixed quadrants material ⇔ concave
}

// offsetConeApex is the apex of the cone offset r to the ball's side: receded by r/sin(α) along
// the axis (the parallel surface of a cone is the coaxial equal-angle cone).
func (s ellipticConeRimSpine) offsetConeApex() math.Point3 {
	shift := -s.side * s.sigC * s.r / stdmath.Sin(s.cone.HalfAngle)
	return s.cone.Apex.TranslateBy(s.cone.AxisDir.AsVector().Scale(shift))
}

// vRimAt is the wall parameter v where the ruling at azimuth u crosses the rim plane — the
// root-selection anchor (the physical station is the one nearest the rim) and the tangency limit.
func (s ellipticConeRimSpine) vRimAt(u float64) float64 {
	p0 := s.ec.PointAt(u, 0)
	return (s.cRim - float64(math.P3(0, 0, 0).VectorTo(p0).Dot(s.nRim.AsVector()))) / s.denRim
}

// solveStation returns the exact rolling-ball station at wall azimuth u. The quadratic in v picks
// the root nearest the rim on the cone's forward nappe; the doubly-degenerate tangency azimuth
// (both coefficients ≈ 0: the offset ruling lies ON the offset cone) returns the pinch limit
// v = vRim exactly. ok=false when no admissible root exists.
func (s ellipticConeRimSpine) solveStation(u float64) (ellipticConeStation, bool) {
	nW, err := math.UnitVector3FromVector(s.ec.NormalAt(u, 0))
	if err != nil {
		return ellipticConeStation{}, false
	}
	off := s.side * s.sigW * s.r
	v, ok := s.solveOffsetConeCrossing(u, nW.AsVector().Scale(off))
	if !ok {
		return ellipticConeStation{}, false
	}
	wallFoot := s.ec.PointAt(u, v)
	center := wallFoot.TranslateBy(nW.AsVector().Scale(off))
	coneFoot, ok := s.coneFootOf(center)
	if !ok {
		return ellipticConeStation{}, false
	}
	return ellipticConeStation{center: center, wallFoot: wallFoot, coneFoot: coneFoot, v: v}, true
}

// solveOffsetConeCrossing solves |B0+v·â|² = ((B0+v·â)·ĉ)²/cos²α for the wall parameter v of the
// ball centre: the crossing of the offset ruling with the offset cone. B0 is the ruling's v=0
// point offset to the ball side, relative to the offset apex.
func (s ellipticConeRimSpine) solveOffsetConeCrossing(u float64, offVec math.Vector3) (float64, bool) {
	apex := s.offsetConeApex()
	b0 := apex.VectorTo(s.ec.PointAt(u, 0).TranslateBy(offVec))
	a := s.ec.AxisDir.AsVector()
	c := s.cone.AxisDir.AsVector()
	inv := 1 / (stdmath.Cos(s.cone.HalfAngle) * stdmath.Cos(s.cone.HalfAngle))
	m := a.Dot(c)
	a2 := 1 - m*m*inv
	b1 := 2 * (b0.Dot(a) - b0.Dot(c)*m*inv)
	c0 := b0.Dot(b0) - b0.Dot(c)*b0.Dot(c)*inv
	return s.pickCrossingRoot(u, apex, offVec, a2, b1, c0)
}

// coneTangencyDegeneracyTol classifies the doubly-degenerate tangency azimuth: both quadratic
// coefficients vanish RELATIVE to their natural scales (a2 against 1 — it is dimensionless on
// unit vectors — and b1 against the model-sized |c0|/r). The certificates in the station gate
// (fillet_elliptic_cone_canal.go) re-verify every station against both hosts, so a borderline
// classification here can only produce an honest decline, never a wrong band.
const coneTangencyDegeneracyTol = 1e-7

// pickCrossingRoot resolves the quadratic's physical root: nearest the rim, on the forward nappe.
// The tangency azimuth (a2 ≈ 0 AND b1 ≈ 0: every v satisfies the touching condition) returns the
// pinch limit vRim.
func (s ellipticConeRimSpine) pickCrossingRoot(u float64, apex math.Point3, offVec math.Vector3, a2, b1, c0 float64) (float64, bool) {
	vRim := s.vRimAt(u)
	scale := stdmath.Abs(c0)/s.r + s.r
	if stdmath.Abs(a2) <= coneTangencyDegeneracyTol {
		if stdmath.Abs(b1) <= coneTangencyDegeneracyTol*scale {
			return vRim, true // tangency azimuth: the offset ruling lies on the offset cone — pinch limit
		}
		return s.admitRoot(u, apex, offVec, -c0/b1)
	}
	disc := b1*b1 - 4*a2*c0
	if disc < 0 {
		return 0, false
	}
	q := -0.5 * (b1 + stdmath.Copysign(stdmath.Sqrt(disc), b1))
	best, ok := 0.0, false
	for _, v := range []float64{q / a2, safeDiv(c0, q)} {
		if cand, admitted := s.admitRoot(u, apex, offVec, v); admitted &&
			(!ok || stdmath.Abs(cand-vRim) < stdmath.Abs(best-vRim)) {
			best, ok = cand, true
		}
	}
	return best, ok
}

// safeDiv is c0/q with a NaN guard for the q=0 double-root case (the caller filters NaN through
// admitRoot's forward-nappe check, which rejects non-finite candidates).
func safeDiv(c0, q float64) float64 {
	if q == 0 {
		return stdmath.NaN()
	}
	return c0 / q
}

// admitRoot keeps a candidate v only when the resulting centre sits on the offset cone's FORWARD
// nappe (axial coordinate s > 0 from the offset apex) — the mirrored nappe root the squaring
// introduced must never be lofted.
func (s ellipticConeRimSpine) admitRoot(u float64, apex math.Point3, offVec math.Vector3, v float64) (float64, bool) {
	if stdmath.IsNaN(v) || stdmath.IsInf(v, 0) {
		return 0, false
	}
	center := s.ec.PointAt(u, v).TranslateBy(offVec)
	if apex.VectorTo(center).Dot(s.cone.AxisDir.AsVector()) <= 0 {
		return 0, false
	}
	return v, true
}

// coneFootOf drops the ball centre perpendicularly onto the cone: the centre is on the offset
// cone by construction, so the foot is exactly C − side·σc·r·n̂cone at C's azimuth. ok=false when
// the centre sits on the cone axis (no azimuth).
func (s ellipticConeRimSpine) coneFootOf(center math.Point3) (math.Point3, bool) {
	c := s.cone.AxisDir.AsVector()
	d := s.cone.Apex.VectorTo(center)
	rad, err := math.UnitVector3FromVector(d.Sub(c.Scale(d.Dot(c))))
	if err != nil {
		return math.Point3{}, false
	}
	cosH, sinH := stdmath.Cos(s.cone.HalfAngle), stdmath.Sin(s.cone.HalfAngle)
	nCone := rad.AsVector().Scale(cosH).Sub(c.Scale(sinH))
	return center.TranslateBy(nCone.Scale(-s.side * s.sigC * s.r)), true
}

// stationCertificateError is the do-no-harm certificate at one station: the larger of the two
// true tangency defects |dist(C, host) − r|. The wall distance reads the generic point inversion
// (as the plane-cap spine does); the cone distance is the exact slant-perpendicular form.
func (s ellipticConeRimSpine) stationCertificateError(st ellipticConeStation) float64 {
	_, _, foot := geom.ClosestPointOnSurface(s.ec, st.center)
	wallErr := stdmath.Abs(float64(foot.DistanceTo(st.center)) - s.r)
	return stdmath.Max(wallErr, stdmath.Abs(s.coneDistance(st.center)-s.r))
}

// coneDistance is the exact unsigned distance from p to the cone surface (slant-perpendicular:
// |ρ·cosα − s·sinα| in apex coordinates).
func (s ellipticConeRimSpine) coneDistance(p math.Point3) float64 {
	c := s.cone.AxisDir.AsVector()
	d := s.cone.Apex.VectorTo(p)
	ax := d.Dot(c)
	rho := float64(d.Sub(c.Scale(ax)).Length())
	return stdmath.Abs(rho*stdmath.Cos(s.cone.HalfAngle) - ax*stdmath.Sin(s.cone.HalfAngle))
}

// sectionHalfAngle is the half-angle of the station's cross-section arc — the pinch detector's
// objective (zero exactly at the host-tangency azimuth).
func (s ellipticConeRimSpine) sectionHalfAngle(st ellipticConeStation) float64 {
	da := st.center.VectorTo(st.wallFoot)
	db := st.center.VectorTo(st.coneFoot)
	den := float64(da.Length()) * float64(db.Length())
	if den == 0 {
		return 0
	}
	return 0.5 * stdmath.Acos(stdmath.Max(-1, stdmath.Min(1, da.Dot(db)/den)))
}
