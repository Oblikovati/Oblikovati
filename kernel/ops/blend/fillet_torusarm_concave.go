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

// CONCAVE torus-host closed-rim arm (OCCT blend/simple J5, bfuseblend A5/A6) — the torus-host sibling
// of concaveSphereArmSurface / concaveConeArmSurface (fillet_curved_closed_rim_concave.go). On a CONCAVE
// closed circular rim where a host TORUS meets a LATITUDE cap plane (plane ⊥ torus axis), the rolling
// ball sits in the VOID, tangent to the cap plane and to the host tube. Unlike the sphere/cone duals,
// the tube-offset SIGN is not fixed by concavity: it follows which side of the tube carries material —
//   - material INSIDE the tube (J5's volcano crater, A5's quarter-ring on a plate): the void ball is
//     OUTSIDE the tube, external tangency, ball-centre distance a+r from the tube-centre circle;
//   - material OUTSIDE the tube (A6's flared funnel, T2's mushroom overhang): the void ball is INSIDE
//     the tube region, internal tangency, distance a−r.
// DRAWEXE 8.0.0 receipts (wave-report-E): J5's cove spine is the circle ρ=151.266, z=35 = R−√((a+r)²−η²)
// with η = capH + r·n̂z (its blend band decodes as exactly that torus); A6's plate hole grows to
// ρ=136.754 = R−√((a−r)²−η²) — the internal-tangency branch on the same formula.

// tubeMaterialSign reports which side of the host tube carries material at point p on the host torus
// face: +1 when the face's material-outward normal agrees with the tube-outward geometry normal
// (material inside the tube — external tangency), −1 when it opposes (material outside — internal
// tangency). ok=false on a degenerate normal.
func tubeMaterialSign(hostFace *topo.Face, host geom.Torus, p math.Point3) (float64, bool) {
	u, v := host.ParamAt(p)
	tubeOut := host.NormalAt(u, v)
	outward, ok := outwardFaceNormal(hostFace, p)
	if !ok {
		return 0, false
	}
	d := float64(outward.Dot(tubeOut))
	if stdmath.Abs(d) < 0.5 {
		return 0, false // normals near-orthogonal: p is not cleanly on the tube — no side to read
	}
	if d > 0 {
		return +1, true
	}
	return -1, true
}

// concaveTorusHostArmSurface builds the exact torus arm of a VOID-side rolling ball of radius r on the
// CONCAVE latitude-cut rim where a cap plane (axial coord capH, material-outward normal component
// nDotZ = ẑ·n̂) meets the host torus. The ball centre rides the coaxial spine circle at axial coord
// η = capH + r·nDotZ (r along +n̂ off the plane, into the void) with major M = R + ε·√(b²−η²), where
// b = a + s·r (s = tubeMaterialSign) and ε = torusBranchSign(ρ_rim, R) — the same branch rule as the
// convex arm. Returns a non-torusArmBuilt reason for a spindle (internal ball consumes the tube), a
// cap clearing the offset tube, a grazing tangency, or a degenerate frame.
//
// Example: concaveTorusHostArmSurface(torus{O:origin,axis:+ẑ,R:200,a:50}, capH:25, ρ_rim:156.7, r:10,
// nDotZ:+1, s:+1, res) → the J5 cove arm torus centre (0,0,35), axis +ẑ, major 151.2664, minor 10
// (DRAWEXE-matched).
func concaveTorusHostArmSurface(host geom.Torus, capH, rhoRim, r, nDotZ, s float64, res tol.Resolution) (geom.Torus, torusArmReject) {
	b := host.MinorRadius + s*r
	if b < armSpindleBand*res.Weld() {
		return geom.Torus{}, torusArmSpindle // internal ball r ≥ a: the void ball consumes the tube
	}
	eta := capH + r*nDotZ // void-side spine axial coord (the convex arm uses capH − r·n̂z)
	d2 := b*b - eta*eta
	if d2 <= 0 {
		return geom.Torus{}, torusArmClears // the offset cap clears the offset tube — no spine circle
	}
	dd := stdmath.Sqrt(d2)
	if dd < armSpindleBand*res.Weld() {
		return geom.Torus{}, torusArmGrazing // grazing cap: the spine circle collapsed onto the tube
	}
	majorR := host.MajorRadius + torusBranchSign(rhoRim, host.MajorRadius, res)*dd
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, torusArmDegenerate // spine circle collapsed onto the axis
	}
	z := host.AxisDir.AsVector()
	tor, err := geom.NewTorusWithRef(host.Center.TranslateBy(z.Scale(math.Scalar(eta))), z, host.Ref.AsVector(), majorR, r)
	if err != nil {
		return geom.Torus{}, torusArmDegenerate
	}
	return tor, torusArmBuilt
}

