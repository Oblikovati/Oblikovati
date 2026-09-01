// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// singularDetTol is the magnitude below which a normal determinant (three planes' triple
// product) or a scalar triple product of edge vectors is treated as zero: the three planes
// share no unique meeting point, or the four tetra points are coplanar so the cut has no
// volume. It sits below the linear DefaultTolerance because it bounds a product of three
// (roughly unit) vectors, not a length.
const singularDetTol = 1e-12

// chamferEdges bevels each selected edge with a triangular wedge tool that runs along it. All
// tools are built from the original body up front (a boolean rebuilds topology with new lineage,
// so a reference key would not survive the first boolean), then applied in turn.
//
// A CONVEX edge cuts a wedge from the material (the corner bevel). A CONCAVE (internal) edge —
// where the faces fold over the material — instead either fills the inside corner with a gusset
// (strategy ChamferConcaveOutward, the default) or cuts a recessed relief groove
// (ChamferConcaveInward); see concaveChamferWedge.
//
// When flatCorners is set, every vertex where exactly three selected CONVEX edges meet also gets
// a tetrahedron cut that trims the pointy three-plane intersection into one flat triangular face
// — the default corner blend. With it clear the three chamfer planes are left to meet at a point.
// The blend honours per-face setbacks (d1, d2), so it is correct for asymmetric chamfers too.
func chamferEdges(in Input, keys [][]byte, d1, d2 float64, feat string, flatCorners bool,
	strategy types.ChamferConcaveStrategy, run chamferRun, anchors map[string]math.Point3) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if d1 <= 0 || d2 <= 0 {
		return Output{}, fmt.Errorf("chamfer: setbacks (%g, %g) must both be > 0", d1, d2)
	}
	edges, heals, err := resolveEdges(body, keys, anchors)
	if err != nil {
		return Output{}, err
	}
	// The analytic conical chamfer (#127) assumes a SYMMETRIC setback on a CONVEX cylinder rim;
	// an asymmetric chamfer or any concave edge takes the faceted-wedge path (which develops the
	// per-edge fill/relief). The flat-corner blend handles both symmetric and asymmetric setbacks.
	// A partial chamfer covers only a span of each rim, which the whole-rim analytic cone cannot
	// express, so it takes the wedge path (#1888).
	if d1 == d2 && !run.isPartial() && allConvex(edges) {
		// A rim of a simple analytic cylinder gets a TRUE conical chamfer (one geom.Cone face) by
		// rebuilding the body as a surface of revolution (#127). Anything else falls through.
		if res, ok := analyticCylinderChamfer(body, edges, d1, feat); ok {
			return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
		}
	}
	out, err := chamferByWedges(in, body, edges, d1, d2, feat, flatCorners, strategy, run)
	if err != nil {
		return Output{}, err
	}
	out.Heals = heals
	return out, nil
}

// allConvex reports whether every edge is a convex dihedral (so the analytic conical fast path,
// which assumes material is cut away, is valid for the whole selection).
func allConvex(edges []*topo.Edge) bool {
	for _, e := range edges {
		if blend.ClassifyEdgeConvexity(e) == blend.EdgeConcave {
			return false
		}
	}
	return true
}

// wedgeOp is one chamfer tool body paired with the boolean that applies it: convex/relief wedges
// Cut, an outward concave fill Joins.
type wedgeOp struct {
	body *topo.Body
	op   ops.PartFeatureOperation
}

