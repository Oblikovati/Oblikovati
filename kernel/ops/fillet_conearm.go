// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-host arm fillets — the cone-host trihedral-corner campaign, Slice CN1
// (cone-host-corner-derivation.md §2 "Arm A"). Rounding a CONVEX edge where a plane meets the host
// CONE (apex A, axis â into the opening, half-angle α, material inside) with a constant rolling-ball
// radius r is an EXACT torus when the plane is the CAP plane (⊥ the axis, the circle edge): the inner
// r-offset of a cone is the coaxial cone with the same α and apex shifted r/sinα along +â, and its
// planar section by the offset cap plane is a circle — the torus's spine (major) circle — with minor
// radius r. The OTHER cone∧plane configuration, the RULING edge (plane CONTAINS the axis), is a canal
// over a hyperbola — NOT a torus — built EXACTLY by coneCanalArmFillet (fillet_conecanal.go, CN2).
// Dispatched from computeEdgeFillet AFTER sphereArmEdge, before curvedAdjacentError, so every non-cone edge keeps
// its existing path byte-identically (a Cone∧Plane edge exists only in the still-red cone cases, and
// their corner solve fails first at "corner face must be planar" — CN1 greens nothing on its own).
// Convex-external only (material INSIDE the cone, A′ = A + r/sinα·â); the concave conical bore (material
// outside, A′ = A − r/sinα·â) honest-rejects here (do-no-harm) — see coneHostMaterialSign.

// conePlaneEdge reports an edge bounded by one host-cone face and one plane face, returning both
// surfaces AND both topo.Faces. The plane FACE fixes the offset sign (its material-outward normal,
// Reversed-aware); the CONE FACE fixes the host material side (coneHostMaterialSign) — a conical bore
// rim (Reversed cone face, material OUTSIDE the cone) is a genuinely CONVEX edge, so edge convexity
// cannot tell it apart from the convex-external host this slice supports, and only the cone face's
// Reversed-aware normal can. The sibling of spherePlaneEdge for the cone host.
func conePlaneEdge(e *topo.Edge) (co geom.Cone, pl geom.Plane, coneFace, planeFace *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Cone{}, geom.Plane{}, nil, nil, false
	}
	for i := 0; i < 2; i++ {
		c, okc := faces[i].Geometry().(geom.Cone)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okc && okp {
			return c, p, faces[i], faces[1-i], true
		}
	}
	return geom.Cone{}, geom.Plane{}, nil, nil, false
}

// coneArmReject names why a Cone∧Plane edge could not be rounded, so the reject carries the REAL cause
// with its offending value (mirroring sphereArmReject) instead of a generic string. coneArmBuilt is the
// success sentinel (the arm was built, no reject).
type coneArmReject uint8

const (
	coneArmBuilt            coneArmReject = iota // the arm (torus or canal) was built — not a reject
	coneArmVaryingRadius                         // r0≠r1: the cone arm is constant-radius only
	coneArmConcaveBore                           // material OUTSIDE the cone (bore, s=−1) — a follow-on slice
	coneArmOblique                               // the plane is neither ⊥ nor ∥ the axis (oblique) — out of slice
	coneArmNearCylinder                          // sinα below band: apex shift r/sinα blows up (a true cylinder host)
	coneArmNearPlane                             // cosα below band: near-plane cone (α→π/2)
	coneArmClears                                // h′ ≤ 0: the offset cap plane clears the offset cone, no spine circle
	coneArmGrazing                               // spine (major) radius R_s below band — a grazing/tangent cap
	coneArmDegenerate                            // the torus/canal/normal constructor declined — a collapsed frame
	coneArmRulingNoFit                           // ruling canal (CN2): the ball never fits the picked span
	coneArmRulingSpan                            // ruling canal (CN2): the fittable x_f span collapses
	coneArmRulingFold                            // ruling canal (CN2): a band-arc station is irregular (self-intersects)
	coneArmRulingUnresolved                      // ruling canal (CN2): between-station envelope error over bound at the station cap
)

// coneArmClass is which cone∧plane configuration an edge is, decided by the angle between the plane
// normal and the cone axis (the cone sibling of classifyCurvedArm): plane ⊥ axis (cap plane) makes the
// edge a circle and the arm an exact TORUS (this task); plane ∥ axis (containing the axis) makes the
// edge a ruling and the arm a CANAL over a hyperbola (coneClassRuling, CN2); anything oblique has no
// closed-form arm in this corpus (coneClassOblique).
type coneArmClass uint8

