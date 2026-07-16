// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A (m5-curved-arm-derivation.md §D2/§D3/§D5): the exact analytic arm surfaces that
// classifyCurvedArm dispatches to. A Plane∧Cylinder rolling-ball fillet is a TORUS when the axis is
// ⊥ the plane (config i, circle edge) and a CYLINDER when the axis is ∥ the plane (config ii, line
// edge) — both closed-form primitives, no canal surface — plus the cross-section quarter-arc the
// corner engine consumes. Wired into nothing here (Task 3 does that); unit-pinned to the OCCT B3
// BREP oracle (`5 … 40 10` torus, `2 … 10` cylinder).

// armSpindleBand is k in the existence guard R−r < k·res.Weld() (§Numerical pitfalls): a length band,
// scaled to the model (ADR-0042), below which the convex torus tube reaches the axis and the surface
// self-intersects (spindle/horn torus). k=4 sits at the top of the derivation's k≈2..4 so we reject
// generously rather than emit a near-degenerate torus. Not a bare 1e-6.
const armSpindleBand = 4

// torusArmSurface builds the config-(i) exact torus arm (m5-curved-arm-derivation.md §D2): a
// rolling-ball fillet of radius r on the CIRCLE edge where cylinder cyl meets plane pl with the axis
// perpendicular to the plane. The tube minor radius is r; the major radius is R−r for the convex
// external rim (the corpus case — a protruding boss cap), and the centre is the cylinder axis
// projected onto the plane, offset r into the material (opposite the plane's outward normal). Built
// via NewTorusWithRef so the u=0 seam aligns with the boss wall (Oblikovati#129). Returns false when
// R−r collapses onto the axis (r ≥ R): a self-intersecting spindle torus is never emitted.
//
// Example: torusArmSurface(bossWall{R:50}, topCap{z:100}, 10, res) → torus centre (0,0,90), axis ẑ,
// major 40, minor 10 (the B3 top-rim arm, OCCT BREP `5 0 0 90 0 0 1 … 40 10`).
func torusArmSurface(cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) (geom.Torus, bool) {
	majorR := cyl.Radius - r
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return geom.Torus{}, false
	}
	inward := n.Negate().AsVector() // outward cap normal → into the material by r
	center := projectOntoPlane(cyl.Origin, pl).TranslateBy(inward.Scale(r))
	tor, err := geom.NewTorusWithRef(center, n.AsVector(), cyl.Ref.AsVector(), majorR, r)
	return tor, err == nil
}

// cylinderArmSurface builds the config-(ii) exact cylinder arm (m5-curved-arm-derivation.md §D3): a
// rolling-ball fillet of radius r on the LINE edge where cylinder cyl meets plane pl with the axis
// parallel to the plane. The fillet cylinder's axis is the ruling of P_r∩C_ρ (the offset plane meets
// the coaxial offset cylinder of radius ρ=R−r in a pair of rulings; edge selects the near one) and
// its radius is r. Returns false when P_r clears C_ρ (no real ruling — the plane misses the wall).
//
// Example: cylinderArmSurface(wallEdge, bossWall{R:50}, radialPlane, 10) → radius-10 cylinder about
// the wall ruling (the B3 vertical-wall arm, OCCT BREP `2 … 10`).
func cylinderArmSurface(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64) (geom.Cylinder, bool) {
	rho := cyl.Radius - r
	if rho <= 0 {
		return geom.Cylinder{}, false // convex spindle: the offset cylinder has collapsed
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return geom.Cylinder{}, false
	}
	base, ok := armRulingBase(edge, cyl, pl, n, rho, r)
	if !ok {
		return geom.Cylinder{}, false
	}
	arm, err := geom.NewCylinderWithRef(base, cyl.AxisDir.AsVector(), n.AsVector(), r)
	return arm, err == nil
}

// armRulingBase returns a point on the selected ruling of P_r∩C_ρ (m5-curved-arm-derivation.md §D3).
// In the axis frame a ruling centre w satisfies |w|=ρ and w·n̂ = −r − (A−p_P)·n̂ (the offset plane at
// signed distance −r into the material); the two solutions w = m·n̂ ± √(ρ²−m²)·(â×n̂) are the two
// rulings, and the edge midpoint picks the near one. Returns false when the radicand is non-positive
// (P_r grazes or clears C_ρ — no real intersection).
func armRulingBase(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, n math.UnitVector3, rho, r float64) (math.Point3, bool) {
	a := cyl.AxisDir.AsVector()
	b := a.Cross(n.AsVector()) // ⟂ both axis and normal (config ii: n̂ ⟂ â), unit length
	m := -r - pl.Origin.VectorTo(cyl.Origin).Dot(n.AsVector())
	disc := rho*rho - m*m
	if disc <= 0 {
		return math.Point3{}, false
	}
	t := stdmath.Sqrt(disc)
	off := n.AsVector().Scale(m)
	plus := cyl.Origin.TranslateBy(off.Add(b.Scale(t)))
	minus := cyl.Origin.TranslateBy(off.Sub(b.Scale(t)))
	return nearerRuling(edge, plus, minus), true
}

