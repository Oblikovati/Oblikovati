// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-ruling canal arm FAR-RUNOUT CAP — the cone-host trihedral-corner campaign, Slice CN4b-1
// (cone-host-corner-derivation.md §4 "Far-runout NEW cases"). The exact canal arm (CN2, a
// geom.BSplineSurface carrying an armCanalSpine) cannot close its far boundary loop without its cap:
// the arm's 4th boundary is canal ∩ cap-plane, and the far-runout engine's dispatchers (armSprings,
// intersectArmCapping, springsForHosts) only knew torus/cylinder arms. This file adds the CANAL cases
// they route to when ef.armCanalSpine != nil — the two exact host-contact SPRINGS (the feet ride them),
// their closed-form cap crossings (springCapFoot), and the far-cap TRIM. Two regimes by the cap's pose
// to the cone axis:
//   - cap ⊥ axis (C2/C6/C8, oblique to the hyperbola spine): the trim is the swept locus of the
//     per-station characteristic-circle ∩ cap crossings, traced by the MONOTONE arc angle ψ (x_f is
//     NON-monotone along it — it turns around near ψ=π/2, geometry-math CN4b-1 review), a polyline;
//   - the D1 SNOUT: the ruling outlives the ball, the spine ends at its hyperbola VERTEX (x_f=0), and the
//     far RADIAL cap plane CONTAINS the terminal characteristic circle, so the trim is that single
//     crossSectionArc — guarded ⊥ the vertex spine tangent (true only for the 90° wedge).
// It greens NO corpus case (the corner weld is CN4b-2); the cap is exercised through the far-runout
// engine on the REAL imported ruling arms.

const (
	// canalCapTrimSamples is the segment count of the ⊥-axis cap trim polyline. The trim is swept by the
	// monotone arc angle ψ; 32 chords resolve its shallow curvature across the (short) far band well inside
	// the fold-free mesh's needs. Endpoints are anchored to the exact feet, not sampled.
	canalCapTrimSamples = 32
	// canalCapSolveIters is the bisection budget for the per-ψ station solve f(x_f)=axial−h_cap=0 (f is
	// single-rooted on the bracket); 60 halvings reach machine precision on any model scale.
	canalCapSolveIters = 60
)

// coneCanalSpring is one host-contact SPRING of a cone-ruling canal arm as a geom.Curve3: the exact
// plane-foot locus (onCone=false, a conic in the radial plane) or the cone-foot locus (onCone=true, a
// curve on the cone) over the arm's x_f span [lo,hi]. Consumed ONLY by springCapFoot's canal case, which
// crosses it with the cap plane in closed form; it is never tessellated. TangentAt is a clamped central
// difference for interface completeness (the foot solve never differentiates the spring).
type coneCanalSpring struct {
	spine  coneCanalSpine
	lo, hi float64
	onCone bool
}

// xfAt maps the curve parameter t∈[0,1] to the station x_f∈[lo,hi].
func (s coneCanalSpring) xfAt(t float64) float64 { return s.lo + t*(s.hi-s.lo) }

// foot is the exact host-contact point of the ball at station x_f: the plane foot m+r·n̂ or the cone
// meridian foot T. On the spine ρ≥r>0, so coneFoot never declines.
func (s coneCanalSpring) foot(xf float64) math.Point3 {
	m := s.spine.center(xf)
	if s.onCone {
		f, _ := s.spine.coneFoot(m)
		return f
	}
	return s.spine.planeFoot(m)
}

// PointAt returns the spring foot at parameter t.
func (s coneCanalSpring) PointAt(t float64) math.Point3 { return s.foot(s.xfAt(t)) }

// TangentAt is a clamped central difference dP/dt (interface completeness only).
func (s coneCanalSpring) TangentAt(t float64) math.Vector3 {
	const h = 1e-5
	a, b := math.Clamp(t-h, 0, 1), math.Clamp(t+h, 0, 1)
	if b == a {
		return math.Vector3{}
	}
	return s.PointAt(a).VectorTo(s.PointAt(b)).Scale(1 / (b - a))
}