const (
	coneClassOblique coneArmClass = iota // neither ⊥ nor ∥ the axis: no closed-form arm in this corpus
	coneClassTorus                       // cap plane ⊥ axis: circle edge, exact torus arm (§2 Arm A)
	coneClassRuling                      // plane contains the axis: ruling edge, exact canal arm (§2 Arm B, CN2)
)

// coneArmClassifyCoef is k in the ⊥/∥ classification band ε_ang = k·res.Weld()/R_edge (ADR-0042): a
// MODEL-relative angular band — the weld resolution over the actual EDGE radius (the cone has no
// intrinsic length), matching M5's angArmClassifyCoef which divides by the cylinder radius (CN1-review
// Minor #1: the earlier res.Size() collapsed the band to a scale-free constant). k=3 sits mid-band of
// the derivation's k≈2..4. Over-tight (a large R_edge) errs toward an honest oblique-reject — conservative.
const coneArmClassifyCoef = 3

// coneAlphaBandCoef is k in the α-limit existence bands sinα < k·res.Weld()/L (near-cylinder: the apex
// shift r/sinα blows up) and cosα < k·res.Weld()/L (near-plane, α→π/2), where L = |A − plane.Origin| is
// the apex-to-cap-plane length standing in for |A−v| (cone-host-corner-derivation.md §"α-limit
// conditioning"). Both are length bands scaled to the model, never bare constants.
const coneAlphaBandCoef = 3

// classifyConeArm decides which cone∧plane configuration the edge (co, pl) is (coneClassTorus when the
// plane is (near-)perpendicular to the axis, coneClassRuling when (near-)parallel, coneClassOblique
// otherwise). rEdge is the cone radius at the edge (the arc/ruling radius) — the classification band is
// ε_ang = k·res.Weld()/R_edge (CN1-review Minor #1: model-relative to the EDGE, not res.Size()).
func classifyConeArm(co geom.Cone, pl geom.Plane, rEdge float64, res Resolution) coneArmClass {
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return coneClassOblique
	}
	s := stdmath.Abs(co.AxisDir.Dot(n))
	epsAng := coneArmClassifyCoef * res.Weld() / stdmath.Max(rEdge, res.Weld())
	switch {
	case s > 1-epsAng:
		return coneClassTorus // cap plane ⊥ axis — the circle edge (torus arm, CN1)
	case s < epsAng:
		return coneClassRuling // plane contains the axis — the ruling edge (canal arm, CN2)
	default:
		return coneClassOblique
	}
}

// coneRadiusAt is the cone's radius at point p: the perpendicular distance from p to the axis. It stands
// in for the edge's arc/ruling radius R_edge in the model-relative classification band.
func coneRadiusAt(co geom.Cone, p math.Point3) float64 {
	w := co.Apex.VectorTo(p)
	a := co.AxisDir.AsVector()
	perp := w.Sub(a.Scale(float64(w.Dot(a))))
	return float64(perp.Length())
}

// coneHostMaterialSign is the host material-side test that fixes whether this slice may round the edge.
// Edge convexity is NOT the host material side: a conical BORE rim (the cone face is Reversed → material
// is OUTSIDE the cone) is a genuinely CONVEX edge, yet its offset apex is A − r/sinα·â, not the
// A + r/sinα·â this slice builds. s = n̂·êr where n̂ is the cone face's material-outward normal
// (Reversed-aware) at the edge midpoint and êr is the outward radial direction there: s > 0 ⇒ material
// INSIDE the cone (convex-external host, this slice) ⇒ build; s ≤ 0 ⇒ material OUTSIDE (concave bore) ⇒
// reject. Evaluated at the edge midpoint (on the rim, AWAY from the apex) — never at the apex, where the
// radial direction is 0/0 (cone-host-corner-derivation.md §"Apex singularity").
func coneHostMaterialSign(e *topo.Edge, co geom.Cone, coneFace *topo.Face) (float64, bool) {
	mid := edgeMidpoint(e)
	n, ok := outwardFaceNormal(coneFace, mid)
	if !ok {
		return 0, false
	}
	radial, err := coneRadialDir(co, mid)
	if err != nil {
		return 0, false // on the axis (apex): no radial direction — caller honest-rejects
	}
	return float64(n.Dot(radial)), true
}