// nearerRuling returns whichever ruling base lies closer to the picked edge's midpoint — the physical
// disambiguation of the two P_r∩C_ρ rulings (the edge sits on exactly one of them).
func nearerRuling(edge *topo.Edge, plus, minus math.Point3) math.Point3 {
	lo, hi := edge.StartVertex().Point(), edge.EndVertex().Point()
	mid := lo.TranslateBy(lo.VectorTo(hi).Scale(0.5))
	if float64(mid.DistanceTo(plus)) <= float64(mid.DistanceTo(minus)) {
		return plus
	}
	return minus
}

// curvedArmSectionArc is the corner cross-section arc the corner-blend engine consumes as its
// arm↔sphere rail (m5-curved-arm-derivation.md §D5, m5-arm-section-derivation.md). It dispatches on
// the arm kind, because the two exact Plane∧Cylinder configs subtend DIFFERENT angles:
//   - the config-i TORUS arm (axis ⊥ plane) subtends a true π/2 quarter — in the axial section the
//     cylinder appears straight, so the ball does not roll within the section (advisor-certified, NOT
//     defective); left exactly as built.
//   - the config-ii CYLINDER arm (axis ∥ plane) subtends π/2 + asin(r/ρ) convex / π/2 − asin(r/ρ)
//     concave: the ball rolls onto the CURVED wall and its contact is displaced by asin(r/ρ). A plain
//     π/2 (armSectionArc, the planar setback rail) silently mis-places the weld here — that was the
//     latent Task-2 defect this constructor corrects.
//
// host carries the host cylinder radius R (=host.Radius); ρ = R−r (convex) / R+r (concave); convex
// selects the span sign. A non-torus, non-cylinder surface returns nil.
func curvedArmSectionArc(arm geom.Surface, station float64, host geom.Cylinder, r float64, convex bool) geom.Curve3 {
	if tor, ok := arm.(geom.Torus); ok {
		return torusSectionQuarter(tor, station)
	}
	if cyl, ok := arm.(geom.Cylinder); ok {
		return cylinderSectionArc(cyl, station, host, r, convex)
	}
	return nil
}

// torusSectionQuarter is the config-i torus arm's cross-section quarter-arc (m5-curved-arm-derivation.md
// §D5): at the corner u-station the tube circle v∈[0,π/2] runs from the cyl-contact (v=0, on the wall)
// to the plane-contact (v=π/2, on the cap). A true 90° quadrant — the axial section sees the cylinder
// as a straight ruling, so there is no in-section roll and no asin correction. Unchanged from Task 2.
func torusSectionQuarter(tor geom.Torus, station float64) geom.Curve3 {
	arc, err := geom.Arc3dByThreePoints(tor.PointAt(station, 0), tor.PointAt(station, armSectionMid), tor.PointAt(station, armSectionEnd))
	if err != nil {
		return nil
	}
	return arc
}

// armSectionMid / armSectionEnd are the tube v-angles of the torus cross-section quarter: the 45°
// bisector sample (for the through-point) and the plane-contact end. v=0 (the cyl-contact) is implicit.
const (
	armSectionMid = stdmath.Pi / 4
	armSectionEnd = stdmath.Pi / 2
)

// cylinderSectionArc is the config-ii cylinder arm's cross-section arc, on the arm cylinder (radius r,
// axis â, ref n̂_P) at the given station. Its endpoints are the rolling ball's two contact points at
// that station — the curved host wall (start) and the flat plane (end) — placed at the certified span
// π/2 ± asin(r/ρ) and joined through the span bisector (mirrors torusSectionQuarter's 3-point build,
// but at the true span, not π/2). Returns nil when no fillet of radius r fits the rim (see
// armSectionSpan). Scoped to the certified δ=90° edges (plane through the host axis), Slice A's corpus.
func cylinderSectionArc(arm geom.Cylinder, station float64, host geom.Cylinder, r float64, convex bool) geom.Curve3 {
	span, ok := armSectionSpan(host.Radius, r, convex)
	if !ok {
		return nil
	}
	center := arm.Origin.TranslateBy(arm.AxisDir.AsVector().Scale(station))
	wall, flat := armContactNormals(arm, center, host, span, convex)
	return sectionArcThrough(center, wall, flat, r)
}

