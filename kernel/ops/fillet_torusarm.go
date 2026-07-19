// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Torus-host arm fillets — Slice E7 (OCCT blend/simple/E7). Rounding a CONVEX edge where a plane meets a
// host TORUS (centre O, axis ẑ, major R, minor a, material inside the tube) with a constant rolling-ball
// radius r is an EXACT torus ONLY when the cap plane is a LATITUDE cut — perpendicular to the torus axis
// (the circle rim). The ball-centre spine is then the coaxial circle at axial coord η with major
// M_a = R + ε·√(b²−η²) (b = a−r the r-shrunk tube, ε the outer/inner branch), and the arm is the torus of
// that spine with minor radius r. When the cap plane is instead PARALLEL to the axis (a MERIDIAN cut, the
// spiric E5/E9 cases) the ball-centre spine is a quartic spiric section of Perseus, NOT a circle, so no
// exact-torus arm exists — torusArmMeridian honest-rejects it (do-no-harm), keeping E5/E9 at their floor.
// Dispatched from curvedHostArmEdge LAST (after cone), so every non-torus host mix falls through unchanged
// and only a PICKED Torus∧Plane edge reaches here (computeEdgeFillet runs only on picked edges): the whole
// corpus picks a Torus∧Plane edge in exactly the E5/E7/E9 simple cases, so nothing else can flip.
// Convex-external-tube only (material INSIDE the tube, b = a−r); a concave toroidal bore does not occur in
// this corpus and is a follow-on slice.

// torusPlaneEdge reports an edge bounded by one host-torus face and one plane face, returning the torus,
// the plane, and the plane FACE (whose Reversed-aware material-outward normal fixes both the latitude gate
// and the offset sign). The sibling of spherePlaneEdge / conePlaneEdge for the torus host.
func torusPlaneEdge(e *topo.Edge) (host geom.Torus, pl geom.Plane, planeFace *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Torus{}, geom.Plane{}, nil, false
	}
	for i := 0; i < 2; i++ {
		h, okh := faces[i].Geometry().(geom.Torus)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okh && okp {
			return h, p, faces[1-i], true
		}
	}
	return geom.Torus{}, geom.Plane{}, nil, false
}

// torusArmReject names why a Torus∧Plane edge could not be rounded, so the reject carries the REAL cause
// with its offending value (mirroring sphereArmReject / coneArmReject). torusArmBuilt is the success
// sentinel (the arm was built, no reject).
type torusArmReject uint8

const (
	torusArmBuilt         torusArmReject = iota // the exact torus arm was built — not a reject
	torusArmVaryingRadius                       // r0≠r1: the torus arm is constant-radius only
	torusArmMeridian                            // cap plane ∥ axis — a spiric canal (E5/E9), no exact-torus arm
	torusArmSpindle                             // r ≥ a: b = a−r ≤ 0, the offset tube reaches its own centre
	torusArmClears                              // d² = b²−η² ≤ 0: the cap clears the offset torus, no spine circle
	torusArmGrazing                             // spine circle collapsed onto the tube — a grazing/tangent cap
	torusArmDegenerate                          // the torus/normal constructor declined — a collapsed frame
)

// torusLatitudeCoef is k in the latitude-vs-meridian band ε_ang = k·res.Weld()/R_rim (ADR-0042): a
// model-relative angular band — the weld resolution over the rim radius (matching coneArmClassifyCoef,
// which divides by the edge radius). Only |ẑ·n̂| > 1−ε_ang (cap ⊥ axis) is admitted; a meridian cut
// (|ẑ·n̂| near 0, E5/E9) rejects. k=3 sits mid-band.
const torusLatitudeCoef = 3

// torusLatitudeCut reports whether the cap plane is (near-)perpendicular to the torus axis — the ONLY
// configuration with an exact-torus arm. nDotZ is ẑ·n̂ (signed); the band is model-relative to the rim.
func torusLatitudeCut(nDotZ, rhoRim float64, res Resolution) bool {
	epsAng := torusLatitudeCoef * res.Weld() / stdmath.Max(rhoRim, res.Weld())
	return stdmath.Abs(nDotZ) > 1-epsAng
}

// torusRimRadius is the rim's radial coord ρ_rim: the perpendicular distance from the edge midpoint to the
// torus axis. It fixes the branch ε (outer equator vs inner) in torusBranchSign.
func torusRimRadius(e *topo.Edge, host geom.Torus) float64 {
	w := host.Center.VectorTo(edgeMidpoint(e))
	z := host.AxisDir.AsVector()
	return w.Sub(z.Scale(w.Dot(z))).Length()
}

// torusBranchSign is ε = sign(ρ_rim − R): +1 when the rim is OUTSIDE the equator (the outer branch),
// −1 when inside. On the equator (|ρ_rim − R| within a model-relative band) it snaps to +1 (brief).
func torusBranchSign(rhoRim, majorR float64, res Resolution) float64 {
	if rhoRim-majorR < -armSpindleBand*res.Weld() {
		return -1
	}
	return +1
}

// torusHostArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the CONVEX LATITUDE
// (circle) edge where a cap plane at axial coord capH (material-outward normal component nDotZ = ẑ·n̂)
// meets the host torus (§Torus-host arm fillets header). b = a−r is the r-shrunk tube the ball centre
// rides; η = capH − r·nDotZ is the arm spine axial coord; d² = b²−η²; the spine circle is coaxial at
// O + η·ẑ with major M_a = R + ε·√(d²) and minor r. Built via NewTorusWithRef so the u=0 seam aligns with
// the host torus frame. Returns a non-torusArmBuilt reason for a spindle (r ≥ a), when the cap clears the
// offset torus (d² ≤ 0), a grazing tangency (√(d²) below band), or a degenerate frame — the length bands
// scaled by res (ADR-0042).
//
// Example: torusHostArmSurface(torus{O:origin,axis:+ẑ,R:100,a:100}, capH:70.7107, ρ_rim:170.71, r:10, nDotZ:+1,
// res) → the E7 arm torus centre (0,0,60.7107), axis +ẑ, major 166.4395, minor 10 (oracle-matched).
func torusHostArmSurface(host geom.Torus, capH, rhoRim, r, nDotZ float64, res Resolution) (geom.Torus, torusArmReject) {
	b := host.MinorRadius - r
	if b < armSpindleBand*res.Weld() {
		return geom.Torus{}, torusArmSpindle // r ≥ a: the offset tube reaches its own centre
	}
	eta := capH - r*nDotZ // arm spine axial coord
	d2 := b*b - eta*eta
	if d2 <= 0 {
		return geom.Torus{}, torusArmClears // the cap clears the offset torus — no spine circle
	}
	dd := stdmath.Sqrt(d2)
	if dd < armSpindleBand*res.Weld() {
		return geom.Torus{}, torusArmGrazing // grazing cap: the spine circle has collapsed onto the tube
	}
	majorR := host.MajorRadius + torusBranchSign(rhoRim, host.MajorRadius, res)*dd
	z := host.AxisDir.AsVector()
	tor, err := geom.NewTorusWithRef(host.Center.TranslateBy(z.Scale(eta)), z, host.Ref.AsVector(), majorR, r)
	if err != nil {
		return geom.Torus{}, torusArmDegenerate
	}
	return tor, torusArmBuilt
}

// torusHostContactCircle is the torus-arm↔host-torus contact (E7): the arm's rolling ball, centred on the
// arm spine circle coaxial with the host torus, touches the host tube along a coaxial circle. With b = a−r
// the r-shrunk host tube and k = a/b, the contact circle is host.Center + k·(arm.Center−host.Center) with
// radius R + k·(M_a − R) — the SAME (a/b) scaling the E7 derivation gives (this file's header). The result
// circle lies in a plane ⊥ ẑ (the host/arm frame), so curvedHostArc's [0→φ] sweep lands correctly. Rejects
// a non-coaxial arm, a spindle host (b ≤ 0), or a tube not internally tangent (√((M_a−R)² + η²) ≠ b).
func torusHostContactCircle(host, arm geom.Torus, res Resolution) (math.Point3, float64, bool) {
	if !host.AxisDir.IsParallelTo(arm.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false // arm axis not coaxial with the host torus axis
	}
	z := host.AxisDir.AsVector()
	eta := float64(host.Center.VectorTo(arm.Center).Dot(z)) // η: arm spine axial coord above the host centre
	band := res.Weld() * (host.MajorRadius + host.MinorRadius)
	if float64(host.Center.TranslateBy(z.Scale(math.Scalar(eta))).DistanceTo(arm.Center)) > band {
		return math.Point3{}, 0, false // arm centre off the host axis — not coaxial
	}
	b := host.MinorRadius - arm.MinorRadius
	if b < res.Weld()*host.MinorRadius {
		return math.Point3{}, 0, false // spindle host (b = a−r ≤ 0)
	}
	dMajor := arm.MajorRadius - host.MajorRadius
	if stdmath.Abs(stdmath.Hypot(dMajor, eta)-b) > band {
		return math.Point3{}, 0, false // host tube not internally tangent to the ball (√((M_a−R)²+η²) ≠ b)
	}
	k := host.MinorRadius / b
	return host.Center.TranslateBy(z.Scale(math.Scalar(k * eta))), host.MajorRadius + k*dMajor, true
}

