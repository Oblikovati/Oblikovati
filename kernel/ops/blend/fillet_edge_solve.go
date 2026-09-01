// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Solving ONE edge's blend: from a resolved pick to an edgeFillet (split out of fillet.go for
// #2217).
//
// The edge's two host faces give the frame, the corners at each end give the extents, and the
// section functional gives the shape between them. A curved host arm takes a different route into
// the same record, because its frame cannot be read off two planes.

// computeEdgeFillet solves the rolling-ball geometry for one convex straight edge, using a
// corner blend at either endpoint that is a shared corner. A varying pick gets its end arcs
// sampled as chords (shared by the ruling strips and the end faces).
func computeEdgeFillet(body *topo.Body, p filletPick, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill) (edgeFillet, error) {
	e := p.edge
	if ef, handled, err := curvedHostArmEdge(body, e, p, concave); handled {
		return ef, err // an exact cylinder/sphere/cone-host arm on a convex curved rim (or concave cyl arm), or its honest reject
	}
	if err := curvedAdjacentError(e); err != nil {
		return edgeFillet{}, err // any other curved neighbour (cyl∩cyl miter seam, torus, sphere)
	}
	// A planar edge whose END runs into a PRIOR round (#1797) is NO LONGER rejected here: the corner
	// is built and certified by Validate downstream (build-then-certify). filletResolvedEdges names the
	// #1797 cause only if that certificate fails (the still-uncloseable symmetric equal-radius corner).
	a, b, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		return edgeFillet{}, err
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return edgeFillet{}, fmt.Errorf("fillet: degenerate edge")
	}
	in, err := filletFrame(body, e, nA, nB, (p.r0+p.r1)/2, concave)
	if err != nil {
		return edgeFillet{}, err
	}
	in.a, in.b, in.axis = a, b, axis.AsVector()
	in.weld = tol.ForBody(body).Weld() // F3a: the spine-concurrence tolerance cornerAt gates the override on
	return solvedEdgeFillet(e, p, in, blends, miters)
}

// curvedHostArmEdge dispatches an edge that borders a CURVED host face (cylinder, sphere, cone, or torus)
// to the matching exact-arm builder, in the do-no-harm order cylinder → sphere → cone → torus (each fires
// only for its own host pair, so a Plane∧Plane edge and every other host mix falls through unchanged). handled=true
// means one builder OWNED the edge and computeEdgeFillet must return its result — the built arm or the
// cause-specific honest reject; handled=false leaves the edge to curvedAdjacentError / the planar path.
func curvedHostArmEdge(body *topo.Body, e *topo.Edge, p filletPick, concave ConcaveFill) (edgeFillet, bool, error) {
	if ef, handled, err := concaveCurvedRimArmEdge(body, e, p, concave); handled {
		return ef, handled, err // S2/S5: concave CLOSED sphere/cone cap rim → external-tangency cove arm (or spill reject)
	}
	if ef, handled, err := cylinderArmEdge(body, e, p, concave); handled {
		return ef, handled, err // M5 Slice A: exact cylinder/torus arm on a convex (or concave N3/M4/N9) axis-aligned rim
	}
	if ef, handled, err := sphereArmEdge(body, e, p); handled {
		return ef, handled, err // SP1: exact torus arm on a convex Sphere∧Plane rim
	}
	if ef, handled, err := coneArmEdge(body, e, p); handled {
		return ef, handled, err // CN1: exact torus arm on a convex Cone∧Plane cap (circle) edge
	}
	if ef, handled := ellipticalCylinderArmEdge(body, e, p); handled {
		return ef, true, nil // F4: exact circular-cylinder arm on a convex EllipticalCylinder∧Plane ruling edge
	}
	if ef, handled := ellipticClosedRimArmEdge(body, e, p); handled {
		return ef, true, nil // J6/J8: canal band on a CLOSED EllipticalCylinder∧Plane rim (spine = a closed non-analytic curve)
	}
	if ef, handled := cylCylMiterArmEdge(body, e, p); handled {
		return ef, true, nil // family B: exact cylinder arm on an equal-parallel Cylinder∧Cylinder miter edge (P5)
	}
	return torusArmEdge(body, e, p) // E7: exact torus arm on a convex latitude-cut Torus∧Plane rim
}

