// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Solving the CORNERS: what happens where two or more blended edges meet at a vertex (split out
// of fillet.go for #2217).
//
// Each vertex touched by the picks becomes either a blend (a spherical or arm corner) or a miter,
// decided from the radii and the edge count at that vertex. The radius helpers here exist because
// a corner is only well posed when the edges meeting at it agree on one radius.

// computeCorners finds the shared corners of the filleted edge set and solves a corner
// treatment for each, keyed by corner vertex id:
//
//   - three filleted edges at a trihedral (3-face) vertex → a spherical corner patch (blend);
//   - two filleted edges that share a face, the third edge staying sharp → a miter seam where
//     the two rolling-ball cylinders mutually trim (miter).
//
// Edges meeting at a corner may carry DIFFERENT radii: an equal-radius corner solves the sphere
// blend / mirror-plane miter seam (solveCorner's uniform path), a mixed-radius corner (P9/V9 miter,
// A4 trihedral) routes to solveAsymmetricCorner. A variable edge's faceted end chords still cannot
// meet a corner watertight, so a varying pick at a mixed corner is rejected there.
func computeCorners(body *topo.Body, picks []filletPick) (map[uint64]*cornerBlend, map[uint64]*cornerMiter, error) {
	groups := map[uint64][]filletPick{}
	for _, p := range picks {
		groups[p.edge.StartVertex().ID()] = append(groups[p.edge.StartVertex().ID()], p)
		groups[p.edge.EndVertex().ID()] = append(groups[p.edge.EndVertex().ID()], p)
	}
	blends := map[uint64]*cornerBlend{}
	miters := map[uint64]*cornerMiter{}
	for vid, ps := range groups {
		if len(ps) < 2 {
			continue
		}
		cb, cm, err := solveCorner(body, vid, ps)
		if err != nil {
			return nil, nil, err
		}
		if cb != nil {
			blends[vid] = cb
		}
		if cm != nil {
			miters[vid] = cm
		}
	}
	return blends, miters, nil
}

// solveCorner solves the corner treatment at vertex vid where the picks ps meet: a sphere blend
// (3 edges, trihedral vertex) or a miter seam (2 edges sharing a face), at the corner's one shared
// radius. Exactly one of (blend, miter) is returned; any other configuration errors.
func solveCorner(body *topo.Body, vid uint64, ps []filletPick) (*cornerBlend, *cornerMiter, error) {
	if !uniformCornerRadii(vid, ps) {
		// Edges meeting at a shared corner with DIFFERENT radii (P9/V9: a box top corner, r1
		// on one arm, r0.5 on the other; A4: a trihedral corner, r10/r5/r5): the equal-radius
		// sphere/mirror-seam no longer applies — each arm keeps its own rolling-ball radius and
		// the corner reconciles them (the true cyl∩cyl seam for a 2-edge miter, a torus patch for
		// a trihedral). Routed off the byte-identical equal-radius path so that path is untouched.
		return solveAsymmetricCorner(body, vid, ps)
	}
	r, err := cornerRadius(vid, ps)
	if err != nil {
		return nil, nil, err
	}
	v := vertexByID(edgesOf(ps), vid)
	faces := facesAtVertex(v)
	switch {
	case len(ps) == 3 && len(faces) == 3:
		cb, err := solveBlend(body, v, faces, r)
		return cb, nil, err
	case len(ps) == 2:
		return solveTwoEdgeCorner(v, ps, r)
	case len(ps) >= 3 && len(ps) < len(faces):
		// Partial corner (D3/E6/E7/E8): 3+ edges filleted at a higher-than-trihedral-valence vertex,
		// with at least one edge left sharp. Never reached by the ordinary ps==2 or ps==3&&faces==3
		// cases above (disjoint by construction), so those stay byte-identical. Scoped to ps>=3 ONLY
		// — an earlier attempt to also intercept ps==2 (D5) here broke simple/V3's legitimate 2-edge
		// shared-face miter at its own 5-valent vertex (TestClassifyEndCornersExcludesKGreaterThanOne):
		// the ordinary miter's cyl∩cyl seam is LOCAL to the two picked edges and their 3 relevant
		// faces (shared + 2 outers) and does not, in general, need the vertex's total valence — so
		// D5's specific invalidity is a genuine seam defect on ITS geometry, not a valence-3
		// assumption, and is NOT this wave's to force through this mechanism. See
		// fillet_corner_partial.go for the derivation of why the ps>=3 case IS safe to route here.
		return solvePartialCorner(body, v, faces, ps, r)
	case len(ps) >= 4 && len(faces) == len(ps):
		// Full-round K-arm corner (X8/A1): every edge at a K-valent planar vertex filleted at one
		// radius, closed by the exact common-tangent-sphere K-gon (fillet_corner_fullround.go).
		cb, err := solveFullRoundCorner(body, v, faces, ps, r)
		return cb, nil, err
	default:
		return nil, nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex is not a supported blend (need 3 edges at a trihedral vertex, or 2 edges sharing a face)", len(ps), len(faces))
	}
}

