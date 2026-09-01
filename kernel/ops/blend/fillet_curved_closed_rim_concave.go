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

// CONCAVE closed-rim curved arm (OCCT blend/simple S2/S5) — the concave dual of the convex J1 band
// (fillet_curved_closed_rim.go). On a CONCAVE closed circular rim (StartVertex==EndVertex,
// ClassifyEdgeConvexity==EdgeConcave) where a host sphere/cone bump meets its cap plane, the rolling
// ball sits in the VOID and the fillet ADDS a cove — so the arm is tangent-EXTERNAL (ρ = R+r, not the
// convex R−r). concave-sphere-cone-arm-derivation.md derives the exact torus arm as the convex builder
// with BOTH offsets flipped outward. The band then welds through the J1 rim skeleton with a concave
// contact-circle branch (external tangency), a winding flip (cove fills the void, volume stays > 0),
// and an OUTWARD-growing plate retrim (the hole enlarges from the rim to the contact circle).
//
// The plane-contact radius equals the arm major; when it EXCEEDS the cap face's loop (the cove spills
// off the plate onto the box side walls — S2 major 16.246 > plate ±15, S5 major 15.716 > 15), the clean
// closed band is not watertight and this slice HONEST-REJECTS carrying the offending major vs the plate
// extent (a follow-on slice does the cove-onto-sidewall construction). See §5 of the derivation.

// concaveSphereArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the CONCAVE
// edge where the cap plane pl (material-outward unit normal outwardN) meets the host sphere sp — the
// concave dual of sphereArmSurface (concave-sphere-cone-arm-derivation.md §1). BOTH offsets flip vs the
// convex builder: the tube-radius offset R−r → ρ = R+r (offset the sphere OUTWARD) and the cap-plane
// offset +r → −r (push the plane r into the VOID along +n̂). O′ = O − h·n̂ and major = √(ρ²−h²) are
// byte-identical in form. Returns a non-sphereArmBuilt reason only for a degenerate frame or a collapsed
// major — ρ = R+r > 0 and |h| = r < ρ always hold, so the concave sphere arm is otherwise unconditional.
//
// Example: concaveSphereArmSurface(sphere{O:origin,R:13}, capPlane{z:0,n̂:+ẑ}, +ẑ, 3, res) → the S5 arm
// torus centre (0,0,3), axis ẑ, major √247 = 15.716234, minor 3.
func concaveSphereArmSurface(sp geom.Sphere, pl geom.Plane, outwardN math.UnitVector3, r float64, res tol.Resolution) (geom.Torus, sphereArmReject) {
	rho := sp.Radius + r // offset the sphere OUTWARD (convex: R − r)
	n := outwardN.AsVector()
	h := pl.Origin.VectorTo(sp.Center).Dot(n) - r // (O−P)·n̂ − r: cap plane pushed r into the VOID (convex: + r)
	spineSq := rho*rho - h*h
	if spineSq <= 0 {
		return geom.Torus{}, sphereArmClears // impossible for R+r, kept as a defensive guard
	}
	majorR := stdmath.Sqrt(spineSq)
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, sphereArmTangent // grazing: the spine circle has collapsed to a point
	}
	center := sp.Center.TranslateBy(n.Scale(-h))
	tor, err := geom.NewTorusWithRef(center, n, pl.UAxis.AsVector(), majorR, r)
	if err != nil {
		return geom.Torus{}, sphereArmTangent // degenerate torus frame
	}
	return tor, sphereArmBuilt
}

// concaveConeArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on the CONCAVE
// CAP-plane (circle) edge where the cap plane pl (material-outward unit normal outwardN) meets the host
// cone co — the concave dual of coneArmSurface (concave-sphere-cone-arm-derivation.md §2). Two flips vs
// the convex s=+1 call: the apex-shift sign s = −1 (A′ = A − (r/sinα)·â, offset cone BIGGER/outward) and
// the cap-plane offset pOff = P + r·n̂ (into the VOID). h′, major = tanα·h′, O′ and the NewTorusWithRef
// frame are byte-identical in form. Rejects a near-cylinder / near-plane cone (α bands), a cleared cap
// (h′ ≤ 0), a grazing cap (major below band), or a degenerate frame — scaled by res (ADR-0042).
//
// Example: concaveConeArmSurface(cone apex (0,0,40) â=−ẑ tanα=0.25, cap z=0 n̂=+ẑ, +ẑ, 8, res) → the S2
// arm torus centre (0,0,8), axis −ẑ, major 16.246211, minor 8.
func concaveConeArmSurface(co geom.Cone, pl geom.Plane, outwardN math.UnitVector3, r float64, res tol.Resolution) (geom.Torus, coneArmReject) {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	aband := coneAlphaBandCoef * res.Weld() / stdmath.Max(float64(co.Apex.DistanceTo(pl.Origin)), res.Weld())
	if sinA < aband {
		return geom.Torus{}, coneArmNearCylinder // apex shift r/sinα blows up — a cylinder host (M5 concave path)
	}
	if cosA < aband {
		return geom.Torus{}, coneArmNearPlane
	}
	a := co.AxisDir.AsVector()
	apexPrime := co.Apex.TranslateBy(a.Scale(-r / sinA))        // A′ = A − (r/sinα)·â (offset cone bigger)
	pOff := pl.Origin.TranslateBy(outwardN.AsVector().Scale(r)) // cap plane pushed r into the VOID
	return coneOffsetTorus(co, apexPrime, pOff, r, res)
}

// coneOffsetTorus is the shared tail of both cone arm builders (convex coneArmSurface and concave
// concaveConeArmSurface): given the offset apex A′ and the offset cap plane point pOff, it forms the
// spine height h′ = (pOff − A′)·â, the major radius tanα·h′, the spine foot O′ = A′ + h′·â, and the exact
// torus (minor r, axis â, frame co.Ref). The two callers differ ONLY in how A′ and pOff are signed, so
// this keeps the downstream construction single-sourced (no duplication). Rejects a cleared cap
// (h′ ≤ band), a grazing cap (major ≤ band), or a degenerate torus frame.
func coneOffsetTorus(co geom.Cone, apexPrime, pOff math.Point3, r float64, res tol.Resolution) (geom.Torus, coneArmReject) {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	a := co.AxisDir.AsVector()
	hPrime := float64(apexPrime.VectorTo(pOff).Dot(a)) // h′ = (pOff − A′)·â
	if hPrime < armSpindleBand*res.Weld() {
		return geom.Torus{}, coneArmClears // offset cap plane on the wrong side of A′ — no spine circle
	}
	majorR := sinA / cosA * hPrime // tanα·h′
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, coneArmGrazing
	}
	tor, err := geom.NewTorusWithRef(apexPrime.TranslateBy(a.Scale(hPrime)), a, co.Ref.AsVector(), majorR, r)
	if err != nil {
		return geom.Torus{}, coneArmDegenerate
	}
	return tor, coneArmBuilt
}

// isConcaveClosedRimArm is assembleCurvedArmBody's dispatch classifier for the CONCAVE closed-band arm:
// exactly ONE pick carrying ONE exact torus arm marked armConcave on a CONCAVE closed circular rim
// (StartVertex==EndVertex). The concave dual of isConvexClosedRimArm; every other configuration (convex
// rim, cylinder/canal arm, open runout) is false, so its path keeps flooring unchanged (do-no-harm).
func isConcaveClosedRimArm(fils []edgeFillet) bool {
	curved := curvedArmsOf(fils)
	if len(fils) != 1 || len(curved) != 1 || curved[0].edge == nil || !curved[0].armConcave {
		return false
	}
	ef := curved[0]
	if _, ok := ef.armSurface.(geom.Torus); !ok {
		return false // only an exact torus arm forms a closed torus band
	}
	if ef.edge.StartVertex().ID() != ef.edge.EndVertex().ID() {
		return false // an OPEN rim is a runout, not a closed band
	}
	return ClassifyEdgeConvexity(ef.edge) == EdgeConcave
}