// torusArmFillet builds the exact torus arm on a CONVEX LATITUDE-cut Torus∧Plane edge — the sibling of
// sphereArmFillet / coneArmFillet for the torus host, carried in the same edgeFillet the straight-edge path
// emits. Returns a non-torusArmBuilt reason — so the caller honest-rejects via torusArmError (do-no-harm) —
// for a varying radius, a meridian cut (the spiric E5/E9 canal), or any surface decline (spindle, the cap
// clearing the offset torus, a grazing tangency).
func torusArmFillet(e *topo.Edge, host geom.Torus, pl geom.Plane, planeFace *topo.Face, p filletPick, res Resolution) (edgeFillet, torusArmReject) {
	if p.varying() {
		return edgeFillet{}, torusArmVaryingRadius
	}
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, torusArmDegenerate // degenerate plane normal — no offset direction
	}
	rhoRim := torusRimRadius(e, host)
	nDotZ := host.AxisDir.Dot(n)
	if !torusLatitudeCut(nDotZ, rhoRim, res) {
		return edgeFillet{}, torusArmMeridian // cap ∥ axis: the spiric E5/E9 canal is a follow-on slice
	}
	capH := host.Center.VectorTo(pl.Origin).Dot(host.AxisDir.AsVector())
	tor, reason := torusHostArmSurface(host, capH, rhoRim, p.r0, nDotZ, res)
	if reason != torusArmBuilt {
		return edgeFillet{}, reason
	}
	if ef, ok := curvedArmEdgeFillet(e, tor, true); ok {
		return ef, torusArmBuilt
	}
	return edgeFillet{}, torusArmDegenerate
}

// torusArmEdge dispatches a Torus∧Plane edge to the exact-arm builder (E7). handled=true means the edge WAS
// a torus∧plane edge and computeEdgeFillet must return this result — either the built torus arm or the
// cause-specific honest reject; handled=false leaves a non-torus edge to the existing curvedAdjacentError
// path unchanged (byte-identity for the cylinder/planar/sphere/cone corpus). The sibling of coneArmEdge.
func torusArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool, error) {
	host, pl, planeFace, ok := torusPlaneEdge(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	res := ResolutionForBody(body)
	ef, reason := torusArmFillet(e, host, pl, planeFace, p, res)
	if reason == torusArmBuilt {
		return ef, true, nil // exact torus arm on a convex latitude-cut Torus∧Plane rim (E7)
	}
	return edgeFillet{}, true, torusArmError(reason, e, host, p.r0) // do-no-harm, cause-specific reject
}

// torusCapNormalDot recomputes |ẑ·n̂| for the meridian reject's provenance — the measured value that failed
// the latitude gate, so the error names WHY (cap ∥ axis) with the offending number.
func torusCapNormalDot(e *topo.Edge, host geom.Torus) float64 {
	for _, f := range e.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok {
			if n, err := math.UnitVector3FromVector(outwardPlaneNormal(f, pl)); err == nil {
				return stdmath.Abs(host.AxisDir.Dot(n))
			}
		}
	}
	return 0
}

// torusArmError reports why a Torus∧Plane edge could not be rounded when torusArmFillet declined, each
// message naming the ACTUAL cause and its offending value — mirroring sphereArmError / coneArmError. The
// surface-construction rejects (spindle, clearance, grazing) split into torusHostArmSurfaceError to stay short.
func torusArmError(reason torusArmReject, e *topo.Edge, host geom.Torus, r float64) error {
	switch reason {
	case torusArmSpindle, torusArmClears, torusArmGrazing:
		return torusHostArmSurfaceError(reason, host, r)
	case torusArmMeridian:
		return fmt.Errorf("fillet: torus∧plane fillet with radius %g — cap plane is not perpendicular to the torus "+
			"axis (|ẑ·n̂|=%g, want ~1) — spiric canal, unsupported", r, torusCapNormalDot(e, host))
	case torusArmVaryingRadius:
		return fmt.Errorf("fillet: a varying-radius pick (r0≠r1) on a Torus∧Plane edge is not supported; the torus "+
			"arm requires a constant radius (got %g)", r)
	default: // torusArmDegenerate
		return fmt.Errorf("fillet: degenerate Torus∧Plane arm frame — no rolling-ball arm for radius %g (host major %g)", r, host.MajorRadius)
	}
}

// torusHostArmSurfaceError names the surface-construction rejects: a spindle (r ≥ a), the cap clearing the
// offset torus (d² ≤ 0), or a grazing cap (√(d²) below band) — each carrying the offending minor radius.
func torusHostArmSurfaceError(reason torusArmReject, host geom.Torus, r float64) error {
	switch reason {
	case torusArmSpindle:
		return fmt.Errorf("fillet: cannot round this Torus∧Plane edge — spindle: radius %g ≥ host minor a=%g "+
			"(b = a−r ≤ 0, the offset tube reaches its own centre)", r, host.MinorRadius)
	case torusArmClears:
		return fmt.Errorf("fillet: cannot round this Torus∧Plane edge with radius %g — the cap clears the offset "+
			"torus (d² = b²−η² ≤ 0): the ball cannot touch both surfaces (host minor a=%g)", r, host.MinorRadius)
	default: // torusArmGrazing
		return fmt.Errorf("fillet: edge between the host torus (a=%g) and a grazing cap plane is smooth (the spine "+
			"circle collapsed onto the tube) — no corner to round with radius %g", host.MinorRadius, r)
	}
}
