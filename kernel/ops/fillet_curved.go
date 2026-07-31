// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-adjacent fillets — rounding an edge that borders a CYLINDER face (the surface a prior
// fillet created) rather than two planes. The plane-plane rolling ball is a cylinder; against a
// cylinder neighbour it is a cylinder (a straight, axis-parallel edge) or a torus (an arc edge
// around the cylinder axis). Phase A classifies the edge and reports precisely what is and is not
// yet handled; Phase B builds the torus. See computeEdgeFillet for the dispatch.

// cylinderPlaneEdge reports an edge bounded by one cylinder face and one plane face, returning
// both surfaces. This is the "fillet of a fillet" input — the prior fillet left the cylinder.
func cylinderPlaneEdge(e *topo.Edge) (cyl geom.Cylinder, pl geom.Plane, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Cylinder{}, geom.Plane{}, false
	}
	for i := 0; i < 2; i++ {
		c, okc := faces[i].Geometry().(geom.Cylinder)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okc && okp {
			return c, p, true
		}
	}
	return geom.Cylinder{}, geom.Plane{}, false
}

// curvedTangencyBand is k in the surface-tangency band ε = k·res.Weld()/R (ADR-0042): the
// dot-product slack |n̂_C·n̂_P| > 1−ε below which the cylinder is treated as G1-tangent into the
// plane. It is a MODEL-relative angular band (weld resolution over the arc radius), never a bare
// 1e-6 — the same normal slack is negligible on a small cylinder and a visible mis-call on a huge
// one. k=1 keeps the band as tight as the derivation allows for a smooth/sharp discrimination.
const curvedTangencyBand = 1

// curvedFilletError reports why a cylinder+plane edge cannot (yet) be rounded. A tangent edge —
// the cylinder is G1-smooth into the plane (a fillet cylinder meeting the very face it was made
// tangent to) — has NO corner to round, so it is rejected as smooth, not "unsupported". Any other
// cylinder+plane edge that the arm builder declined (concave rim, oblique ellipse edge, or a
// degenerate spindle/clearance) errors clearly instead of the misleading "invalid solid" / "miter"
// the planar path emitted. res scales the tangency band to the model (ADR-0042).
func curvedFilletError(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, res Resolution) error {
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	u, _ := cyl.ParamAt(mid)
	eps := curvedTangencyBand * res.Weld() / cyl.Radius
	if stdmath.Abs(cyl.NormalAt(u, 0).Dot(pl.Normal())) > 1-eps {
		return fmt.Errorf("fillet: edge between a cylinder and a tangent plane is smooth (no corner to round)")
	}
	return fmt.Errorf("fillet: rounding an edge that borders a curved (cylinder) face is not yet supported")
}

// cylinderArmEdge dispatches a Cylinder∧Plane edge to the M5 Slice A arm builder. handled=true means
// the edge WAS a cylinder∧plane edge and computeEdgeFillet must return this result — the built arm or
// the honest reject; handled=false leaves the edge to the sphere/planar dispatch unchanged. Split out
// (the sibling of sphereArmEdge) to keep computeEdgeFillet within funlen; behavior is identical to the
// former inline branch.
func cylinderArmEdge(body *topo.Body, e *topo.Edge, p filletPick, concave ConcaveFill) (edgeFillet, bool, error) {
	// Cluster-B additive arm ABOVE the curvedAdjacentError decline (the agreed wave seam): a CLOSED
	// Cylinder∧Cylinder SSI-seam loop builds its exact-station canal band (fillet_cylcyl_seam.go)
	// instead of flat-refusing. Parallel-axis pairs are excluded inside the classifier, so the
	// equal-parallel miter arm (P5, cylCylMiterArmEdge) is never shadowed; every decline falls
	// through to the byte-identical refusal.
	if ef, handled := cylCylSeamArmEdge(body, e, p); handled {
		return ef, true, nil
	}
	cyl, pl, ok := cylinderPlaneEdge(e)
	if !ok {
		return edgeFillet{}, false, nil
	}
	res := ResolutionForBody(body)
	if ef, built := curvedArmFillet(e, cyl, pl, p, res); built {
		return ef, true, nil // exact torus/cylinder arm on a convex axis-aligned rim
	}
	if ef, built, err := concaveCurvedArmFillet(body, e, cyl, pl, p, res, concave); built || err != nil {
		return ef, true, err // N3/M4/N9 arm, or the honest spindle/clearance reject (r, R in the message)
	}
	return edgeFillet{}, true, curvedFilletError(e, cyl, pl, res) // torus/oblique concave / decline — do-no-harm
}

