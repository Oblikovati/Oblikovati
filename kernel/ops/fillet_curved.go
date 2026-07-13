// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
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

// curvedFilletError reports why a cylinder+plane edge cannot (yet) be rounded. A tangent edge —
// the cylinder is G1-smooth into the plane (a fillet cylinder meeting the very face it was made
// tangent to) — has NO corner to round, so it is rejected as smooth, not "unsupported". Any other
// cylinder+plane edge (a sharp arc cap, or a sharp axial cut) is a real fillet target Phase B/C
// will build; until then it errors clearly instead of producing the misleading "invalid solid" /
// "miter" the planar path emitted.
func curvedFilletError(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane) error {
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	u, _ := cyl.ParamAt(mid)
	if stdmath.Abs(cyl.NormalAt(u, 0).Dot(pl.Normal())) > 1-1e-6 {
		return fmt.Errorf("fillet: edge between a cylinder and a tangent plane is smooth (no corner to round)")
	}
	return fmt.Errorf("fillet: rounding an edge that borders a curved (cylinder) face is not yet supported")
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