// Domain returns [0, 1].
func (s coneCanalSpring) Domain() (lo, hi float64) { return 0, 1 }

// canalArmSprings returns the cone-ruling arm's two host-contact springs ordered [plane, cone], over the
// picked ruling's loose x_f span (endpoint stations, sign on the ruling side). springsForHosts re-orders
// them onto (ef.a, ef.b) by which host is the geom.Plane vs geom.Cone.
func canalArmSprings(sp coneCanalSpine, e *topo.Edge, r float64) ([2]geom.Curve3, bool) {
	if e == nil {
		return [2]geom.Curve3{}, false
	}
	lo, hi := sp.xfSpanLoose(e)
	planeS := coneCanalSpring{spine: sp, lo: lo, hi: hi, onCone: false}
	coneS := coneCanalSpring{spine: sp, lo: lo, hi: hi, onCone: true}
	_ = r
	return [2]geom.Curve3{planeS, coneS}, true
}

// xfSpanLoose is the ruling's x_f span from its endpoint stations (signed by the ruling's ê side), the
// guard-free sibling of edgeXfSpan for the spring domain (springCapFoot solves the crossing in closed
// form and never samples the span, so the collapse/no-fit guards are edgeXfSpan's job, not the spring's).
func (s coneCanalSpine) xfSpanLoose(e *topo.Edge) (lo, hi float64) {
	sgn := 1.0
	if float64(s.apex.VectorTo(edgeMidpoint(e)).Dot(s.ePerp)) < 0 {
		sgn = -1
	}
	x0, _ := s.xfAtEndpoint(e.StartVertex().Point())
	x1, _ := s.xfAtEndpoint(e.EndVertex().Point())
	return sortedSpan(sgn*x0, sgn*x1)
}

// canalCapFoot crosses a canal spring with the cap plane and returns the crossing nearer the far vertex
// `near` (the spring's own side of the axis). The cap is classified by its pose to the cone axis: ⊥ axis
// (C2/C6/C8, the axial closed form) or a radial plane ∥ the axis THROUGH the axis (the D1 snout, x_f=0);
// a general oblique cap is out of the cone corpus and declines with the offending pose. Called from
// springCapFoot (which drops the reason) and springCapFootReasoned (which surfaces it).
func (s coneCanalSpring) canalCapFoot(capping geom.Surface, near math.Point3, res Resolution) (math.Point3, bool, string) {
	pl, ok := capping.(geom.Plane)
	if !ok {
		return math.Point3{}, false, fmt.Sprintf("canal cap foot: capping is %T, want geom.Plane", capping)
	}
	axisDot := stdmath.Abs(float64(pl.Normal().Dot(s.spine.axis)))
	switch {
	case axisDot >= 1-sinFloor:
		return s.capFootPerpAxis(pl, near)
	case axisDot <= sinFloor:
		return s.capFootRadial(pl, res)
	}
	return math.Point3{}, false, fmt.Sprintf(
		"canal cap foot: cap plane obliquely posed to the cone axis (|n̂·â|=%g, need ≥%g [⊥ axis] or ≤%g [radial through axis]) — out of the cone corpus",
		axisDot, 1-sinFloor, sinFloor)
}