// solveTwoEdgeCorner dispatches the 2-edge corner: a CLOSED rim takes no corner treatment, a
// variable-radius edge cannot miter, and the miter itself is either the ordinary shared-face seam
// or — when the two edges share NO face (D4: opposite pyramid arms) — the vertex-only seam whose
// ends lie on the corner's surviving sharp edges instead.
func solveTwoEdgeCorner(v *topo.Vertex, ps []filletPick, r float64) (*cornerBlend, *cornerMiter, error) {
	if closedRimPick(ps) {
		return nil, nil, nil // a CLOSED rim (one edge counted twice: StartVertex==EndVertex) is not a
		// 2-edge miter corner — it has no second edge and no shared face, so it takes no corner treatment
		// and reaches the closed-band arm assembly (fillet_curved_closed_rim.go) instead of solveMiter (J1).
	}
	if p := varyingPick(ps); p != nil {
		return nil, nil, fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a 2-edge miter corner (its cone has no seam with a cylinder); round the third edge for a setback instead", p.r0, p.r1)
	}
	if sharedFace(ps[0].edge, ps[1].edge) == nil {
		// D4: two filleted edges meeting ONLY at the vertex (opposite pyramid arms). The rolling-
		// ball cylinders still mutually trim — the seam is cylA∩cylB with both ends on the corner's
		// sharp edges (fillet_miter_vertexonly.go), matching DRAWEXE's D4 band boundary exactly.
		cm, err := solveVertexOnlyMiter(v, ps, r)
		return nil, cm, err
	}
	cm, err := solveMiter(v, ps, r)
	return nil, cm, err
}

// cornerRadius returns the radius every pick carries AT the shared corner vertex vid — the radius of
// the corner sphere. A variable edge is allowed (e.g. a setback's tapered third edge) as long as its
// radius at this corner matches the others; only the far ends may differ. Reached only on the
// equal-radius path (solveCorner routes a mixed-radius corner to solveAsymmetricCorner first), so the
// mismatch error is now a defensive backstop rather than the primary guard.
func cornerRadius(vid uint64, ps []filletPick) (float64, error) {
	r := radiusAtVertex(ps[0], vid)
	for _, p := range ps {
		if rv := radiusAtVertex(p, vid); rv != r {
			return 0, fmt.Errorf("fillet: edges meeting at a shared corner must use one radius there (got %g and %g)", r, rv)
		}
	}
	return r, nil
}

// uniformCornerRadii reports whether every pick carries the SAME radius at the shared corner vid. The
// equal-radius corner treatments (the sphere blend, the mirror-plane miter seam) require it; a corner
// whose arms differ (P9/V9, A4) takes the asymmetric path. This is exactly cornerRadius's old guard
// condition, split out so the equal-radius path stays byte-identical to before the asymmetric work.
func uniformCornerRadii(vid uint64, ps []filletPick) bool {
	r := radiusAtVertex(ps[0], vid)
	for _, p := range ps {
		if radiusAtVertex(p, vid) != r {
			return false
		}
	}
	return true
}

// closedRimPick reports whether the two picks grouped at a corner vertex are the SAME closed edge
// counted twice — a full-circle rim whose StartVertex==EndVertex lands its single pick in the
// seam-vertex group at both endpoints. That is NOT a 2-edge miter (no second edge, no shared face),
// so solveCorner returns no corner treatment and the rim reaches the closed-band arm assembly (J1).
func closedRimPick(ps []filletPick) bool {
	return len(ps) == 2 && ps[0].edge.ID() == ps[1].edge.ID() &&
		ps[0].edge.StartVertex().ID() == ps[0].edge.EndVertex().ID()
}

// varyingPick returns the first pick whose radius varies along the edge, or nil if all are constant.
func varyingPick(ps []filletPick) *filletPick {
	for i := range ps {
		if ps[i].varying() {
			return &ps[i]
		}
	}
	return nil
}

// radiusAtVertex returns the pick's radius at the endpoint vid (r0 at the start vertex, else r1).
func radiusAtVertex(p filletPick, vid uint64) float64 {
	if p.edge.StartVertex().ID() == vid {
		return p.r0
	}
	return p.r1
}

// edgesOf projects the picks' edges.
func edgesOf(ps []filletPick) []*topo.Edge {
	out := make([]*topo.Edge, len(ps))
	for i, p := range ps {
		out[i] = p.edge
	}
	return out
}

