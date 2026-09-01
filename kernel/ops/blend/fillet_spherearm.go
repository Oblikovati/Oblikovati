// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sphere-host arm fillets — the sphere-host trihedral-corner campaign, Slice SP1
// (sphere-host-corner-derivation.md §"The arms"). Rounding a CONVEX edge where a plane meets the
// host SPHERE (centre O, radius R, material inside) with a constant rolling-ball radius r is an
// EXACT torus in EVERY configuration: the ball-centre locus {|c−O|=ρ} ∩ {plane offset r inward}
// (ρ = R−r for the convex host) is always a circle — the torus's spine (major) circle — and the
// tube (minor) radius is r. There is no oblique "config iii" hole the cylinder host had, because a
// line/plane offset of a sphere is a concentric sphere and its planar section is always a circle.
// Dispatched from computeEdgeFillet BETWEEN the cylinderPlaneEdge branch and curvedAdjacentError, so
// a Plane∧Plane third edge keeps the existing straight cylinder arm and any non-sphere curved
// neighbour still reaches curvedAdjacentError unchanged (byte-identity: the 57 cylinder/planar greens
// have no sphere face, so none enter this branch). This slice is convex-external only (material
// INSIDE the sphere, ρ = R−r); the concave DIMPLE host (material outside, ρ = R+r) honest-rejects
// here (do-no-harm) — see sphereHostMaterialSign for why edge convexity alone cannot decide this.

// spherePlaneEdge reports an edge bounded by one host-sphere face and one plane face, returning both
// surfaces AND both topo.Faces. The plane FACE fixes the offset sign (its material-outward normal,
// Reversed-aware); the SPHERE FACE fixes the host material side (sphereHostMaterialSign) — a spherical
// dimple rim (Reversed sphere face, material OUTSIDE the sphere) is a genuinely CONVEX edge, so edge
// convexity cannot tell it apart from the convex-external host this slice supports, and only the
// sphere face's Reversed-aware normal can (sphere-host-corner-derivation.md §"Numerical pitfalls").
// The sibling of cylinderPlaneEdge for the sphere host.
func spherePlaneEdge(e *topo.Edge) (sp geom.Sphere, pl geom.Plane, sphereFace, planeFace *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Sphere{}, geom.Plane{}, nil, nil, false
	}
	for i := range 2 {
		s, oks := faces[i].Geometry().(geom.Sphere)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if oks && okp {
			return s, p, faces[i], faces[1-i], true
		}
	}
	return geom.Sphere{}, geom.Plane{}, nil, nil, false
}

// sphereArmReject names why a Sphere∧Plane edge could not be rounded, so the reject carries the REAL
// cause with its offending value (mirroring curvedFilletError for the cylinder host) instead of a
// generic string. sphereArmBuilt is the success sentinel (the arm was built, no reject).
type sphereArmReject uint8

const (
	sphereArmBuilt         sphereArmReject = iota // the torus arm was built — not a reject
	sphereArmVaryingRadius                        // r0≠r1: the sphere arm is constant-radius only
	sphereArmConcaveHost                          // material OUTSIDE the sphere (dimple, ρ = R+r) — a follow-on slice
	sphereArmSpindle                              // r ≥ R: ρ = R−r ≤ 0, the tube reaches the sphere centre
	sphereArmClears                               // |h| ≥ ρ: the plane's offset clears the offset sphere, no spine circle
	sphereArmTangent                              // spine circle collapsed to a point — a grazing/tangent plane, no corner
)