// capFootPerpAxis crosses the spring with a cap plane ⊥ the axis at apex-height h_cap: the ball-centre
// axis distance ρ closing on h_cap is closed form (coneCapRho), and x_f=±√(ρ²−r²), the sign the ruling's
// side (nearerFoot). ρ<r ⇒ the cap sits below the hyperbola vertex; decline with the offending values.
func (s coneCanalSpring) capFootPerpAxis(pl geom.Plane, near math.Point3) (math.Point3, bool, string) {
	hCap := float64(s.spine.apex.VectorTo(pl.Origin).Dot(s.spine.axis))
	rho := coneCapRho(s.spine, hCap, s.onCone)
	if rho < s.spine.radius {
		return math.Point3{}, false, fmt.Sprintf(
			"canal cap foot: rolling ball never reaches the ⊥-axis cap on the %s spring (ρ=%g < r=%g at cap height h_cap=%g, cone half-angle α=%g rad)",
			s.springName(), rho, s.spine.radius, hCap, s.spine.halfAngle())
	}
	xf := stdmath.Sqrt(rho*rho - s.spine.radius*s.spine.radius)
	return s.nearerFoot(xf, near), true, ""
}

// coneCapRho is the ball-centre axis distance ρ at which a canal spring's foot reaches apex-height h_cap:
// plane spring ρ=(h_cap−r/sinα)·tanα, cone spring ρ=h_cap·tanα−r·cosα (cone-host-corner-derivation §4).
// Raw (ungated) so callers can both test ρ<r AND report the offending ρ. Shared by capFootPerpAxis and
// coneFootStation (was duplicated).
func coneCapRho(sp coneCanalSpine, hCap float64, onCone bool) float64 {
	if onCone {
		return hCap*sp.tanA - sp.radius*sp.cosA
	}
	return (hCap - sp.radius/sp.sinA) * sp.tanA
}

// springName is "plane"/"cone" — the host the spring rides, for decline diagnostics.
func (s coneCanalSpring) springName() string {
	if s.onCone {
		return "cone"
	}
	return "plane"
}

// nearerFoot evaluates the spring foot at +x_f and −x_f (the two mirror crossings across the axis plane)
// and keeps whichever lands nearer the far vertex — the one on the picked ruling's side.
func (s coneCanalSpring) nearerFoot(xf float64, near math.Point3) math.Point3 {
	p, q := s.foot(xf), s.foot(-xf)
	if p.DistanceTo(near) <= q.DistanceTo(near) {
		return p
	}
	return q
}

// capFootRadial crosses the spring with a radial cap plane (∥ the axis) that passes THROUGH the axis —
// the D1 snout, where both feet sit at the spine vertex x_f=0. A radial cap NOT through the axis, or one
// whose normal is ⊥ the ruling's transverse ê, is out of scope and declines with the offending value.
func (s coneCanalSpring) capFootRadial(pl geom.Plane, res Resolution) (math.Point3, bool, string) {
	sp := s.spine
	n := pl.Normal()
	e0 := float64(sp.apex.VectorTo(pl.Origin).Dot(n)) // (o_cap − A)·n̂_cap
	q := float64(sp.ePerp.Dot(n))
	if tol := res.Weld() * sp.radius; stdmath.Abs(e0) > tol {
		return math.Point3{}, false, fmt.Sprintf(
			"canal cap foot: radial cap misses the cone axis (incidence |(o−A)·n̂|=%g > tol=%g) — not the D1 snout", stdmath.Abs(e0), tol)
	}
	if stdmath.Abs(q) < sinFloor {
		return math.Point3{}, false, fmt.Sprintf(
			"canal cap foot: radial cap normal ⊥ the ruling transverse ê (|ê·n̂|=%g < %g) — no crossing", stdmath.Abs(q), sinFloor)
	}
	xf := e0 / q
	if s.onCone {
		xf = sp.radius * float64(sp.nOut.Dot(n)) / q
	}
	return s.foot(xf), true, ""
}

// halfAngle is the cone's half-angle α (radians) — for decline diagnostics.
func (s coneCanalSpine) halfAngle() float64 { return stdmath.Atan2(s.sinA, s.cosA) }