// solveBlend builds the corner sphere at the trihedral vertex v. An all-planar corner solves the
// sphere equidistant (r) from the three planes (the historical path, byte-identical below). A corner
// whose host set is ONE cylinder and two planes (M5 Slice A: a curved-rim boss corner) instead solves
// the ball tangent to the cylinder and the two planes — an analytic geom.Sphere of the same radius r,
// matching OCCT's equal-radius corner KPart (BREP surface code 4). A ONE sphere + two planes host set
// solves the same analytic corner via sphereHostCorner (SP2). Any other host mix (two curved faces, a
// cone/torus host, or ≥2 curved faces) still returns "corner face must be planar" (do-no-harm).
func solveBlend(body *topo.Body, v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	if cyl, planes, ok := cylinderHostCorner(faces); ok {
		return solveCurvedBlend(body, v, faces, cyl, planes, r) // curved rim: analytic sphere corner
	}
	// SP2: a sphere host + two planes solves the analytic sphere corner too (sphere-host campaign).
	// Ordered AFTER the untouched cylinderHostCorner and before solvePlanarBlend, so the cylinder (M5)
	// and all-planar paths stay byte-identical (unreachable-by-construction for non-sphere hosts).
	if sph, sphereFace, planes, ok := sphereHostCorner(faces); ok {
		return solveSphereBlend(v, faces, sph, sphereFace, planes, r)
	}
	// CN3: a cone host + two planes solves the analytic sphere corner too (cone-host campaign). Ordered
	// AFTER sphereHostCorner and before solvePlanarBlend, so every non-cone corner stays byte-identical.
	if co, coneFace, planes, ok := coneHostCorner(faces); ok {
		return solveConeBlend(v, faces, co, coneFace, planes, r)
	}
	// R4: TWO parallel-axis cylinder hosts + one plane perpendicular to the shared axis (simple/O9 P7,
	// the cyl∧cyl corner unblocked once the cylinder∧cylinder seam band landed) reduces to the same
	// analytic sphere corner via a 2D circle∩circle closed form (fillet_twocyl_corner.go). Ordered AFTER
	// every 1-curved-host recognizer (each requires exactly 1 cylinder, so a 2-cylinder corner is
	// unreachable there) and before solvePlanarBlend, so every other corner stays byte-identical.
	if cylFaces, planeFace, ok := twoParallelCylinderHostCorner(faces); ok {
		return solveTwoCylinderBlend(v, cylFaces, planeFace, r)
	}
	// R4: a TORUS host + two planes (simple/E6 E8 F1 F3) solves the same analytic sphere corner via
	// the line-vs-offset-torus tangency quartic (fillet_torus_corner.go). Ordered AFTER every other
	// recognizer (none of which match a torus face) and before solvePlanarBlend, so every other
	// corner stays byte-identical.
	if tor, torusFace, planes, ok := torusHostCorner(faces); ok {
		return solveTorusBlend(v, faces, tor, torusFace, planes, r)
	}
	return solvePlanarBlend(v, faces, r)
}

// solvePlanarBlend builds the corner sphere from the three planar faces meeting at v (the point
// at distance r from all three, inside) and its tangent points on each.
func solvePlanarBlend(v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	var a [3][3]float64
	var b [3]float64
	for i, f := range faces {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return nil, fmt.Errorf("fillet: corner face must be planar")
		}
		// Material-OUTWARD normal (respects face.Reversed()): the centre sits r INSIDE each face,
		// n·s = n·origin − r, which only holds when n points outward. A raw plane normal on a
		// reversed (imported) face solves for a sphere on the wrong side (same defect as the miter).
		n := outwardPlaneNormal(f, pl)
		a[i] = [3]float64{n.X, n.Y, n.Z}
		b[i] = n.Dot(pl.Origin.AsVector()) - r // distance r on the inside of each face
	}
	x, ok := probe.Solve3(a, b)
	if !ok {
		return nil, fmt.Errorf("fillet: cannot solve corner blend sphere (degenerate faces)")
	}
	s := math.P3(x[0], x[1], x[2])
	sph, err := geom.NewSphere(s, r)
	if err != nil {
		return nil, err
	}
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		// Tangent point is the sphere centre pushed r along the OUTWARD normal to reach the face.
		tan[f.ID()] = s.TranslateBy(outwardPlaneNormal(f, f.Geometry().(geom.Plane)).Scale(r))
	}
	return &cornerBlend{vertex: v, center: s, sphere: sph, tan: tan}, nil
}

// vertexByID returns the vertex with id vid from the edge set.
func vertexByID(edges []*topo.Edge, vid uint64) *topo.Vertex {
	for _, e := range edges {
		if e.StartVertex().ID() == vid {
			return e.StartVertex()
		}
		if e.EndVertex().ID() == vid {
			return e.EndVertex()
		}
	}
	return nil
}

// facesAtVertex returns the distinct faces meeting at v.
func facesAtVertex(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var out []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				out = append(out, f)
			}
		}
	}
	return out
}