// concaveTorusRimArmEdge dispatches a CONCAVE CLOSED latitude-cut Torus∧Plane rim (J5/A5/A6) to the
// void-side torus-host arm builder, marking the edgeFillet armConcave so assembleCurvedArmBody routes it
// to the concave cove band. handled=false leaves every meridian cut (the spiric A4), open rim, and
// non-torus pairing to the existing dispatch unchanged (byte-identity); handled=true with an error is the
// cause-specific honest reject.
func concaveTorusRimArmEdge(e *topo.Edge, p filletPick, res tol.Resolution) (edgeFillet, bool, error) {
	host, pl, hostFace, planeFace, ok := torusPlaneEdgeFaces(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, true, fmt.Errorf("fillet: cannot round this concave torus rim — degenerate cap-plane normal %v (need a unit material-outward normal)", pl.Normal())
	}
	rhoRim := torusRimRadius(e, host)
	nDotZ := host.AxisDir.Dot(n)
	if !torusLatitudeCut(nDotZ, rhoRim, res) {
		return edgeFillet{}, false, nil // meridian cut (the spiric family) — not this branch's rim
	}
	s, ok := tubeMaterialSign(hostFace, host, edgeMidpoint(e))
	if !ok {
		return edgeFillet{}, true, fmt.Errorf("fillet: cannot round this concave torus rim — the rim midpoint reads no tube material side on the host (major %g, minor %g)", host.MajorRadius, host.MinorRadius)
	}
	capH := host.Center.VectorTo(pl.Origin).Dot(host.AxisDir.AsVector())
	tor, reason := concaveTorusHostArmSurface(host, capH, rhoRim, p.r0, nDotZ, s, res)
	if reason != torusArmBuilt {
		return edgeFillet{}, true, torusArmError(reason, e, host, p.r0)
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: tor, armConcave: true}, true, nil
}

// torusPlaneEdgeFaces is torusPlaneEdge extended with the host torus FACE (whose Reversed-aware outward
// normal fixes the tube material side for the concave arm). Kept separate so torusPlaneEdge's existing
// callers stay untouched.
func torusPlaneEdgeFaces(e *topo.Edge) (host geom.Torus, pl geom.Plane, hostFace, planeFace *topo.Face, ok bool) {
	host, pl, planeFace, ok = torusPlaneEdge(e)
	if !ok {
		return geom.Torus{}, geom.Plane{}, nil, nil, false
	}
	for _, f := range e.Faces() {
		if f != planeFace {
			hostFace = f
		}
	}
	return host, pl, hostFace, planeFace, true
}

// concaveTorusHostTubeContact is the cove-arm ↔ host-torus contact for the CONCAVE closed-rim band:
// the void ball, centred on the coaxial spine circle (major M at axial η off the host centre), touches
// the host tube along the coaxial circle scaled by k = a/(a+s·r) from the tube-centre circle — s read
// from the tangency certificate itself (√((M−R)²+η²) = a+r external, a−r internal). The concave dual of
// torusHostContactCircle, a SEPARATE entry so the convex E7 contact keeps its internal-tangency assert
// byte-identically. ok=false for a non-coaxial arm or a tube tangent on neither branch.
func concaveTorusHostTubeContact(host, arm geom.Torus, res tol.Resolution) (math.Point3, float64, bool) {
	if !host.AxisDir.IsParallelTo(arm.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false
	}
	z := host.AxisDir.AsVector()
	eta := float64(host.Center.VectorTo(arm.Center).Dot(z))
	band := res.Weld() * (host.MajorRadius + host.MinorRadius)
	if float64(host.Center.TranslateBy(z.Scale(math.Scalar(eta))).DistanceTo(arm.Center)) > band {
		return math.Point3{}, 0, false // arm centre off the host axis — not coaxial
	}
	dMajor := arm.MajorRadius - host.MajorRadius
	k, ok := tubeContactScale(stdmath.Hypot(dMajor, eta), host.MinorRadius, arm.MinorRadius, band)
	if !ok {
		return math.Point3{}, 0, false // ball tangent to the host tube on neither branch
	}
	return host.Center.TranslateBy(z.Scale(math.Scalar(k * eta))), host.MajorRadius + k*dMajor, true
}

// tubeContactScale resolves the contact-projection factor k = a/(a±r) from the measured spine-to-tube
// distance hyp: external tangency (hyp = a+r, void ball outside the tube) or internal (hyp = a−r, void
// ball inside). ok=false when hyp certifies neither.
func tubeContactScale(hyp, a, r, band float64) (float64, bool) {
	if stdmath.Abs(hyp-(a+r)) <= band {
		return a / (a + r), true
	}
	if stdmath.Abs(hyp-(a-r)) <= band && a-r > band {
		return a / (a - r), true
	}
	return 0, false
}