// filletFrame resolves the rolling-ball centre offset and the tangent-point normals for an edge,
// choosing the side from the edge's convexity and (for concave edges) the fill strategy:
//   - convex: the ball centre sits INSIDE the solid (offDir = −(nA+nB)/(1+nA·nB)); the corner is
//     rounded away. A centre that is not inside means the edge is not actually convex.
//   - concave + outward (default): the ball sits in the VOID so the cylinder bridges the faces and
//     FILLS the inside corner — the same offDir/normals negated to put the centre on the void side.
//   - concave + inward: the ball sits in the MATERIAL (the convex formula's side), rounding a recess
//     into the corner.
func filletFrame(body *topo.Body, e *topo.Edge, nA, nB math.Vector3, rMid float64, concave ConcaveFill) (cornerInputs, error) {
	offDir := nA.Add(nB).Scale(-1 / (1 + nA.Dot(nB))) // per-unit-radius centre offset into the solid
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	if ClassifyEdgeConvexity(e) == EdgeConcave {
		if concave == FillConcaveOutward {
			return cornerInputs{nA: nA.Scale(-1), nB: nB.Scale(-1), offDir: offDir.Scale(-1), flip: true}, nil
		}
		// Inward recess: the ball rolls in the MATERIAL (the convex-formula side). Its tangent points
		// land off the bounded faces unless they extend that way, so it is only valid on geometry that
		// permits it (e.g. a pocket). The explicit realizability gate rejects the impossible case
		// honestly — before it existed the rejection was an ACCIDENT of inconsistent loop winding, which
		// B2's orientFilletShell (fee0da5c) laundered into a Validate-passing self-intersecting solid.
		// A concave edge's natural fillet is the outward fill above.
		if p, ok := concaveInwardRealizable(body, e, nA, nB, offDir, rMid); !ok {
			return cornerInputs{}, fmt.Errorf("fillet: inward recess unrealizable at concave edge — tangent point %v is not material-backed (must lie on a bounded face with material behind and void in front)", p)
		}
		return cornerInputs{nA: nA, nB: nB, offDir: offDir}, nil
	}
	if !brep.PointInside(body, mid.TranslateBy(offDir.Scale(rMid))) {
		return cornerInputs{}, fmt.Errorf("fillet: edge is not convex (only convex edges are supported)")
	}
	return cornerInputs{nA: nA, nB: nB, offDir: offDir}, nil
}

// solvedEdgeFillet assembles the edgeFillet once the edge's frame is known: corners per end
// radius, then either the chord-sampled varying blend or the constant cylinder.
func solvedEdgeFillet(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (edgeFillet, error) {
	c0, c1, err := edgeCorners(e, p, in, blends, miters)
	if err != nil {
		return edgeFillet{}, err
	}
	if p.chordPath() {
		mids := midProfiles(e, in, p.mids, cornerChordCount(in), p.cross, p.rho)
		ef := edgeFillet{a: in.a, b: in.b, c0: c0, c1: c1, mids: mids, edge: e, varying: true, flip: in.flip}
		if w, ok := exactSectionWeight(p, in); ok && plainEnds(c0, c1) {
			// The blend is exactly a rational ruled surface between conic sections (#1606):
			// no chord sampling — corners keep their true arcs (or carry the conic weight for
			// the end trims) and the faces are emitted analytic. G2-quintic cross-sections and
			// miter/blend-terminated ends keep the strip fallback, now diagnostic-flagged.
			ef.exact, ef.secW = true, w
			setBlendShoulders(&ef, in, p)
			return ef, nil
		}
		sampleCornerChords(&ef.c0, &ef.c1, in, p.cross, p.rho)
		return ef, nil
	}
	cyl, err := geom.NewCylinder(c0.cen, in.axis, p.r0)
	if err != nil {
		return edgeFillet{}, err
	}
	// The band must END where the solid does: trim each terminal section against the wall it stops on
	// instead of squaring it off in the section plane at the edge's end vertex (fillet_farend_trim.go).
	trimBandEndsToWalls(&c0, &c1, in)
	return edgeFillet{a: in.a, b: in.b, cyl: cyl, c0: c0, c1: c1, edge: e, flip: in.flip}, nil
}

// edgeCorners solves the rounded corners at both endpoints of an edge (each blended when its
// vertex is a shared corner), with the pick's per-end radius.
func edgeCorners(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (c0, c1 corner, err error) {
	if c0, err = cornerAt(e.StartVertex(), in, p.r0, blends[e.StartVertex().ID()], miters[e.StartVertex().ID()], p.chordPath()); err != nil {
		return corner{}, corner{}, err
	}
	c1, err = cornerAt(e.EndVertex(), in, p.r1, blends[e.EndVertex().ID()], miters[e.EndVertex().ID()], p.chordPath())
	return c0, c1, err
}
