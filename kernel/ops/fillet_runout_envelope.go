// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// runoutEnvelope is the exact rolling-ball model of a SETBACK-CLOSE run-out (runout-envelope-report.md
// "Recommendation & derivation"). A constant-radius fillet on a STRAIGHT edge is computed section by
// section on the pencil of planes Π(s) normal to the spine; where a boss blocks the plain fillet the
// ball no longer touches both hosts and the run-out splits into a PARTITION of three band types:
//
//	plain    ball tangent to both hosts                        → the fillet cylinder
//	surf-rst ball tangent to host B, THROUGH the A footprint   → surfRstCentre
//	rst-rst  ball THROUGH both footprints                      → rstRstCentre
//
// Each band's centre is available in CLOSED FORM (no marching, no convergence story) because the
// section plane reduces every constraint to a line/circle intersection inside Π(s). This is the model
// a Coons fill through the same four rails does NOT reproduce: the rails are right and the interior is
// not the envelope, which is the defect coons4-audit.md measured at 9–19% of r on nine corpus greens.
//
// Grounded in DRAWEXE 8.0.0 on the corpus's own inputs: the predicted centre reproduces OCCT's own
// contact rail to 1e-11 on S4 and exactly (−2√2, −5√3) on S6/S9 — see the report's receipts table.
type runoutEnvelope struct {
	cyl    geom.Cylinder // the plain fillet: its axis IS the plain ball-centre line A(s)
	radius float64       // rolling-ball radius r
	spine  math.Vector3  // unit spine direction e (= cyl.AxisDir)
}

// newRunoutEnvelope frames a straight constant-radius fillet for the closed-form station solves.
//
// The straight-spine assumption every solve here rests on is enforced STRUCTURALLY upstream, so no
// runtime guard is warranted: the only two callers are resolveSetbackTiling and resolveSingleBossTiling,
// and both reach it only past setbackHostPlanes / singleBossHostPlanes, which require BOTH hosts to be
// a geom.Plane. Two planes meet in a straight line, so cyl's axis is straight by construction.
func newRunoutEnvelope(cyl geom.Cylinder) runoutEnvelope {
	return runoutEnvelope{cyl: cyl, radius: cyl.Radius, spine: cyl.AxisDir.AsVector()}
}

// axisPoint is A(s), the PLAIN fillet's ball centre at spine station s — the point every band's
// closed form is written relative to (it lies on both host offset planes, so it is on L_B(s) for
// free).
func (f runoutEnvelope) axisPoint(s float64) math.Point3 {
	return f.cyl.Origin.TranslateBy(f.spine.Scale(math.Scalar(s)))
}

// hostNormal is the CENTRE-WARD unit normal of a host plane the fillet is tangent to, read off the
// geometry itself ((A(s) − foot)/r) rather than from the plane's own orientation — so no material-side
// bookkeeping can flip it. ok=false when the plane is not at ball distance from the axis (not a host
// of this fillet), which is a caller construction bug and declines (do-no-harm).
func (f runoutEnvelope) hostNormal(host geom.Plane, s, weld float64) (math.Vector3, bool) {
	a := f.axisPoint(s)
	v := projectOntoPlane(a, host).VectorTo(a)
	l := float64(v.Length())
	if stdmath.Abs(l-f.radius) > weld || l == 0 {
		return math.Vector3{}, false
	}
	return v.Scale(math.Scalar(1 / l)), true
}

// surfRstCentre solves the SURF-RST band: the ball is tangent to `tangent` and passes THROUGH the
// point q on the restriction curve (the boss footprint lying in `restrict`). Derivation:
//
//	c ∈ L_B(s) = { A(s) + t·d },  d = n_B × e          (L_B = Π(s) ∩ the r-offset of `tangent`)
//	|c − q| = r  ⇒  t = −(w·d) ± √(r² − |w_⊥|²),  w = A(s) − q,  |w_⊥|² = |w|² − (w·d)²
//
// The BRANCH is a constant, not a continuation: at the cut station q is the plain contact on
// `restrict`, so w = r·n_A, t = 0 is a root and the sign resolves to σ = sign(n_A·d) — frame-only,
// hence valid for the whole band (it can only flip where the discriminant vanishes, which is guarded).
// |w_⊥|² is formed as |w|² − (w·d)² in ONE subtraction (report pitfall 10: never re-sum projected
// components). ok=false — never a clamped discriminant, which would fabricate a tangency — when
// the hosts are near-parallel (σ undefined) or q is farther than r from L_B (no ball exists).
func (f runoutEnvelope) surfRstCentre(tangent, restrict geom.Plane, s float64, q math.Point3, weld float64) (math.Point3, bool) {
	nb, ok0 := f.hostNormal(tangent, s, weld)
	na, ok1 := f.hostNormal(restrict, s, weld)
	if !ok0 || !ok1 {
		return math.Point3{}, false
	}
	d := nb.Cross(f.spine)
	sigma := float64(na.Dot(d))
	if stdmath.Abs(sigma) < stdmath.Sin(tessellate.SeamAngularTol) {
		return math.Point3{}, false // hosts near-parallel: the branch sign is undefined (pitfall 5)
	}
	a := f.axisPoint(s)
	w := q.VectorTo(a)
	wd := float64(w.Dot(d))
	perpSq := float64(w.LengthSquared()) - wd*wd
	h := f.radius*f.radius - perpSq
	if h < 0 {
		return math.Point3{}, false // q farther than r from L_B: no surf-rst ball (pitfall 1)
	}
	t := -wd + stdmath.Copysign(stdmath.Sqrt(h), sigma)
	return a.TranslateBy(d.Scale(math.Scalar(t))), true
}