// curvedArmFillet builds the exact rolling-ball arm on a CONVEX axis-aligned Plane∧Cylinder edge
// (M5 Slice A, m5-curved-arm-derivation.md): an exact torus (axis ⊥ plane, circle edge) or an exact
// cylinder (axis ∥ plane, line edge), carried in the same edgeFillet the straight-edge path emits.
// Returns false — so the caller honest-rejects via curvedFilletError (do-no-harm) — for a CONCAVE
// (root) rim, a varying radius, config iii (oblique ellipse edge, Slice B), or any constructor
// decline (spindle/clearance). The arm constructors are convex-external only (Task 2 caveat), so
// this owns the material-side gate: an R−r torus on a concave rim is never emitted.
func curvedArmFillet(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, p filletPick, res Resolution) (edgeFillet, bool) {
	if p.varying() || ClassifyEdgeConvexity(e) != EdgeConvex {
		return edgeFillet{}, false // constant-radius convex-external only
	}
	outwardN, ok := planeHostNormal(e, pl)
	if !ok {
		return edgeFillet{}, false // no readable plane host normal — cannot orient the arm into the material
	}
	eps := convexArmWallSign(e, cyl) // +1 boss (R−r) / −1 bore/notch (R+r) — corner-blend-weld foundation
	switch classifyCurvedArm(cyl, pl, res) {
	case armTorus:
		tor, ok := torusArmSurface(cyl, pl, outwardN, p.r0, eps, res)
		return curvedArmEdgeFillet(e, tor, ok)
	case armCylinder:
		arm, ok := cylinderArmSurface(e, cyl, pl, outwardN, p.r0, eps)
		return curvedArmEdgeFillet(e, arm, ok)
	default:
		return edgeFillet{}, false // armRejected: oblique ellipse edge (config iii, Slice B)
	}
}

// convexArmWallSign is ε ∈ {+1,−1} for a CONVEX Cylinder∧Plane arm: the sense that selects the arm
// surface's offset radius R−ε·r. ε=+1 for a BOSS wall (material inside the cylinder — the historical
// case, R−r) and ε=−1 for a BORE/NOTCH wall (material OUTSIDE the cylinder, R+r; corner-blend-weld
// foundation — where OCCT places the corner at R+r but the pre-foundation code hard-coded R−r and
// mirrored the corner into the void). It reuses cylinderHostRadialSign (the concave-arm engine's exact
// ε=n_C·r̂ read) and DEFAULTS to +1 when the sign is unreadable (an on-axis edge), so a boss stays
// byte-identical to the prior code — a bore always has a well-defined radial normal, so the default
// never suppresses a real notch.
func convexArmWallSign(e *topo.Edge, cyl geom.Cylinder) float64 {
	if eps, ok := cylinderHostRadialSign(e, cyl); ok {
		return eps
	}
	return 1
}

// planeHostNormal is the material-outward unit normal of the planar host face of a Cylinder∧Plane edge —
// the plane's normal negated when that face is Reversed (outwardPlaneNormal). The M5 arm builders must
// offset the rolling ball into the MATERIAL, so they take this orientation-robust normal rather than the
// raw geom normal, which on an imported face can point either way: B3's cap normal happens to point out of
// the material, but B6's radial-cut normal points INTO it, which put the arm in the void (curved-runout R1).
// ok=false when the edge borders no plane face (a defensive guard; cylinderPlaneEdge already found one).
func planeHostNormal(e *topo.Edge, pl geom.Plane) (math.UnitVector3, bool) {
	for _, f := range e.Faces() {
		if p, isPlane := f.Geometry().(geom.Plane); isPlane && p == pl {
			n, err := math.UnitVector3FromVector(outwardPlaneNormal(f, pl))
			return n, err == nil
		}
	}
	return math.UnitVector3{}, false
}

// curvedArmEdgeFillet packs a built arm surface into an edgeFillet carrying the edge's two host
// faces, or returns false when the constructor declined (built == false) — the do-no-harm relay.
func curvedArmEdgeFillet(e *topo.Edge, arm geom.Surface, built bool) (edgeFillet, bool) {
	if !built {
		return edgeFillet{}, false
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: arm}, true
}