// armSectionSpan returns the config-ii cross-section span and ok=false when no rolling-ball fillet of
// radius r exists on the rim {R, r, ρ}. Convex (protruding rim, ρ = R−r): π/2 + asin(r/ρ) — the
// contact rolls PAST 90° onto the convex wall by asin(r/ρ). Concave (root, ρ = R+r): π/2 − asin(r/ρ)
// — it rolls back. The sign is carried EXPLICITLY (never hard-coded +) so the concave branch is honest
// even though Slice A builds convex only. ok=false when r ≥ ρ, i.e. r > R/2 convex: r/ρ ≥ 1 leaves
// asin's domain and the two contacts go antipodal — no fillet fits (a NaN arc is never emitted).
// Concave ρ = R+r > r for all r > 0, so the concave branch never rejects (guard stays honest).
func armSectionSpan(hostRadius, r float64, convex bool) (float64, bool) {
	rho := hostRadius + r
	if convex {
		rho = hostRadius - r
	}
	if rho <= 0 || r >= rho {
		return 0, false // r=%v ≥ ρ=%v (R=%v): antipodal/out-of-domain, no radius-r fillet on this rim
	}
	roll := stdmath.Asin(r / rho)
	if !convex {
		roll = -roll
	}
	return stdmath.Pi/2 + roll, true
}

// armContactNormals returns the ball's two contact-normal unit vectors at the section station: `wall`,
// the host-cylinder contact set at exactly `span` from the flat and tilted to the binormal side of the
// ball centre's radial from the host axis (the physical wall, not its mirror ruling); and `flat`, the
// plane contact (∓n̂_P, toward the flat). Decomposed in the arm frame (n̂_P, binormal = â×n̂_P) so the
// asin span drives the wall roll explicitly: wall = flat·cos(span) + side·sin(span)·binormal.
func armContactNormals(arm geom.Cylinder, center math.Point3, host geom.Cylinder, span float64, convex bool) (wall, flat math.Vector3) {
	ref := arm.Ref.AsVector()
	binormal := arm.AxisDir.Cross(arm.Ref) // ⟂ â and n̂_P; unit in config ii
	d := footOnHostAxis(center, host).VectorTo(center)
	flat = ref.Scale(-signOf(d.Dot(ref)))                // face the flat (plane through the host axis)
	side := signOf(d.Dot(binormal)) * convexSign(convex) // wall side: outward for convex, inward for concave
	cos, sin := stdmath.Cos(span), stdmath.Sin(span)
	wall = flat.Scale(cos).Add(binormal.Scale(side * sin))
	return wall, flat
}

// sectionArcThrough builds the radius-r cross-section arc centred at the ball centre from `startDir`'s
// contact to `endDir`'s contact through their bisector (the through-point that fixes the short-arc
// winding). startDir/endDir are unit contact normals; the upstream span guard keeps them non-antipodal,
// so the bisector is well-defined — the arcBisectorTiny floor (ADR-0042, radius-scaled) is defensive.
func sectionArcThrough(center math.Point3, startDir, endDir math.Vector3, r float64) geom.Curve3 {
	bisector := startDir.Add(endDir)
	length := bisector.Length()
	if length < arcBisectorTiny*r {
		return nil
	}
	start := center.TranslateBy(startDir.Scale(r))
	mid := center.TranslateBy(bisector.Scale(r / length))
	end := center.TranslateBy(endDir.Scale(r))
	arc, err := geom.Arc3dByThreePoints(start, mid, end)
	if err != nil {
		return nil
	}
	return arc
}

// footOnHostAxis returns the perpendicular foot of p on the host cylinder's axis line — the point from
// which p's radial (p − foot, ⟂ â, length ρ at the ball centre) is measured.
func footOnHostAxis(p math.Point3, host geom.Cylinder) math.Point3 {
	a := host.AxisDir.AsVector()
	along := host.Origin.VectorTo(p).Dot(a)
	return host.Origin.TranslateBy(a.Scale(along))
}

// convexSign maps the convex/concave flag to the +1/−1 span-and-side sign, so callers carry the sign
// explicitly rather than hard-coding a convex assumption (m5-arm-section-derivation.md §sign).
func convexSign(convex bool) float64 {
	if convex {
		return 1
	}
	return -1
}
