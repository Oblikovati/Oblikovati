// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General far-runout engine (FR1 skeleton) — far-runout-engine-architecture.md ADR-1..4.
//
// A constant-radius fillet arm terminates at a FAR VERTEX F where a third, CAPPING face f₃ closes the
// body. The arm's far boundary is the runout trim = armSurface ∩ f₃. A perpendicular cross-section arc
// (farCrossSectionArc) is only the special case f₃ ⊥ the spine (B3's box caps). This engine is the
// single routing layer that dispatches the two regimes per arm, per far vertex:
//   - PERPENDICULAR (fast-path): f₃ is a plane ⊥ the spine → the EXISTING farCrossSectionArc, called
//     with the exact current arguments — byte-identity by call-graph (ADR-2), not by tolerance.
//   - OBLIQUE (the port): everything else in scope → intersectArmCapping (FR2 fills it; FR1 stubs it,
//     so an oblique far vertex declines to the do-no-harm floor exactly as today).
//
// FR1 lands the skeleton + dispatch + scope guard + the intersectArmCapping port interface. It is
// standalone (armRailBundle is NOT rewired until FR3), so the corpus is trivially byte-identical.

// runoutRegime classifies one arm's far termination at its far vertex.
type runoutRegime int

const (
	// runoutUnclassified is the ZERO value: a declined/empty armRunout{} carries this, so it can never be
	// mis-read as a perpendicular runout (iota 0 would otherwise alias runoutPerpendicular). FR3's host
	// router keys on regime, so the empty value must be its own state, not the fast-path.
	runoutUnclassified  runoutRegime = iota
	runoutPerpendicular              // plane f₃, |n̂_cap·t̂_spine| > 1−sinFloor → farCrossSectionArc
	runoutOblique                    // trihedral f₃ oblique/curved → intersectArmCapping (FR2)
)

// armRunout is one arm's far termination as a single SHARED data object: the runout trim (armSurface ∩
// f₃, oriented feet[0]→feet[1] — THE edge both the arm face and the capping bite consume), the two feet
// (the host rails' outer ends), the capping face the trim bites, and the classified regime.
type armRunout struct {
	trim    endSeg
	feet    [2]math.Point3
	capping *topo.Face
	regime  runoutRegime
}

// armFarRunout terminates one arm at its far vertex F: it identifies the capping face (the scope guard),
// classifies the regime, and builds the trim. The PERPENDICULAR branch calls the EXISTING
// farCrossSectionArc with the exact arguments the current armRailBundle passes (ef.armSurface, w.radius,
// h0.from, h1.from) → the bytes are identical to today by construction (ADR-2); the host rails h0/h1
// pass through untouched. The OBLIQUE branch routes to the intersectArmCapping port (FR2); FR1 ships that
// port as a decline stub, so an oblique far vertex returns ok=false and floors to the do-no-harm path
// exactly as today. ok=false ⇒ the caller keeps its clean unwelded error, and the returned reason string
// carries the exact obstruction (offending vertex/edge id + counts) up to the do-no-harm floor — invariant
// #3 of the handoff contract; FR3 plumbs it. filletedEdges is the set of picked edge IDs at this corner:
// the admission gate declines if a SECOND one ends at the far vertex (fillet-fillet interference). Example:
//
//	h0, h1, run, ok, reason := armFarRunout(ef, w, h0, h1, filletedEdges, res)
//	if ok && run.regime == runoutPerpendicular { /* run.trim == farCrossSectionArc(ef.armSurface, …) */ }
func armFarRunout(ef edgeFillet, w cornerWeld, h0, h1 endSeg, filletedEdges map[uint64]bool, res Resolution) (endSeg, endSeg, armRunout, bool, string) {
	if ef.edge == nil {
		return h0, h1, armRunout{}, false, "far runout: arm edge is nil (a bare/unwired arm has no identifiable far vertex)"
	}
	far := farEndVertex(ef.edge, w.center)
	capping, ok, reason := cappingFaceAtFarVertex(far, ef, filletedEdges)
	if !ok {
		return h0, h1, armRunout{}, false, reason // out of scope: not a trihedral far vertex (setback regime)
	}
	if farRunoutIsPerpendicular(capping, ef.armSurface, h0.from) {
		return perpendicularRunout(ef.armSurface, w.radius, capping, h0, h1)
	}
	return obliqueRunout(ef, capping, h0, h1, w.radius, res)
}

// perpendicularRunout is the fast-path far termination: the radius-r cross-section arc from the EXISTING
// farCrossSectionArc, invoked with the exact arguments today's armRailBundle passes, so run.trim is
// byte-identical to the current path (ADR-2). The host rails already end on the feet (their .from ends),
// so they pass through unchanged — the perpendicular regime never moves a rail terminus.
func perpendicularRunout(arm geom.Surface, r float64, capping *topo.Face, h0, h1 endSeg) (endSeg, endSeg, armRunout, bool, string) {
	trim, ok := farCrossSectionArc(arm, r, h0.from, h1.from)
	if !ok {
		return h0, h1, armRunout{}, false, fmt.Sprintf("perpendicular runout: cross-section arc failed on feet %v→%v", h0.from, h1.from)
	}
	run := armRunout{trim: trim, feet: [2]math.Point3{h0.from, h1.from}, capping: capping, regime: runoutPerpendicular}
	return h0, h1, run, true, ""
}