// chamferByWedges bevels the edges by applying a per-edge wedge tool (plus the flat-corner blend
// cuts when requested) — the general faceted path used for every non-analytic chamfer.
func chamferByWedges(in Input, body *topo.Body, edges []*topo.Edge, d1, d2 float64, feat string,
	flatCorners bool, strategy types.ChamferConcaveStrategy, run chamferRun) (Output, error) {
	// A curved body (analytic cylinder) is re-faceted and the selected edges remapped to its faceted
	// segments, so the wedge cut works instead of hitting a degenerate closed edge (#129/#127).
	work, edges := planarizeForEdges(body, edges, feat)
	tools, convex, err := chamferWedgeTools(edges, d1, d2, strategy, run, feat)
	if err != nil {
		return Output{}, err
	}
	// The flat-corner blend is a convex-corner construct; concave edges have no pointy three-plane
	// tip to trim, so only the convex edges feed it.
	if flatCorners {
		for _, t := range cornerCutTools(convex, d1, d2, feat) {
			tools = append(tools, wedgeOp{body: t, op: ops.Cut})
		}
	}
	result, err := applyChamferTools(work, tools, in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// chamferWedgeTools builds the per-edge wedge for each resolved edge, classifying it convex or
// concave, and returns the tools-with-operation plus the convex-edge subset (for corner blending).
func chamferWedgeTools(edges []*topo.Edge, d1, d2 float64, strategy types.ChamferConcaveStrategy,
	run chamferRun, feat string) ([]wedgeOp, []*topo.Edge, error) {
	tools := make([]wedgeOp, 0, len(edges))
	convex := make([]*topo.Edge, 0, len(edges))
	for i, edge := range edges {
		name := fmt.Sprintf("%s/w%d", feat, i)
		if blend.ClassifyEdgeConvexity(edge) == blend.EdgeConcave {
			tool, err := concaveChamferWedge(edge, d1, d2, strategy, run, name)
			if err != nil {
				return nil, nil, err
			}
			tools = append(tools, wedgeOp{body: tool, op: concaveOp(strategy)})
			continue
		}
		tool, err := chamferWedge(edge, d1, d2, run, name)
		if err != nil {
			return nil, nil, err
		}
		tools = append(tools, wedgeOp{body: tool, op: ops.Cut})
		convex = append(convex, edge)
	}
	return tools, convex, nil
}

// concaveOp is the boolean a concave-edge wedge applies: an inward relief Cuts material, the
// default outward fill Joins it.
func concaveOp(strategy types.ChamferConcaveStrategy) ops.PartFeatureOperation {
	if strategy == types.ChamferConcaveInward {
		return ops.Cut
	}
	return ops.Join
}

// applyChamferTools applies each wedge to work in turn (Cut or Join), returning the body.
// rec collects the booleans' fallback diagnostics (#1601; nil discards).
func applyChamferTools(work *topo.Body, tools []wedgeOp, rec *diag.Recorder) (*topo.Body, error) {
	result := work
	for _, t := range tools {
		r, err := ops.BooleanWithDiagnostics(t.op, result, t.body, rec)
		if err != nil {
			return nil, err
		}
		result = r
	}
	return result, nil
}

// resolveEdges binds every edge key against the running body. A key that matches
// EXACTLY one edge binds cleanly (the ADR-0043 P0 guard: more than one is a
// topological-naming collision, never a silent first-match). A key whose exact entity
// is gone is recovered through the tiered binder — a lone surviving sibling sharing
// the key's parent lineage — and reported as a heal (the engine turns heals into a
// Warning, ADR-0043 P6); only a genuinely unrecoverable or ambiguous key is a hard
// error, so the feature goes Sick honestly rather than dressing up the wrong edge.
func resolveEdges(body *topo.Body, keys [][]byte, anchors map[string]math.Point3) ([]*topo.Edge, []ReferenceHeal, error) {
	edges := make([]*topo.Edge, len(keys))
	var heals []ReferenceHeal
	var ents []identity.Entity // built lazily, only when an exact match misses
	for i, k := range keys {
		match := body.EdgesByKey(k)
		if len(match) == 1 {
			edges[i] = match[0]
			continue
		}
		if ents == nil {
			ents = edgeEntities(body)
		}
		if e, mt := recoverEdge(k, anchorFor(k, anchors), ents); mt.IsFallback() && e != nil {
			edges[i] = e
			heals = append(heals, ReferenceHeal{Key: append([]byte(nil), k...), Match: mt})
			continue
		}
		if len(match) > 1 {
			return nil, nil, fmt.Errorf("dress-up: edge reference %q is ambiguous — it matches %d edges (a topological-naming collision) and no surviving sibling could be recovered", keyText(k), len(match))
		}
		return nil, nil, fmt.Errorf("dress-up: edge reference %q lost (no edge with that lineage on the running body, and no surviving sibling to recover it)", keyText(k))
	}
	return edges, heals, nil
}

// keyText renders a reference key as its readable lineage string (the leading kind byte stripped)
// for diagnostics, falling back to a hex-free best effort for an empty key.
func keyText(k []byte) string {
	if len(k) == 0 {
		return "<empty>"
	}
	if k[0] < 0x20 {
		return string(k[1:])
	}
	return string(k)
}

// chamferSetbacks resolves a chamfer definition's mode into the two face setbacks (d1 along
// the first adjacent face, d2 along the second): equal distance, two distances, or a distance
// plus the chamfer-face angle (d2 = d1·tan θ, exact for perpendicular faces, the box case).
func chamferSetbacks(def *ChamferDefinition) (d1, d2 float64, err error) {
	return chamferSetbackValues(def.Type, callOrZero(def.Distance), callOrZero(def.Distance2), callOrZero(def.Angle))
}

// chamferSetbackValues maps a chamfer type and its raw inputs to the two face setbacks (d1, d2).
// Shared by the part chamfer and the sheet-metal corner chamfer (#1967) so the distance /
// two-distance / distance-and-angle rule lives in one place. angle is in radians.
func chamferSetbackValues(ct types.ChamferType, d1, d2Raw, angle float64) (float64, float64, error) {
	switch ct {
	case types.ChamferTwoDistances:
		return d1, d2Raw, nil
	case types.ChamferDistanceAndAngle:
		if angle <= 0 || angle >= stdmath.Pi/2 {
			return 0, 0, fmt.Errorf("chamfer: angle %g rad must be in (0, π/2)", angle)
		}
		return d1, d1 * stdmath.Tan(angle), nil
	default: // ChamferDistance (and the zero value): symmetric
		return d1, d1, nil
	}
}

// chamferWedges builds a plain CUT wedge for each edge (setbacks d1, d2) — the convex-only path
// the sheet-metal corner / corner-seam reliefs reuse (they never fill or relieve a concave edge).
func chamferWedges(edges []*topo.Edge, d1, d2 float64, run chamferRun, feat string) ([]*topo.Body, error) {
	tools := make([]*topo.Body, 0, len(edges))
	for i, edge := range edges {
		tool, err := chamferWedge(edge, d1, d2, run, fmt.Sprintf("%s/w%d", feat, i))
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// cutAll subtracts each tool from work in turn, returning the carved body. Shared with the
// sheet-metal corner reliefs, which only ever cut. rec collects the cuts' boolean-fallback
// diagnostics (#1601; nil discards).
func cutAll(work *topo.Body, tools []*topo.Body, rec *diag.Recorder) (*topo.Body, error) {
	result := work
	for _, tool := range tools {
		r, err := ops.BooleanWithDiagnostics(ops.Cut, result, tool, rec)
		if err != nil {
			return nil, err
		}
		result = r
	}
	return result, nil
}

// chamferOverhang extends the wedge past each end of the edge. The overhang must reach past the
// lip remnant that the per-edge bevel otherwise leaves where the edge runs into an adjacent face
// (so the boolean consumes it and the bevel meets that face FLUSH); the lip sits at most about
// one setback past the end, so the overhang is scaled to the larger setback. The excess past the
// body boundary is trimmed by the boolean, so a generous value is safe at a convex end.
func chamferOverhang(d1, d2 float64) float64 {
	return 2 * stdmath.Max(d1, d2)
}

// wedgePrism builds the triangular prism for an edge from the section frame: a triangle with leg
// s1 along the first adjacent face's interior direction and s2 along the second's, swept along
// the edge with the given overhang past each end. A NEGATIVE leg points the triangle to the
// material side of the face (used by the concave relief cut); a point reflection preserves the
// 2D winding, so buildPrism orients either form into a valid solid the boolean can apply.
func wedgePrism(fr edgeFrame, s1, s2, overhang float64, run chamferRun, feat string) *topo.Body {
	poly := []math.Point2{{X: 0, Y: 0}, fr.proj(fr.t1.Scale(s1)), fr.proj(fr.t2.Scale(s2))}
	return buildPrism(poly, fr.plane, wedgeSpan(fr, run, overhang), 0, feat)
}

// chamferWedge builds the triangular prism CUT to bevel a convex edge: legs d1, d2 along the two
// adjacent faces' interiors, with an overhang past each end so the boolean meets the neighbours
// flush. Equal d1==d2 is the symmetric chamfer; d1≠d2 is the asymmetric two-distance / angle one.
func chamferWedge(edge *topo.Edge, d1, d2 float64, run chamferRun, feat string) (*topo.Body, error) {
	fr, err := edgeCornerFrame(edge, "chamfer")
	if err != nil {
		return nil, err
	}
	s1, s2 := orderedSetbacks(edge, d1, d2, run.reference)
	return wedgePrism(fr, s1, s2, chamferOverhang(s1, s2), run, feat), nil
}

// concaveChamferWedge builds the wedge for a CONCAVE (internal) edge. The two faces' interior
// directions point into the open notch, so the wedge {edge, d1·t1, d2·t2} lies in the void:
//   - outward (default): JOIN that wedge to fill the inside corner with a 45° gusset. No overhang
//     — an overhanging fill would protrude past the edge's end faces (it adds, not removes).
//   - inward: reflect the legs to the material side (−d1·t1, −d2·t2) and CUT, gouging a recessed
//     relief groove out of the corner; an overhang keeps that cut flush at the ends.
func concaveChamferWedge(edge *topo.Edge, d1, d2 float64, strategy types.ChamferConcaveStrategy,
	run chamferRun, feat string) (*topo.Body, error) {
	fr, err := edgeCornerFrame(edge, "chamfer")
	if err != nil {
		return nil, err
	}
	s1, s2 := orderedSetbacks(edge, d1, d2, run.reference)
	if strategy == types.ChamferConcaveInward {
		return wedgePrism(fr, -s1, -s2, chamferOverhang(s1, s2), run, feat), nil
	}
	return wedgePrism(fr, s1, s2, 0, run, feat), nil
}

// edgeFrame is the resolved corner-cut frame at an edge (see edgeCornerFrame).
type edgeFrame struct {
	t1, t2 math.Vector3
	plane  sketch.Plane
	proj   func(math.Vector3) math.Point2
	length float64
}

// edgeCornerFrame resolves the section frame at an edge for a corner cut: the two adjacent
// faces' interior directions (t1/t2), the plane perpendicular to the edge, a projector into
// that plane's 2D coords, and the edge length. Shared by the chamfer wedge and the
// sheet-metal corner-seam relief, which differ only in the 2D notch polygon they build.
func edgeCornerFrame(edge *topo.Edge, what string) (edgeFrame, error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return edgeFrame{}, fmt.Errorf("%s: edge bounds %d faces, need 2", what, len(faces))
	}
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return edgeFrame{}, fmt.Errorf("%s: degenerate edge", what)
	}
	mid := v0.Midpoint(v1)
	t1 := interiorDir(faces[0], mid, e)
	t2 := interiorDir(faces[1], mid, e)
	if t1.LengthSquared() == 0 || t2.LengthSquared() == 0 {
		return edgeFrame{}, fmt.Errorf("%s: cannot orient the cut against the edge faces", what)
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	return edgeFrame{
		t1: t1, t2: t2, plane: plane, length: v0.DistanceTo(v1),
		proj: func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) },
	}, nil
}

// interiorDir returns the unit direction, perpendicular to the edge, pointing from the
// edge into the face's interior — the direction the chamfer sets back along that face.
func interiorDir(f *topo.Face, edgeMid math.Point3, e math.UnitVector3) math.Vector3 {
	toCentroid := edgeMid.VectorTo(centroidOf(faceVertexPoints(f)))
	perp := toCentroid.Sub(e.AsVector().Scale(toCentroid.Dot(e.AsVector())))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return math.V3(0, 0, 0)
	}
	return u.AsVector()
}