// canalCappingTrim is the far-runout cap trim of a cone-ruling canal arm — arm ∩ cap-plane oriented
// feet[0]→feet[1]. It classifies the cap by its pose to the axis and dispatches the ⊥-axis swept trim or
// the D1 snout arc; a general oblique cap declines with the offending pose. Called from
// intersectArmCapping's canal case (which drops the reason) and capTrimDeclineReason (which surfaces it).
func canalCappingTrim(sp coneCanalSpine, capping geom.Surface, feet [2]math.Point3, r float64, res Resolution) (geom.Curve3, bool, string) {
	_ = r
	pl, ok := capping.(geom.Plane)
	if !ok {
		return nil, false, fmt.Sprintf("canal cap trim: capping is %T, want geom.Plane", capping)
	}
	axisDot := stdmath.Abs(float64(pl.Normal().Dot(sp.axis)))
	switch {
	case axisDot >= 1-sinFloor:
		return canalPerpAxisTrim(sp, pl, feet, res)
	case axisDot <= sinFloor:
		return canalSnoutTrim(sp, pl, feet, res)
	}
	return nil, false, fmt.Sprintf(
		"canal cap trim: cap plane obliquely posed to the cone axis (|n̂·â|=%g, need ≥%g [⊥ axis] or ≤%g [radial through axis])",
		axisDot, 1-sinFloor, sinFloor)
}

// canalSnoutTrim is the D1 snout cap: the terminal characteristic arc at the hyperbola vertex (x_f=0),
// which lies IN the radial cap plane. Two-condition guard (do-no-harm): (a) the cap must be ⊥ the vertex
// spine tangent m'(0)=ê (|n̂·ê|≈1, true only for the 90° wedge) AND (b) contain the vertex circle centre;
// otherwise it is not a snout and declines naming the failing condition + its offending value.
func canalSnoutTrim(sp coneCanalSpine, pl geom.Plane, feet [2]math.Point3, res Resolution) (geom.Curve3, bool, string) {
	m := sp.center(0)
	tol := res.Weld() * sp.radius
	if reason := snoutCapGuardReason(sp, pl, m, tol); reason != "" {
		return nil, false, reason
	}
	cT, ok := sp.coneFoot(m)
	if !ok {
		return nil, false, fmt.Sprintf("canal snout: cone foot at the vertex centre %v is degenerate (on the axis)", m)
	}
	if !feetMatchArcEnds(sp.planeFoot(m), cT, feet, tol) {
		return nil, false, fmt.Sprintf("canal snout: supplied feet %v/%v are not the vertex arc ends within tol=%g", feet[0], feet[1], tol)
	}
	arc, ok := orientedCharArc(m, feet, sp.radius, sp.ePerp)
	if !ok {
		return nil, false, "canal snout: characteristic arc build declined (collinear feet + centre)"
	}
	return arc, true, ""
}

// snoutCapGuardReason returns the two-condition snout guard's decline reason (naming the failing
// condition + its offending value), or "" when both hold: (a) the cap is ⊥ the vertex spine tangent ê
// (the 90° wedge), and (b) the cap contains the vertex circle centre m.
func snoutCapGuardReason(sp coneCanalSpine, pl geom.Plane, m math.Point3, tol float64) string {
	n := pl.Normal()
	if align := stdmath.Abs(float64(n.Dot(sp.ePerp))); 1-align > sinFloor {
		return fmt.Sprintf(
			"canal snout: cond (a) far radial plane not ⊥ the vertex spine tangent ê (tilt %g rad, |n̂·ê|=%g < %g; a non-90° wedge)",
			stdmath.Acos(math.Clamp(align, -1, 1)), align, 1-sinFloor)
	}
	if inc := stdmath.Abs(float64(pl.Origin.VectorTo(m).Dot(n))); inc > tol {
		return fmt.Sprintf(
			"canal snout: cond (b) far radial plane does not contain the vertex circle centre %v (incidence %g > tol=%g)", m, inc, tol)
	}
	return ""
}