// obliqueRunout is the oblique-regime far termination (FR3, ADR-4). The engine OWNS the whole termination:
// it fixes the AUTHORITATIVE feet closed-form (armRunoutFeet: armSprings ∩ capping, ordered to the arm's
// hosts ef.a/ef.b), builds the runout trim armSurface ∩ capping through those feet (intersectArmCapping,
// FR2's port), and RE-TERMINATES the two host rails so their outer ends land ON the feet (reterminateRail)
// — the three coincident identities trim.endpoints == feet == rail-outer-ends. Any decline (no springs,
// no foot, trim decline, foot off a rail) floors honestly, carrying the exact obstruction; NEVER a snapped
// curve. h0/h1 are the incoming (perpendicular-built) rails; their .from ends seed springCapFoot's root.
func obliqueRunout(ef edgeFillet, capping *topo.Face, h0, h1 endSeg, r float64, res Resolution) (endSeg, endSeg, armRunout, bool, string) {
	run := armRunout{capping: capping, regime: runoutOblique}
	tol := res.Weld() * r
	feet, ok, reason := armRunoutFeet(ef, capping.Geometry(), h0.from, h1.from, r, res)
	if !ok {
		return h0, h1, run, false, reason
	}
	section, ok := intersectArmCapping(ef, capping.Geometry(), feet, r, res)
	if !ok {
		return h0, h1, run, false, capTrimDeclineReason(ef, capping.Geometry(), feet, r, res)
	}
	h0p, ok0 := reterminateRail(h0, feet[0], tol)
	h1p, ok1 := reterminateRail(h1, feet[1], tol)
	if !ok0 || !ok1 {
		return h0, h1, run, false, fmt.Sprintf("oblique runout: host rail re-termination declined (foot %v on ef.a rail=%v, foot %v on ef.b rail=%v)", feet[0], ok0, feet[1], ok1)
	}
	run.trim = endSeg{from: feet[0], to: feet[1], curve: section, mid: section.PointAt(0.5)}
	run.feet = feet
	return h0p, h1p, run, true, ""
}

// farRunoutIsPerpendicular is the dispatch predicate (per arm, per far vertex): perpendicular iff the
// capping face is a geom.Plane AND is perpendicular to the arm's spine tangent at the far end
// (|n̂_cap·t̂_spine| > 1−sinFloor, via the existing planePerpToDir). A CURVED capping is always oblique —
// even where a cap is ⊥ the spine at a point, its section is not a radius-r circle. Reuses the scale-free
// sinFloor angular gate (ADR-0042 exemption) — no new constant. Corpus probe (ADR-2): B3's plane caps
// read |n̂·t̂|=1 to machine eps (1−=0), D5's oblique cap reads 0.5 (1−=0.5) — the two populations are
// ~15 orders apart, with sinFloor between them.
func farRunoutIsPerpendicular(capping *topo.Face, arm geom.Surface, farFoot math.Point3) bool {
	pl, ok := capping.Geometry().(geom.Plane)
	if !ok {
		return false
	}
	tan, ok := armSpineTangentAtFar(arm, farFoot)
	if !ok {
		return false
	}
	return planePerpToDir(pl, tan)
}

// armSpineTangentAtFar is t̂_spine(F): the direction the arm's ball centre moves at the far end — a
// cylinder arm's axis, or a torus arm's spine (major) circle tangent at the far foot's azimuth
// (AxisDir × radial). Dispatch input only (the perpendicularity test), never geometry. Declines when the
// far foot projects onto the torus axis (the radial — hence the tangent — is undefined there).
func armSpineTangentAtFar(arm geom.Surface, farFoot math.Point3) (math.UnitVector3, bool) {
	switch s := arm.(type) {
	case geom.Cylinder:
		return s.AxisDir, true
	case geom.Torus:
		return torusSpineTangent(s, farFoot)
	}
	return math.UnitVector3{}, false // only torus/cylinder arms carry a rolling-ball spine
}

// torusSpineTangent is the unit tangent of the torus arm's major (spine) circle at the foot's azimuth:
// AxisDir × radial, where radial is the in-plane direction from the torus centre toward the foot.
func torusSpineTangent(t geom.Torus, foot math.Point3) (math.UnitVector3, bool) {
	axis := t.AxisDir.AsVector()
	d := t.Center.VectorTo(foot)
	radial, err := math.UnitVector3FromVector(d.Sub(axis.Scale(d.Dot(axis))))
	if err != nil {
		return math.UnitVector3{}, false // foot on the torus axis: no spine tangent
	}
	tan, err := math.UnitVector3FromVector(axis.Cross(radial.AsVector()))
	if err != nil {
		return math.UnitVector3{}, false
	}
	return tan, true
}