// coneRadialDir is the outward unit radial direction (perpendicular to the axis) at point p — the
// component of (p − A) with its axial part removed. Errors when p lies on the axis (the apex), where the
// radial direction is undefined.
func coneRadialDir(co geom.Cone, p math.Point3) (math.Vector3, error) {
	a := co.AxisDir.AsVector()
	w := co.Apex.VectorTo(p)
	perp := w.Sub(a.Scale(w.Dot(a)))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return math.Vector3{}, err
	}
	return u.AsVector(), nil
}

// coneArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the CONVEX CAP-plane
// (circle) edge where plane pl (material-outward unit normal outwardN) meets the host cone co
// (cone-host-corner-derivation.md §2 Arm A). Offset apex A′ = A + s·(r/sinα)·â (s = +1 convex-external);
// the offset cap plane (pl moved r into the material) sits at height h′ = (P_off − A′)·â above A′; the
// spine circle is the foot O′ = A′ + h′·â with major radius R_s = tanα·h′; the tube minor radius is r.
// Built via NewTorusWithRef so the u=0 seam aligns with the cone's angle-zero frame. Returns a
// non-coneArmBuilt reason for a near-cylinder / near-plane cone (α bands), when the offset cap plane
// clears the offset cone (coneArmClears: h′ ≤ 0), or a grazing tangency (coneArmGrazing: R_s below band)
// — the length bands scaled by res (ADR-0042).
//
// Example: coneArmSurface(C2 frustum apex (0,0,270) â=−ẑ tanα=1/3, cap plane z=0 outward −ẑ, −ẑ, +1, 10,
// res) → torus centre (0,0,10), axis −ẑ, major 76.1257411328, minor 10 (the C2 bottom-rim arm).
func coneArmSurface(co geom.Cone, pl geom.Plane, outwardN math.UnitVector3, s, r float64, res Resolution) (geom.Torus, coneArmReject) {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	aband := coneAlphaBandCoef * res.Weld() / stdmath.Max(float64(co.Apex.DistanceTo(pl.Origin)), res.Weld())
	if sinA < aband {
		return geom.Torus{}, coneArmNearCylinder // apex shift r/sinα blows up — a true cylinder host takes M5
	}
	if cosA < aband {
		return geom.Torus{}, coneArmNearPlane // near-plane cone (α→π/2)
	}
	a := co.AxisDir.AsVector()
	apexPrime := co.Apex.TranslateBy(a.Scale(s * r / sinA))      // A′ = A + s·(r/sinα)·â
	pOff := pl.Origin.TranslateBy(outwardN.AsVector().Scale(-r)) // cap plane moved r into the material
	return coneOffsetTorus(co, apexPrime, pOff, r, res)          // shared spine/major/torus tail (dedup with the concave builder)
}

// coneArmFillet builds the exact torus arm on a CONVEX-EXTERNAL Cone∧Plane CAP-plane edge — the sibling
// of sphereArmFillet for the cone host, carried in the same edgeFillet the straight-edge path emits.
// Returns a non-coneArmBuilt reason — so the caller honest-rejects via coneArmError (do-no-harm) — for a
// varying radius, the ruling edge (canal, CN2), an oblique plane, a concave bore (material outside the
// cone; coneHostMaterialSign), or any surface decline (α bands, clearance, grazing).
func coneArmFillet(e *topo.Edge, co geom.Cone, pl geom.Plane, coneFace, planeFace *topo.Face, p filletPick, res Resolution) (edgeFillet, coneArmReject) {
	if p.varying() {
		return edgeFillet{}, coneArmVaryingRadius
	}
	switch classifyConeArm(co, pl, coneRadiusAt(co, edgeMidpoint(e)), res) {
	case coneClassRuling:
		return coneCanalArmFillet(e, co, pl, coneFace, planeFace, p.r0, res) // exact canal arm (CN2)
	case coneClassOblique:
		return edgeFillet{}, coneArmOblique
	}
	if sgn, ok := coneHostMaterialSign(e, co, coneFace); !ok || sgn <= 0 {
		return edgeFillet{}, coneArmConcaveBore // material outside the cone: A′ = A − r/sinα·â is a follow-on slice
	}
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, coneArmDegenerate // degenerate plane normal — no offset direction
	}
	tor, reason := coneArmSurface(co, pl, n, +1, p.r0, res)
	if reason != coneArmBuilt {
		return edgeFillet{}, reason
	}
	if ef, ok := curvedArmEdgeFillet(e, tor, true); ok {
		return ef, coneArmBuilt
	}
	return edgeFillet{}, coneArmDegenerate
}