// feetMatchArcEnds certifies the supplied feet ARE the vertex arc's two ends (plane foot, cone foot) in
// some order, within tol — the shared-edge identity the engine asserts, checked before emitting the arc.
func feetMatchArcEnds(fP, cT math.Point3, feet [2]math.Point3, tol float64) bool {
	d0 := float64(feet[0].DistanceTo(fP)) <= tol && float64(feet[1].DistanceTo(cT)) <= tol
	d1 := float64(feet[0].DistanceTo(cT)) <= tol && float64(feet[1].DistanceTo(fP)) <= tol
	return d0 || d1
}

// orientedCharArc builds the exact radius-r characteristic arc through the two feet about centre m in the
// plane with normal `normal`, oriented so PointAt(0)=feet[0]: the ≤π signed sweep from (feet[0]−m) to
// (feet[1]−m) is the minor (cavity) arc — the fillet band's cross-section.
func orientedCharArc(m math.Point3, feet [2]math.Point3, r float64, normal math.Vector3) (geom.Curve3, bool) {
	refU, err := math.UnitVector3FromVector(m.VectorTo(feet[0]))
	if err != nil {
		return nil, false
	}
	bin := normal.Cross(refU.AsVector())
	d := m.VectorTo(feet[1])
	sweep := stdmath.Atan2(float64(d.Dot(bin)), float64(d.Dot(refU.AsVector())))
	arc, err := geom.NewArc3d(m, normal, refU.AsVector(), r, 0, sweep)
	if err != nil {
		return nil, false
	}
	return arc, true
}

// canalPerpAxisTrim is the ⊥-axis cap trim: the swept locus of characteristic-circle ∩ cap crossings,
// traced by the MONOTONE arc angle ψ from the plane foot (ψ=0) to the cone foot (ψ=ψ_c). x_f is
// non-monotone along the trim (it turns around near ψ=π/2), so ψ — not x_f — is the sweep parameter
// (geometry-math CN4b-1 review). Delivered as a polyline with the endpoints anchored to the exact feet.
func canalPerpAxisTrim(sp coneCanalSpine, pl geom.Plane, feet [2]math.Point3, res Resolution) (geom.Curve3, bool, string) {
	iP, ok := planeFootIndex(sp, feet, res)
	if !ok {
		return nil, false, fmt.Sprintf("canal cap trim: neither foot %v/%v is uniquely on the radial plane (cannot order plane vs cone)", feet[0], feet[1])
	}
	hCap := float64(sp.apex.VectorTo(pl.Origin).Dot(sp.axis))
	xfP := float64(sp.apex.VectorTo(feet[iP]).Dot(sp.ePerp))
	xfT, ok := coneFootStation(sp, feet[1-iP], hCap)
	if !ok {
		return nil, false, fmt.Sprintf("canal cap trim: cone foot below the hyperbola vertex at cap height h_cap=%g (ρ=%g < r=%g)", hCap, coneCapRho(sp, hCap, true), sp.radius)
	}
	pts, ok := traceCapTrim(sp, hCap, xfP, xfT)
	if !ok {
		return nil, false, fmt.Sprintf("canal cap trim: ψ-sweep station solve found no crossing between x_f stations %g→%g (h_cap=%g)", xfP, xfT, hCap)
	}
	poly, ok := anchoredTrimPolyline(pts, feet, iP)
	if !ok {
		return nil, false, "canal cap trim: swept polyline needs ≥2 vertices"
	}
	return poly, true, ""
}

// planeFootIndex is the index (0/1) of the foot on the radial plane ((foot−A)·n̂ ≈ 0), the plane host's
// foot; the other is the cone foot. Declines when the two feet are not one-of-each (an upstream mishap).
func planeFootIndex(sp coneCanalSpine, feet [2]math.Point3, res Resolution) (int, bool) {
	tol := res.Weld() * sp.radius
	d0 := stdmath.Abs(float64(sp.apex.VectorTo(feet[0]).Dot(sp.nOut)))
	d1 := stdmath.Abs(float64(sp.apex.VectorTo(feet[1]).Dot(sp.nOut)))
	if d0 <= tol && d1 > tol {
		return 0, true
	}
	if d1 <= tol && d0 > tol {
		return 1, true
	}
	return 0, false
}