// curvedAdjacentError rejects an edge bordering a curved (non-planar) face that the cylinder+plane
// classifier does not cover — the miter SEAM between two edge fillets (cylinder∩cylinder), or a
// torus/sphere neighbour a prior round left. The rolling-ball blend needs two PLANAR walls; these
// curved∩curved (and curved∩*) contacts are a fillet-over-fillet the general blend does not yet
// build. Rejecting here with the offending surface named — BEFORE the model layer facets the whole
// body — replaces the misleading "not a valid solid" the triangle-cage path produced (scenario 07).
// Returns nil when both faces are planar (the ordinary edge fillet the caller then solves).
func curvedAdjacentError(e *topo.Edge) error {
	for _, f := range e.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return fmt.Errorf("fillet: cannot round an edge bordering a curved (%s) face — rounding a filleted or otherwise curved edge is not yet supported", surfaceKind(f.Geometry()))
		}
	}
	return nil
}

// runsIntoExistingRound reports the pre-existing curved round face an edge's ENDPOINT runs into, or
// nil. A planar-flanked edge (curvedAdjacentError already filtered curved WALLS) whose end touches a
// curved face that is NOT one of its own two walls and NOT one of this op's own in-progress rounds is
// the fillet-meets-fillet corner rampam hit (#1797): fillet a cube's top rim, then its verticals —
// each vertical is plane∩plane but its top vertex touches the top-rim cylinders.
//
// This USED to reject up front. It no longer does (build-then-certify): the planar corner machinery
// closes many such junctions into a valid solid — an asymmetric-radius round trims cleanly — so we
// BUILD the corner and let Validate certify it, greening the ~14 corner-into-round corpus cases the
// blanket guard wrongly rejected. Only the still-uncloseable symmetric equal-radius case fails
// Validate; filletResolvedEdges then calls this to NAME the actionable cause instead of the misleading
// "not a valid solid" that once shipped a facet-cage octagon. picked holds this op's picks, so a round
// bordering a pick is this op's own corner (solved normally), not a prior round, and is ignored.
func runsIntoExistingRound(e *topo.Edge, picked map[uint64]bool) *topo.Face {
	own := map[uint64]bool{}
	for _, f := range e.Faces() {
		own[f.ID()] = true
	}
	for _, v := range e.Vertices() {
		for _, f := range facesAtVertex(v) {
			if own[f.ID()] {
				continue // one of the edge's own two (planar) walls
			}
			if _, planar := f.Geometry().(geom.Plane); planar {
				continue
			}
			if faceBordersAnyPick(f, picked) {
				continue // this op's own adjacent-edge round, not a pre-existing one
			}
			return f
		}
	}
	return nil
}

// firstCornerIntoRound returns the first picked edge that runs into a pre-existing round and that
// round (nil,nil if none) — used to shape a build-then-certify failure into the actionable #1797
// message rather than the generic invalid-solid one.
func firstCornerIntoRound(edges []filletPick) (*topo.Edge, *topo.Face) {
	picked := make(map[uint64]bool, len(edges))
	for _, p := range edges {
		picked[p.edge.ID()] = true
	}
	for _, p := range edges {
		if round := runsIntoExistingRound(p.edge, picked); round != nil {
			return p.edge, round
		}
	}
	return nil, nil
}

// cornerIntoRoundError names the uncloseable pre-existing round and the fix (#1797): the honest,
// actionable rejection for the symmetric corner the planar blend still cannot close.
func cornerIntoRoundError(e *topo.Edge, round *topo.Face) error {
	return fmt.Errorf("fillet: cannot round edge %d — it runs into an existing rounded (%s) face at its end; "+
		"fillet these edges BEFORE the adjacent rounds, or select them together", e.ID(), surfaceKind(round.Geometry()))
}

// faceBordersAnyPick reports whether face f is bounded by an edge that is one of the ops's picks —
// i.e. f is a round this same op is about to build, not a pre-existing curved neighbour.
func faceBordersAnyPick(f *topo.Face, picked map[uint64]bool) bool {
	for _, e := range f.Edges() {
		if picked[e.ID()] {
			return true
		}
	}
	return false
}

// surfaceKind names a surface for an error message (its concrete geometry type), e.g. "cylinder".
func surfaceKind(s geom.Surface) string {
	switch s.(type) {
	case geom.Cylinder:
		return "cylinder"
	case geom.Cone:
		return "cone"
	case geom.Sphere:
		return "sphere"
	case geom.Torus:
		return "torus"
	default:
		return fmt.Sprintf("%T", s)
	}
}