// concaveCurvedRimArmEdge dispatches a CONCAVE CLOSED circular Sphere∧Plane / Cone∧Plane cap rim (S5/S2)
// — and, since the wave-E slice, a latitude-cut Torus∧Plane rim (J5/A5/A6, concaveTorusRimArmEdge) —
// to the concave (external-tangency) torus-arm builder, marking the edgeFillet armConcave so
// assembleCurvedArmBody routes it to the concave cove band. handled=true means this branch OWNED the
// edge (computeEdgeFillet returns its result); handled=false leaves every convex rim, open runout,
// non-circular edge, and cylinder host to the existing convex sphere/cone/cylinder dispatch unchanged
// (byte-identity). Gate on the DIHEDRAL (EdgeConcave), never the host material-side (which reads S2/S5 as
// convex — concave-sphere-cone-arm-derivation.md §"load-bearing sign fact").
func concaveCurvedRimArmEdge(body *topo.Body, e *topo.Edge, p filletPick, concave ConcaveFill) (edgeFillet, bool, error) {
	if p.varying() || concave != FillConcaveOutward || !isClosedCircleEdge(e) || ClassifyEdgeConvexity(e) != EdgeConcave {
		return edgeFillet{}, false, nil
	}
	res := tol.ForBody(body)
	if sp, pl, _, planeFace, ok := spherePlaneEdge(e); ok {
		return buildConcaveRimArm(e, pl, planeFace, func(n math.UnitVector3) (geom.Torus, bool, error) {
			tor, reason := concaveSphereArmSurface(sp, pl, n, p.r0, res)
			return tor, reason == sphereArmBuilt, sphereArmError(reason, e, sp, nil, p.r0)
		})
	}
	if co, pl, _, planeFace, ok := conePlaneEdge(e); ok {
		return buildConcaveRimArm(e, pl, planeFace, func(n math.UnitVector3) (geom.Torus, bool, error) {
			tor, reason := concaveConeArmSurface(co, pl, n, p.r0, res)
			return tor, reason == coneArmBuilt, coneArmError(reason, co, p.r0)
		})
	}
	// J5/A5/A6: a latitude-cut torus-host rim takes the void-side tube arm (external or internal
	// tangency by material side); a meridian cut falls through to the spiric dispatch unchanged.
	return concaveTorusRimArmEdge(e, p, res)
}

// buildConcaveRimArm resolves the cap plane's material-outward normal and runs the caller's concave arm
// constructor, packing a built torus into an armConcave edgeFillet (handled, no error) or returning the
// constructor's honest reject (handled, error) — the do-no-harm relay shared by the sphere and cone cap.
func buildConcaveRimArm(e *topo.Edge, pl geom.Plane, planeFace *topo.Face, build func(math.UnitVector3) (geom.Torus, bool, error)) (edgeFillet, bool, error) {
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, true, fmt.Errorf("fillet: cannot round this concave cap rim — degenerate cap-plane normal %v (need a unit material-outward normal)", pl.Normal())
	}
	tor, ok, rejectErr := build(n)
	if !ok {
		return edgeFillet{}, true, rejectErr
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: tor, armConcave: true}, true, nil
}

// isClosedCircleEdge reports whether e is a single closed circular edge (StartVertex==EndVertex, with a
// circular curve — a full geom.Circle or a closed geom.Arc3d, the form a STEP rim import carries) — the
// closed-rim shape both the convex J1 band and the concave S2/S5 cove band require.
func isClosedCircleEdge(e *topo.Edge) bool {
	if e.StartVertex().ID() != e.EndVertex().ID() {
		return false
	}
	switch e.Geometry().(type) {
	case geom.Circle, geom.Arc3d:
		return true
	default:
		return false
	}
}

// concaveClosedRimBandBody welds the CONCAVE closed rim into a full torus cove band, or HONEST-REJECTS
// (do-no-harm) when the plane-contact rail spills off the cap face. An empty reason means the returned
// body is the weld; a non-empty reason names the obstruction (with the offending value) and the body is
// nil, so the caller keeps the clean floor (never a partial/non-watertight body).
func concaveClosedRimBandBody(body *topo.Body, ef edgeFillet, res tol.Resolution) (*topo.Body, string) {
	rf, reason := solveConcaveClosedRimBand(ef, res)
	if reason != "" {
		return nil, reason
	}
	b, err := rebuildWithConcaveRimFillet(body, rf)
	if err != nil {
		return nil, fmt.Sprintf("concave closed-rim band rebuild declined: %v", err)
	}
	return b, ""
}
