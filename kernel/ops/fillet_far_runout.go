// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
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
	runoutPerpendicular runoutRegime = iota // plane f₃, |n̂_cap·t̂_spine| > 1−sinFloor → farCrossSectionArc
	runoutOblique                           // trihedral f₃ oblique/curved → intersectArmCapping (FR2)
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
// exactly as today. ok=false ⇒ the caller keeps its clean unwelded error. Example:
//
//	h0, h1, run, ok := armFarRunout(ef, w, h0, h1, res)
//	if ok && run.regime == runoutPerpendicular { /* run.trim == farCrossSectionArc(ef.armSurface, …) */ }
func armFarRunout(ef edgeFillet, w cornerWeld, h0, h1 endSeg, res Resolution) (endSeg, endSeg, armRunout, bool) {
	if ef.edge == nil {
		return h0, h1, armRunout{}, false // a bare/unwired arm edge has no identifiable far vertex
	}
	far := farEndVertex(ef.edge, w.center)
	capping, ok := cappingFaceAtFarVertex(far, ef)
	if !ok {
		return h0, h1, armRunout{}, false // out of scope: not a trihedral far vertex (setback regime)
	}
	if farRunoutIsPerpendicular(capping, ef.armSurface, h0.from) {
		return perpendicularRunout(ef.armSurface, w.radius, capping, h0, h1)
	}
	run, ok := obliqueRunout(ef.armSurface, capping, [2]math.Point3{h0.from, h1.from}, w.radius, res)
	return h0, h1, run, ok
}

// perpendicularRunout is the fast-path far termination: the radius-r cross-section arc from the EXISTING
// farCrossSectionArc, invoked with the exact arguments today's armRailBundle passes, so run.trim is
// byte-identical to the current path (ADR-2). The host rails already end on the feet (their .from ends),
// so they pass through unchanged — the perpendicular regime never moves a rail terminus.
func perpendicularRunout(arm geom.Surface, r float64, capping *topo.Face, h0, h1 endSeg) (endSeg, endSeg, armRunout, bool) {
	trim, ok := farCrossSectionArc(arm, r, h0.from, h1.from)
	if !ok {
		return h0, h1, armRunout{}, false
	}
	run := armRunout{trim: trim, feet: [2]math.Point3{h0.from, h1.from}, capping: capping, regime: runoutPerpendicular}
	return h0, h1, run, true
}

// obliqueRunout is the oblique-regime port entry: the runout trim is armSurface ∩ capping
// (intersectArmCapping, FR2), built on the SAME feet the host rails end on (the shared-edge identity).
// FR1 ships intersectArmCapping as a decline stub, so this returns ok=false and the arm floors to the
// do-no-harm path (D5's oblique far vertex declines exactly as today) — the engine still reports the
// classified regime + capping face for the FR3 host router.
func obliqueRunout(arm geom.Surface, capping *topo.Face, feet [2]math.Point3, r float64, res Resolution) (armRunout, bool) {
	run := armRunout{capping: capping, regime: runoutOblique}
	section, ok := intersectArmCapping(arm, capping.Geometry(), feet, r, res)
	if !ok {
		return run, false // FR2 fills the port; until then oblique floors honestly
	}
	run.trim = endSeg{from: feet[0], to: feet[1], curve: section, mid: section.PointAt(0.5)}
	run.feet = feet
	return run, true
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
// of ANY surface type (the oblique port handles curved cappings). It DECLINES (ok=false) the setback
// regime out of scope for the single-ball trihedral engine: 0 or ≥2 non-host transverse faces at F. At a
// manifold vertex #faces = #edges, so "exactly one non-host face" is equivalently the TRIHEDRAL test
// (valence 3 = 1 filleted edge + 2 sharp) — a second picked/filleted edge or an n-valent apex raises the
// non-host count and declines here. Declines flow to the do-no-harm floor; NEVER a snapped runout.
func cappingFaceAtFarVertex(far *topo.Vertex, ef edgeFillet) (*topo.Face, bool) {
	tan, ok := edgeTangentAt(ef.edge, far)
	if !ok {
		return nil, false // degenerate far-edge tangent — cannot classify transversality
	}
	return uniqueNonHostTransverseFace(far, ef.a, ef.b, tan)
}

// uniqueNonHostTransverseFace returns the single face at v that is neither host a/b and is transverse to
// the edge tangent tan (|tan·n̂| above the scale-free sinFloor, n̂ via outwardFaceNormalAt so it works for
// any surface type). Exactly one such face is the capping face; zero or several → decline (v is not a
// simple trihedral runout). The plane-only transverseNonHostPlane sibling, generalized past planes.
func uniqueNonHostTransverseFace(v *topo.Vertex, a, b *topo.Face, tan math.UnitVector3) (*topo.Face, bool) {
	var found *topo.Face
	n := 0
	for _, f := range facesAround(v) {
		if f.ID() == a.ID() || f.ID() == b.ID() {
			continue
		}
		nrm := outwardFaceNormalAt(f, v.Point())
		if stdmath.Abs(float64(tan.AsVector().Dot(nrm))) <= sinFloor {
			continue // f runs along the edge (tangent), not across it — not a capping face
		}
		found, n = f, n+1
	}
	return found, n == 1
}

// intersectArmCapping is the far-runout PORT (architecture ADR-3): the runout trim armSurface ∩ capping
// between the two feet, exact on BOTH surfaces, analytic-on-the-arm (never a bare polyline). FR1 declares
// the seam and ships a decline STUB; FR2 implements the pairing table (torus∩plane spiric, cyl∩plane
// ellipse, …). The engine never sees which pairing built the curve. Returns (nil, false) until FR2.
func intersectArmCapping(arm, capping geom.Surface, feet [2]math.Point3, r float64, res Resolution) (geom.Curve3, bool) {
	return nil, false
}