// coneFootStation recovers the cone foot's hyperbola station x_f from the cap height (ρ=h_cap·tanα−r·cosα,
// x_f=±√(ρ²−r²)), the sign taken from the foot's ê side. Declines when ρ<r (the cone foot would be the
// snout vertex — a ⊥-axis cap does not reach it).
func coneFootStation(sp coneCanalSpine, cT math.Point3, hCap float64) (float64, bool) {
	rho := coneCapRho(sp, hCap, true)
	if rho < sp.radius {
		return 0, false
	}
	xf := stdmath.Sqrt(rho*rho - sp.radius*sp.radius)
	if float64(sp.apex.VectorTo(cT).Dot(sp.ePerp)) < 0 {
		xf = -xf
	}
	return xf, true
}

// traceCapTrim sweeps the arc angle ψ from 0 (plane foot) to ψ_c (cone foot) and root-finds the station
// x_f for each, returning the cap crossing points plane-foot→cone-foot. The bracket runs from the plane
// foot station (where the axial residual is ≤0) out past the cone foot station (where it is >0), spanning
// the x_f turnaround; the per-ψ residual is single-rooted there. Declines on a missing bracket/angle.
func traceCapTrim(sp coneCanalSpine, hCap, xfP, xfT float64) ([]math.Point3, bool) {
	psiEnd, ok := coneFootAngle(sp, xfT)
	if !ok || psiEnd <= 0 {
		return nil, false
	}
	lo, hi := xfP, xfT+3*sp.radius*signOf(xfT)
	pts := make([]math.Point3, canalCapTrimSamples+1)
	for j := range pts {
		psi := psiEnd * float64(j) / float64(canalCapTrimSamples)
		xf, ok := solveStationAtAngle(sp, hCap, psi, lo, hi)
		if !ok {
			return nil, false
		}
		pts[j] = sp.capPointAt(xf, psi)
	}
	return pts, true
}

// coneFootAngle is the cone foot's characteristic-circle angle ψ_c at station x_f (the trim's far end):
// atan2 of (coneFoot−m) in the {n̂, charBinormal} frame. It is obtuse (cosψ_c=−r·cosα/ρ<0) by
// construction, so the trim sweep passes ψ=π/2.
func coneFootAngle(sp coneCanalSpine, xf float64) (float64, bool) {
	m := sp.center(xf)
	cT, ok := sp.coneFoot(m)
	if !ok {
		return 0, false
	}
	d := m.VectorTo(cT)
	return stdmath.Atan2(float64(d.Dot(sp.charBinormal(xf))), float64(d.Dot(sp.nOut))), true
}

// charBinormal is the in-plane basis vector v̂ = m̂'(x_f) × n̂ of the characteristic circle at station x_f
// (m'(x_f)=ζ'·â+ê, ζ'=x_f/(ρ·tanα)), completing {n̂, v̂}. +v̂ points toward the cone foot (ψ_c>0).
func (s coneCanalSpine) charBinormal(xf float64) math.Vector3 {
	zp := xf / (s.rhoAt(xf) * s.tanA)
	mt, err := math.UnitVector3FromVector(s.axis.Scale(zp).Add(s.ePerp))
	if err != nil {
		return s.ePerp
	}
	v, err := math.UnitVector3FromVector(mt.AsVector().Cross(s.nOut))
	if err != nil {
		return s.ePerp
	}
	return v.AsVector()
}