// sphereArmEdge dispatches a Sphere∧Plane edge to the torus-arm builder (SP1). handled=true means the
// edge WAS a sphere∧plane edge and computeEdgeFillet must return this result — either the built torus
// arm or the honest reject; handled=false leaves a non-sphere edge to the existing curvedAdjacentError
// path unchanged (byte-identity for the cylinder/planar corpus). Split out to keep computeEdgeFillet
// within funlen.
func sphereArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool, error) {
	sp, pl, sphereFace, planeFace, ok := spherePlaneEdge(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	res := tol.ForBody(body)
	ef, reason := sphereArmFillet(e, sp, pl, sphereFace, planeFace, p, res)
	if reason == sphereArmBuilt {
		return ef, true, nil // exact torus arm on a convex-external Sphere∧Plane rim (sphere-host campaign)
	}
	return edgeFillet{}, true, sphereArmError(reason, e, sp, sphereFace, p.r0) // do-no-harm, cause-specific reject
}

// sphereHostMaterialSign is the host material-side test that fixes whether this slice may round the
// edge. Edge convexity is NOT the host material side: a spherical DIMPLE rim (the sphere face is
// Reversed → material is OUTSIDE the sphere, meeting a plate top — the deburr-a-countersink case) is a
// genuinely CONVEX edge, yet its ball-centre locus needs |c−O| = R+r, not the R−r this slice builds
// (regression: SP1 review — a naked convexity guard silently built a wrong-side major = √(ρ²−h²) with
// ρ = R−r). s = (mid−O)·n̂ where n̂ is the sphere face's material-outward normal (Reversed-aware) at the
// edge midpoint mid and O is the sphere centre: s > 0 ⇒ material INSIDE the sphere (convex-external
// host, ρ = R−r, this slice) ⇒ build; s ≤ 0 ⇒ material OUTSIDE (concave dimple, ρ = R+r) ⇒ reject.
func sphereHostMaterialSign(e *topo.Edge, sp geom.Sphere, sphereFace *topo.Face) (float64, bool) {
	mid := edgeMidpoint(e)
	n, ok := outwardFaceNormal(sphereFace, mid)
	if !ok {
		return 0, false
	}
	return float64(sp.Center.VectorTo(mid).Dot(n)), true
}

// sphereArmFillet builds the exact torus arm on a CONVEX-EXTERNAL Sphere∧Plane edge — the sibling of
// curvedArmFillet for the sphere host, carried in the same edgeFillet the straight-edge path emits.
// Returns a non-sphereArmBuilt reason — so the caller honest-rejects via sphereArmError (do-no-harm) —
// for a varying radius, a concave dimple host (material outside the sphere; sphereHostMaterialSign),
// or any constructor decline (spindle r≥R, the plane clearing the offset sphere, or a grazing tangency).
func sphereArmFillet(e *topo.Edge, sp geom.Sphere, pl geom.Plane, sphereFace, planeFace *topo.Face, p filletPick, res tol.Resolution) (edgeFillet, sphereArmReject) {
	if p.varying() {
		return edgeFillet{}, sphereArmVaryingRadius
	}
	if s, ok := sphereHostMaterialSign(e, sp, sphereFace); !ok || s <= 0 {
		return edgeFillet{}, sphereArmConcaveHost // material outside the sphere: ρ = R+r is a follow-on slice
	}
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, sphereArmTangent // degenerate plane normal — no offset direction
	}
	tor, reason := sphereArmSurface(sp, pl, n, p.r0, res)
	if reason != sphereArmBuilt {
		return edgeFillet{}, reason
	}
	if ef, ok := curvedArmEdgeFillet(e, tor, true); ok {
		return ef, sphereArmBuilt
	}
	return edgeFillet{}, sphereArmTangent
}

// sphereArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the convex edge
// where plane pl (material-outward unit normal outwardN) meets the host sphere sp
// (sphere-host-corner-derivation.md §"The arms"). ρ = R−r; the offset plane is pl moved r into the
// material (−n̂); h = n̂·(O−p)+r is the signed distance from the sphere centre O to that offset plane;
// the spine circle is the foot O′ = O − h·n̂ with major radius √(ρ²−h²) in the offset plane; the tube
// minor radius is r. Built via NewTorusWithRef so the u=0 seam aligns with the plane's in-plane axis
// (SP3 may re-pin this to the corner rail). Returns a non-sphereArmBuilt reason for a spindle
// (sphereArmSpindle: r ≥ R, the tube reaches the sphere centre), when the offset plane clears the
// offset sphere (sphereArmClears: |h| ≥ ρ, no spine circle), or a grazing tangency (sphereArmTangent:
// the spine circle collapsed to a point) — the last two as length bands scaled by res (ADR-0042).
//
// Example: sphereArmSurface(sphere{O:origin,R:150}, capPlane{z:129.9038,n̂:+ẑ}, +ẑ, 10, res) → the D5
// rim torus centre (0,0,119.9038), axis ẑ, major √5223.08 = 72.2709, minor 10 (oracle-matched ≤2.3e-4).
func sphereArmSurface(sp geom.Sphere, pl geom.Plane, outwardN math.UnitVector3, r float64, res tol.Resolution) (geom.Torus, sphereArmReject) {
	rho := sp.Radius - r
	if rho < armSpindleBand*res.Weld() {
		return geom.Torus{}, sphereArmSpindle // spindle: r ≥ R, the tube reaches the sphere centre
	}
	n := outwardN.AsVector()
	h := pl.Origin.VectorTo(sp.Center).Dot(n) + r // signed distance O → offset plane (moved r into the material)
	spineSq := rho*rho - h*h
	if spineSq <= 0 {
		return geom.Torus{}, sphereArmClears // the offset plane clears the offset sphere — no spine circle
	}
	majorR := stdmath.Sqrt(spineSq)
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, sphereArmTangent // grazing tangency: the spine circle has collapsed to a point
	}
	center := sp.Center.TranslateBy(n.Scale(-h))
	tor, err := geom.NewTorusWithRef(center, n, pl.UAxis.AsVector(), majorR, r)
	if err != nil {
		return geom.Torus{}, sphereArmTangent // degenerate torus frame — treat as a collapsed corner
	}
	return tor, sphereArmBuilt
}

// sphereArmError reports why a Sphere∧Plane edge could not be rounded when sphereArmFillet declined,
// each message naming the ACTUAL cause and its offending value — mirroring curvedFilletError for the
// cylinder host rather than the misleading generic curvedAdjacentError. The concave-host case
// recomputes and carries the measured material-side sign s (the reject's provenance).
func sphereArmError(reason sphereArmReject, e *topo.Edge, sp geom.Sphere, sphereFace *topo.Face, r float64) error {
	switch reason {
	case sphereArmConcaveHost:
		s, _ := sphereHostMaterialSign(e, sp, sphereFace)
		return fmt.Errorf("fillet: cannot round this Sphere∧Plane edge with radius %g — the host is a CONCAVE "+
			"sphere (material OUTSIDE the sphere; material-side sign s=%g ≤ 0), which needs ρ = R+r = %g, not the "+
			"convex-external R−r this slice builds; concave sphere host (R+r) is a follow-on slice", r, s, sp.Radius+r)
	case sphereArmSpindle:
		return fmt.Errorf("fillet: cannot round this Sphere∧Plane edge — spindle: radius %g ≥ host R=%g "+
			"(ρ = R−r ≤ 0, the tube reaches the sphere centre)", r, sp.Radius)
	case sphereArmClears:
		return fmt.Errorf("fillet: cannot round this Sphere∧Plane edge with radius %g — the plane's r-offset "+
			"clears the offset sphere (|h| ≥ ρ = R−r = %g): the ball cannot touch both surfaces", r, sp.Radius-r)
	case sphereArmTangent:
		return fmt.Errorf("fillet: edge between the host sphere (R=%g) and a grazing/tangent plane is smooth "+
			"(the spine circle collapsed to a point) — no corner to round with radius %g", sp.Radius, r)
	default: // sphereArmVaryingRadius
		return fmt.Errorf("fillet: a varying-radius pick (r0≠r1) on a Sphere∧Plane edge is not supported; "+
			"the sphere arm requires a constant radius (got %g)", r)
	}
}