// cappingFaceAtFarVertex is the engine's admission gate (the scope guard, architecture Q5): it returns
// the UNIQUE non-host face at the far vertex transverse to the arm edge's tangent — the capping face f₃
// closing the corner. It generalizes canalFarFace's transverseNonHostPlane by admitting a capping face
// of ANY surface type (the oblique port handles curved cappings). It DECLINES the setback regime out of
// scope for the single-ball trihedral engine, on TWO independent obstructions:
//
//   - 0 or ≥2 non-host transverse faces at F (n-valent / ≥2-cap apex);
//   - a SECOND filleted (picked) edge also ending at F (fillet-fillet interference).
//
// The second guard is NOT subsumed by the face-count guard — the earlier "valence 3 ⇔ one non-host face"
// claim was WRONG. Counterexample: two adjacent picked edges e₁=A∧B (this arm, hosts a=A,b=B) and e₂=A∧C
// both ending at F leave the non-host set = {C} (exactly one, transverse) → the face-count guard ACCEPTS,
// yet C is a live host of fillet e₂, so treating it as a plain capping face is wrong (that is the
// n-valent-setback / fillet-fillet regime, out of scope). Hence the explicit secondFilletedEdgeAt guard,
// keyed on the picked-edge-id set. Declines flow to the do-no-harm floor with a reason; NEVER a snapped runout.
func cappingFaceAtFarVertex(far *topo.Vertex, ef edgeFillet, filletedEdges map[uint64]bool) (*topo.Face, bool, string) {
	tan, ok := edgeTangentAt(ef.edge, far)
	if !ok {
		return nil, false, fmt.Sprintf("far vertex %d: degenerate arm-edge tangent — cannot classify transversality", far.ID())
	}
	if other, second := secondFilletedEdgeAt(far, ef.edge, filletedEdges); second {
		return nil, false, fmt.Sprintf("far vertex %d: a second filleted edge %d also ends here (fillet-fillet interference / setback regime, out of scope)", far.ID(), other)
	}
	found, n := uniqueNonHostTransverseFace(far, ef.a, ef.b, tan)
	if n != 1 {
		return nil, false, fmt.Sprintf("far vertex %d is not trihedral: %d non-host transverse faces (want exactly 1 capping face)", far.ID(), n)
	}
	return found, true, ""
}

// secondFilletedEdgeAt reports whether any edge at far OTHER than the arm's own edge is itself a filleted
// (picked) edge — the genuine fillet-fillet-interference regime the single-ball trihedral engine declines
// (architecture Q5). filletedEdges is the set of picked edge IDs. Returns the offending edge id (for the
// decline reason) with true, or (0,false) when the arm's edge is the only picked edge reaching far.
func secondFilletedEdgeAt(far *topo.Vertex, armEdge *topo.Edge, filletedEdges map[uint64]bool) (uint64, bool) {
	for _, e := range far.Edges() {
		if e.ID() == armEdge.ID() {
			continue
		}
		if filletedEdges[e.ID()] {
			return e.ID(), true
		}
	}
	return 0, false
}

// uniqueNonHostTransverseFace returns the single face at v that is neither host a/b and is transverse to
// the edge tangent tan (|tan·n̂| above the scale-free sinFloor, n̂ via probe.OutwardFaceNormalAt so it works for
// any surface type), together with the COUNT of such faces (the caller admits only count==1 as the
// capping face; 0 or several ⇒ not a simple trihedral runout). The plane-only transverseNonHostPlane
// sibling, generalized past planes. The count is returned so the decline reason can carry it.
func uniqueNonHostTransverseFace(v *topo.Vertex, a, b *topo.Face, tan math.UnitVector3) (*topo.Face, int) {
	var found *topo.Face
	n := 0
	for _, f := range facesAround(v) {
		if f.ID() == a.ID() || f.ID() == b.ID() {
			continue
		}
		nrm := probe.OutwardFaceNormalAt(f, v.Point())
		if stdmath.Abs(float64(tan.AsVector().Dot(nrm))) <= sinFloor {
			continue // f runs along the edge (tangent), not across it — not a capping face
		}
		found, n = f, n+1
	}
	return found, n
}

// intersectArmCapping is the far-runout PORT (architecture ADR-3): the runout trim armSurface ∩ capping
// between the two feet, exact on BOTH surfaces, analytic-on-the-arm (never a bare polyline). FR1 declared
// the seam; FR2 implements it in fillet_intersect_arm_capping.go (torus∩plane spiric, cyl∩plane ellipse),
// with the un-exercised ∩sphere/∩cone/∩cyl pairings clean-declining. The engine never sees which pairing
// built the curve.