// coneArmEdge dispatches a Cone∧Plane edge to the torus-arm builder (CN1). handled=true means the edge
// WAS a cone∧plane edge and computeEdgeFillet must return this result — either the built torus arm or the
// honest reject; handled=false leaves a non-cone edge to the existing curvedAdjacentError path unchanged
// (byte-identity for the cylinder/planar/sphere corpus). The sibling of sphereArmEdge.
func coneArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool, error) {
	co, pl, coneFace, planeFace, ok := conePlaneEdge(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	res := ResolutionForBody(body)
	ef, reason := coneArmFillet(e, co, pl, coneFace, planeFace, p, res)
	if reason == coneArmBuilt {
		return ef, true, nil // exact torus arm on a convex-external Cone∧Plane cap (circle) edge (cone-host campaign)
	}
	return edgeFillet{}, true, coneArmError(reason, e, co, coneFace, p.r0) // do-no-harm, cause-specific reject
}

// coneArmError reports why a Cone∧Plane edge could not be rounded when coneArmFillet declined, each
// message naming the ACTUAL cause and its offending value — mirroring sphereArmError. Split into the
// classify/material rejects (recognizer stage) and the surface-construction rejects (α bands, clearance,
// grazing) so each helper stays within funlen.
func coneArmError(reason coneArmReject, e *topo.Edge, co geom.Cone, coneFace *topo.Face, r float64) error {
	switch reason {
	case coneArmConcaveBore, coneArmOblique, coneArmVaryingRadius:
		return coneArmClassifyError(reason, e, co, coneFace, r)
	case coneArmRulingNoFit, coneArmRulingSpan, coneArmRulingFold, coneArmRulingUnresolved:
		return coneCanalArmError(reason, co, r) // the ruling-edge canal build declines (CN2)
	default:
		return coneArmSurfaceError(reason, co, r)
	}
}

// coneArmClassifyError names the recognizer-stage rejects: a concave bore host (material outside the
// cone, recomputing the measured material-side sign s as provenance), an oblique plane, or a
// varying-radius pick. (The ruling edge no longer rejects here — CN2 builds its canal arm.)
func coneArmClassifyError(reason coneArmReject, e *topo.Edge, co geom.Cone, coneFace *topo.Face, r float64) error {
	switch reason {
	case coneArmConcaveBore:
		s, _ := coneHostMaterialSign(e, co, coneFace)
		return fmt.Errorf("fillet: cannot round this Cone∧Plane edge with radius %g — the host is a CONCAVE cone "+
			"(material OUTSIDE the cone; material-side sign s=%g ≤ 0), which needs A′ = A − r/sinα·â; the concave "+
			"conical bore is a follow-on slice", r, s)
	case coneArmOblique:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane edge with radius %g — the plane is neither ⊥ nor ∥ the "+
			"cone axis (oblique, half-angle %g); an oblique cone∧plane arm is out of slice", r, co.HalfAngle)
	default: // coneArmVaryingRadius
		return fmt.Errorf("fillet: a varying-radius pick (r0≠r1) on a Cone∧Plane edge is not supported; the cone arm "+
			"requires a constant radius (got %g)", r)
	}
}

// coneArmSurfaceError names the surface-construction rejects: a near-cylinder or near-plane cone (α
// limits), the r-offset cap plane clearing the offset cone (h′ ≤ 0), a grazing cap (R_s below band), or
// a degenerate frame — each carrying the offending half-angle / radius.
func coneArmSurfaceError(reason coneArmReject, co geom.Cone, r float64) error {
	switch reason {
	case coneArmNearCylinder:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane edge with radius %g — cone half-angle %g is too small "+
			"(sinα below band; the apex shift r/sinα blows up); a true cylinder host takes the M5 path", r, co.HalfAngle)
	case coneArmNearPlane:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane edge with radius %g — cone half-angle %g is too close to "+
			"π/2 (cosα below band; a near-plane cone)", r, co.HalfAngle)
	case coneArmClears:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane edge with radius %g — the r-offset cap plane clears the "+
			"offset cone (h′ ≤ 0): the ball cannot touch both surfaces", r)
	case coneArmGrazing:
		return fmt.Errorf("fillet: edge between the host cone (half-angle %g) and a grazing cap plane is smooth (the "+
			"spine circle collapsed to a point) — no corner to round with radius %g", co.HalfAngle, r)
	default: // coneArmDegenerate
		return fmt.Errorf("fillet: degenerate Cone∧Plane arm frame — no rolling-ball arm for radius %g", r)
	}
}
