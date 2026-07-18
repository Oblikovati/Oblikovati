// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
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
// have no sphere face, so none enter this branch). This slice is convex-external only (s = +1); the
// concave bore (ρ = R+r) is a follow-on slice and honest-rejects here (do-no-harm).

// spherePlaneEdge reports an edge bounded by one host-sphere face and one plane face, returning both
// surfaces and the plane's topo.Face — the plane's material-outward normal (from the FACE, respecting
// Reversed()) fixes the offset sign, and a flipped normal would build a valid-looking ball on the
// wrong side of the host (sphere-host-corner-derivation.md §"Numerical pitfalls"). The sibling of
// cylinderPlaneEdge for the sphere host.
func spherePlaneEdge(e *topo.Edge) (sp geom.Sphere, pl geom.Plane, planeFace *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Sphere{}, geom.Plane{}, nil, false
	}
	for i := 0; i < 2; i++ {
		s, oks := faces[i].Geometry().(geom.Sphere)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if oks && okp {
			return s, p, faces[1-i], true
		}
	}
	return geom.Sphere{}, geom.Plane{}, nil, false
}

// sphereArmEdge dispatches a Sphere∧Plane edge to the torus-arm builder (SP1). handled=true means the
// edge WAS a sphere∧plane edge and computeEdgeFillet must return this result — either the built torus
// arm or the honest reject; handled=false leaves a non-sphere edge to the existing curvedAdjacentError
// path unchanged (byte-identity for the cylinder/planar corpus). Split out to keep computeEdgeFillet
// within funlen.
func sphereArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool, error) {
	sp, pl, pf, ok := spherePlaneEdge(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	if ef, built := sphereArmFillet(e, sp, pl, pf, p, ResolutionForBody(body)); built {
		return ef, true, nil // exact torus arm on a convex Sphere∧Plane rim (sphere-host campaign)
	}
	return edgeFillet{}, true, sphereArmError(sp, p.r0) // concave bore / spindle / grazing — do-no-harm
}

// sphereArmFillet builds the exact torus arm on a CONVEX Sphere∧Plane edge — the sibling of
// curvedArmFillet for the sphere host, carried in the same edgeFillet the straight-edge path emits.
// Returns false — so the caller honest-rejects via sphereArmError (do-no-harm) — for a varying
// radius, a concave/bore edge (this slice is convex-external only), or any constructor decline
// (spindle R≤r, or a grazing plane that clears the offset sphere so the ball cannot touch both).
func sphereArmFillet(e *topo.Edge, sp geom.Sphere, pl geom.Plane, planeFace *topo.Face, p filletPick, res Resolution) (edgeFillet, bool) {
	if p.varying() || ClassifyEdgeConvexity(e) != EdgeConvex {
		return edgeFillet{}, false // constant-radius convex-external only (concave bore is a follow-on slice)
	}
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, false
	}
	tor, ok := sphereArmSurface(sp, pl, n, p.r0, res)
	return curvedArmEdgeFillet(e, tor, ok)
}

// sphereArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the convex edge
// where plane pl (material-outward unit normal outwardN) meets the host sphere sp
// (sphere-host-corner-derivation.md §"The arms"). ρ = R−r; the offset plane is pl moved r into the
// material (−n̂); h = n̂·(O−p)+r is the signed distance from the sphere centre O to that offset plane;
// the spine circle is the foot O′ = O − h·n̂ with major radius √(ρ²−h²) in the offset plane; the tube
// minor radius is r. Built via NewTorusWithRef so the u=0 seam aligns with the plane's in-plane axis
// (SP3 may re-pin this to the corner rail). Returns false for a spindle (ρ < band: r ≥ R, the tube
// reaches the sphere centre) or when |h| ≥ ρ (the offset plane clears the offset sphere: no spine
// circle — the ball cannot touch both), both as length bands scaled by res (ADR-0042).
//
// Example: sphereArmSurface(sphere{O:origin,R:150}, capPlane{z:129.9038,n̂:+ẑ}, +ẑ, 10, res) → the D5
// rim torus centre (0,0,119.9038), axis ẑ, major √5223.08 = 72.2709, minor 10 (oracle-matched ≤2.3e-4).
func sphereArmSurface(sp geom.Sphere, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution) (geom.Torus, bool) {
	rho := sp.Radius - r
	if rho < armSpindleBand*res.Weld() {
		return geom.Torus{}, false // spindle: r ≥ R, the tube reaches the sphere centre
	}
	n := outwardN.AsVector()
	h := pl.Origin.VectorTo(sp.Center).Dot(n) + r // signed distance O → offset plane (moved r into the material)
	spineSq := rho*rho - h*h
	if spineSq <= 0 {
		return geom.Torus{}, false // the offset plane clears the offset sphere — no spine circle
	}
	majorR := stdmath.Sqrt(spineSq)
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false // grazing tangency: the spine circle has collapsed to a point
	}
	center := sp.Center.TranslateBy(n.Scale(-h))
	tor, err := geom.NewTorusWithRef(center, n, pl.UAxis.AsVector(), majorR, r)
	return tor, err == nil
}

// sphereArmError reports why a Sphere∧Plane edge could not be rounded when sphereArmFillet declined —
// a concave bore (ρ = R+r, out of this slice), a spindle (r ≥ R), or a grazing plane that clears the
// offset sphere. An honest, config-specific reject (do-no-harm) — mirroring curvedFilletError for the
// cylinder host — rather than the misleading generic curvedAdjacentError.
func sphereArmError(sp geom.Sphere, r float64) error {
	return fmt.Errorf("fillet: cannot round this Sphere∧Plane edge with radius %g — only a convex sphere host "+
		"(material inside, r < R = %g) is supported; concave bore and spindle are follow-on slices", r, sp.Radius)
}