// rstRstCentre solves the RST-RST band: the ball passes through BOTH footprint points qa, qb in the
// section plane Π(s). Inside Π(s) this is the intersection of two radius-r circles:
//
//	M = ½(qa+qb),  δ = qb−qa,  m = (e×δ)/|e×δ|,  c = M + σ′·√(r² − |δ|²/4)·m
//
// with σ′ = sign((A(s)−M)·m): the centre sits on the SAME side of the chord as the plain fillet's
// centre. On the corpus that discriminates by a factor of 6 (S4 x=0: the kept root is 2.70 from A(s),
// the rejected one 16.3). ok=false when the footprints are farther than 2r apart, when they osculate
// (m ill-conditioned), or when A(s) lies on the chord (σ′ ambiguous) — all honest declines.
func (f runoutEnvelope) rstRstCentre(s float64, qa, qb math.Point3, weld float64) (math.Point3, bool) {
	mid := qa.Midpoint(qb)
	delta := qa.VectorTo(qb)
	cross := f.spine.Cross(delta)
	l := float64(cross.Length())
	if l < weld {
		return math.Point3{}, false // footprints osculate / δ ∥ spine: the chord normal is ill-posed
	}
	m := cross.Scale(math.Scalar(1 / l))
	half := 0.5 * float64(delta.Length())
	h := f.radius*f.radius - half*half
	if h < 0 {
		return math.Point3{}, false // |qa−qb| > 2r: no common radius-r ball (pitfall 2)
	}
	side := float64(mid.VectorTo(f.axisPoint(s)).Dot(m))
	if stdmath.Abs(side) < weld {
		return math.Point3{}, false // A(s) on the chord: the branch is ambiguous (pitfall 4)
	}
	return mid.TranslateBy(m.Scale(math.Scalar(stdmath.Copysign(stdmath.Sqrt(h), side)))), true
}

// runoutStation is one EXACT cross-section of a run-out canal: the ball centre and both contact
// points, each algebraically at distance r from the centre. It is precisely the triple
// geom.LoftCanalStations consumes (and self-asserts), so a mis-derived station is declined, not lofted.
type runoutStation struct {
	s      float64     // spine station
	centre math.Point3 // ball centre c(s)
	footA  math.Point3 // contact on the A side (a footprint point, or the plain contact at a cut)
	footB  math.Point3 // contact on the B side (the tangency foot, or the second footprint point)
}

// sectionArc is the station's exact rolling-ball cross-section: the MINOR radius-r arc footA→footB
// about centre. It is the same construction armSectionArc uses at a plain station (bisector midpoint
// on the circle), so a run-out band and the plain fillet share the identical curve where they meet.
// ok=false when the two feet and the centre are near-collinear (report pitfall 6/7).
func (st runoutStation) sectionArc(radius float64) (geom.Arc3d, bool) {
	da := st.centre.VectorTo(st.footA)
	db := st.centre.VectorTo(st.footB)
	la, lb := float64(da.Length()), float64(db.Length())
	if la == 0 || lb == 0 {
		return geom.Arc3d{}, false
	}
	bis := da.Scale(math.Scalar(1 / la)).Add(db.Scale(math.Scalar(1 / lb)))
	l := float64(bis.Length())
	if l < arcBisectorTiny {
		return geom.Arc3d{}, false // feet near-antipodal: the arc reaches a half turn (pitfall 6)
	}
	arc, err := geom.Arc3dByThreePoints(st.footA, st.centre.TranslateBy(bis.Scale(math.Scalar(radius/l))), st.footB)
	return arc, err == nil
}

// sectionArcFrom is sectionArc traced from the requested endpoint. The two bands sharing a seam
// station build their common rail through THIS one call (the tiling stores the arc once and hands the
// same object to both), so the shared seam is watertight by identity, not by agreement.
func (st runoutStation) sectionArcFrom(radius float64, from math.Point3, weld float64) (geom.Arc3d, bool) {
	arc, ok := st.sectionArc(radius)
	if !ok {
		return geom.Arc3d{}, false
	}
	if float64(curveStart(arc).DistanceTo(from)) <= weld {
		return arc, true
	}
	flipped := runoutStation{s: st.s, centre: st.centre, footA: st.footB, footB: st.footA}
	return flipped.sectionArc(radius)
}