// capPointAt is the characteristic-circle point Q(x_f, ψ) = m + r·(cosψ·n̂ + sinψ·v̂) — the surface point
// at station x_f and arc angle ψ (ψ=0 is the plane foot n̂ direction).
func (s coneCanalSpine) capPointAt(xf, psi float64) math.Point3 {
	m := s.center(xf)
	cos, sin := stdmath.Cos(psi), stdmath.Sin(psi)
	off := s.nOut.Scale(s.radius * cos).Add(s.charBinormal(xf).Scale(s.radius * sin))
	return m.TranslateBy(off)
}

// solveStationAtAngle root-finds the station x_f where the characteristic point at fixed angle ψ hits the
// cap: f(x_f)=ζ(x_f) − r·sinψ/√(1+ζ'²) − h_cap = 0. f is single-rooted on [lo,hi] (f(lo)≤0<f(hi)); a
// sign-based bisection converges regardless of the ruling's ê side. Declines when the bracket has no root.
func solveStationAtAngle(sp coneCanalSpine, hCap, psi, lo, hi float64) (float64, bool) {
	f := func(xf float64) float64 {
		zp := xf / (sp.rhoAt(xf) * sp.tanA)
		return sp.zetaAt(xf) - sp.radius*stdmath.Sin(psi)/stdmath.Sqrt(1+zp*zp) - hCap
	}
	flo, fhi := f(lo), f(hi)
	if flo > 0 || fhi <= 0 {
		return 0, false
	}
	for i := 0; i < canalCapSolveIters; i++ {
		mid := 0.5 * (lo + hi)
		if (f(mid) <= 0) == (flo <= 0) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return 0.5 * (lo + hi), true
}

// anchoredTrimPolyline orients the plane→cone trace onto feet[0]→feet[1] (reversing when feet[0] is the
// cone foot) and pins both endpoints to the exact feet, then wraps it as a geom.Polyline.
func anchoredTrimPolyline(pts []math.Point3, feet [2]math.Point3, planeIdx int) (geom.Curve3, bool) {
	if planeIdx == 1 {
		pts = reversePoints(pts)
	}
	pts[0], pts[len(pts)-1] = feet[0], feet[1]
	poly, err := geom.NewPolyline(pts)
	if err != nil {
		return nil, false
	}
	return poly, true
}

// springCapFootReasoned crosses a spring with the capping and, on decline, carries the CANAL spring's
// offending values (ρ<r, oblique/radial cap pose) — the CLAUDE.md "offending value" rule. Non-canal
// (torus/cylinder) springs route through the unchanged springCapFoot with no reason, so armRunoutFeet's
// torus/cylinder decline string is byte-identical. Used by armRunoutFeet in place of the bare springCapFoot.
func springCapFootReasoned(spring geom.Curve3, capping geom.Surface, near math.Point3, res Resolution) (math.Point3, bool, string) {
	if canal, ok := spring.(coneCanalSpring); ok {
		return canal.canalCapFoot(capping, near, res)
	}
	p, ok := springCapFoot(spring, capping, near, res)
	return p, ok, ""
}

// capTrimDeclineReason is the trim decline message obliqueRunout surfaces: for a CANAL arm it re-derives
// canalCappingTrim's offending-value reason (decline path only); every other arm keeps the byte-identical
// generic string. It is called only after intersectArmCapping declined, so it never alters a good trim.
func capTrimDeclineReason(ef edgeFillet, capping geom.Surface, feet [2]math.Point3, r float64, res Resolution) string {
	if ef.armCanalSpine != nil {
		if _, _, reason := canalCappingTrim(*ef.armCanalSpine, capping, feet, r, res); reason != "" {
			return reason
		}
	}
	return fmt.Sprintf("oblique runout: intersectArmCapping declined the trim through feet %v→%v", feet[0], feet[1])
}

// firstReason returns the first non-empty reason, else the fallback — so a canal spring's specific decline
// wins over the generic one, while a torus/cylinder decline (both empty) keeps the byte-identical fallback.
func firstReason(a, b, fallback string) string {
	if a != "" {
		return a
	}
	if b != "" {
		return b
	}
	return fallback
}
