// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
func chamferEdges(in Input, keys [][]byte, d1, d2 float64, feat string, flatCorners bool, strategy types.ChamferConcaveStrategy) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if d1 <= 0 || d2 <= 0 {
		return Output{}, fmt.Errorf("chamfer: setbacks (%g, %g) must both be > 0", d1, d2)
	}
	edges, err := resolveEdges(body, keys)
	if err != nil {
		return Output{}, err
	}
	// The analytic conical chamfer (#127) assumes a SYMMETRIC setback on a CONVEX cylinder rim;
	// an asymmetric chamfer or any concave edge takes the faceted-wedge path (which develops the
	// per-edge fill/relief). The flat-corner blend handles both symmetric and asymmetric setbacks.
	if d1 == d2 && allConvex(edges) {
		// A rim of a simple analytic cylinder gets a TRUE conical chamfer (one geom.Cone face) by
		// rebuilding the body as a surface of revolution (#127). Anything else falls through.
		if res, ok := analyticCylinderChamfer(body, edges, d1, feat); ok {
			return Output{Bodies: replaceBody(in.Bodies, body, res)}, nil
		}
	}
	return chamferByWedges(in, body, edges, d1, d2, feat, flatCorners, strategy)
}

// allConvex reports whether every edge is a convex dihedral (so the analytic conical fast path,
// which assumes material is cut away, is valid for the whole selection).
func allConvex(edges []*topo.Edge) bool {
	for _, e := range edges {
		if ops.ClassifyEdgeConvexity(e) == ops.EdgeConcave {
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
func chamferByWedges(in Input, body *topo.Body, edges []*topo.Edge, d1, d2 float64, feat string, flatCorners bool, strategy types.ChamferConcaveStrategy) (Output, error) {
	// A curved body (analytic cylinder) is re-faceted and the selected edges remapped to its faceted
	// segments, so the wedge cut works instead of hitting a degenerate closed edge (#129/#127).
	work, edges := planarizeForEdges(body, edges, feat)
	tools, convex, err := chamferWedgeTools(edges, d1, d2, strategy, feat)
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
	result, err := applyChamferTools(work, tools)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// chamferWedgeTools builds the per-edge wedge for each resolved edge, classifying it convex or
// concave, and returns the tools-with-operation plus the convex-edge subset (for corner blending).
func chamferWedgeTools(edges []*topo.Edge, d1, d2 float64, strategy types.ChamferConcaveStrategy, feat string) ([]wedgeOp, []*topo.Edge, error) {
	tools := make([]wedgeOp, 0, len(edges))
	convex := make([]*topo.Edge, 0, len(edges))
	for i, edge := range edges {
		name := fmt.Sprintf("%s/w%d", feat, i)
		if ops.ClassifyEdgeConvexity(edge) == ops.EdgeConcave {
			tool, err := concaveChamferWedge(edge, d1, d2, strategy, name)
			if err != nil {
				return nil, nil, err
			}
			tools = append(tools, wedgeOp{body: tool, op: concaveOp(strategy)})
			continue
		}
		tool, err := chamferWedge(edge, d1, d2, name)
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
func applyChamferTools(work *topo.Body, tools []wedgeOp) (*topo.Body, error) {
	result := work
	for _, t := range tools {
		r, err := ops.Boolean(t.op, result, t.body)
		if err != nil {
			return nil, err
		}
		result = r
	}
	return result, nil
}

// resolveEdges binds every edge key against the original body, erroring if a key is lost
// (so the feature goes sick honestly).
func resolveEdges(body *topo.Body, keys [][]byte) ([]*topo.Edge, error) {
	edges := make([]*topo.Edge, len(keys))
	for i, k := range keys {
		edge, ok := body.FindEdgeByKey(k)
		if !ok {
			return nil, fmt.Errorf("chamfer: edge reference lost")
		}
		edges[i] = edge
	}
	return edges, nil
}

// chamferSetbacks resolves a chamfer definition's mode into the two face setbacks (d1 along
// the first adjacent face, d2 along the second): equal distance, two distances, or a distance
// plus the chamfer-face angle (d2 = d1·tan θ, exact for perpendicular faces, the box case).
func chamferSetbacks(def *ChamferDefinition) (d1, d2 float64, err error) {
	d1 = callOrZero(def.Distance)
	switch def.Type {
	case types.ChamferTwoDistances:
		d2 = callOrZero(def.Distance2)
	case types.ChamferDistanceAndAngle:
		a := callOrZero(def.Angle)
		if a <= 0 || a >= stdmath.Pi/2 {
			return 0, 0, fmt.Errorf("chamfer: angle %g rad must be in (0, π/2)", a)
		}
		d2 = d1 * stdmath.Tan(a)
	default: // ChamferDistance (and the zero value): symmetric
		d2 = d1
	}
	return d1, d2, nil
}

// chamferWedges builds a plain CUT wedge for each edge (setbacks d1, d2) — the convex-only path
// the sheet-metal corner / corner-seam reliefs reuse (they never fill or relieve a concave edge).
func chamferWedges(edges []*topo.Edge, d1, d2 float64, feat string) ([]*topo.Body, error) {
	tools := make([]*topo.Body, 0, len(edges))
	for i, edge := range edges {
		tool, err := chamferWedge(edge, d1, d2, fmt.Sprintf("%s/w%d", feat, i))
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// cutAll subtracts each tool from work in turn, returning the carved body. Shared with the
// sheet-metal corner reliefs, which only ever cut.
func cutAll(work *topo.Body, tools []*topo.Body) (*topo.Body, error) {
	result := work
	for _, tool := range tools {
		r, err := ops.Boolean(ops.Cut, result, tool)
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
func wedgePrism(fr edgeFrame, s1, s2, overhang float64, feat string) *topo.Body {
	poly := []math.Point2{{X: 0, Y: 0}, fr.proj(fr.t1.Scale(s1)), fr.proj(fr.t2.Scale(s2))}
	return buildPrism(poly, fr.plane, span{near: -overhang, far: fr.length + overhang}, 0, feat)
}

// chamferWedge builds the triangular prism CUT to bevel a convex edge: legs d1, d2 along the two
// adjacent faces' interiors, with an overhang past each end so the boolean meets the neighbours
// flush. Equal d1==d2 is the symmetric chamfer; d1≠d2 is the asymmetric two-distance / angle one.
func chamferWedge(edge *topo.Edge, d1, d2 float64, feat string) (*topo.Body, error) {
	fr, err := edgeCornerFrame(edge, "chamfer")
	if err != nil {
		return nil, err
	}
	return wedgePrism(fr, d1, d2, chamferOverhang(d1, d2), feat), nil
}

// concaveChamferWedge builds the wedge for a CONCAVE (internal) edge. The two faces' interior
// directions point into the open notch, so the wedge {edge, d1·t1, d2·t2} lies in the void:
//   - outward (default): JOIN that wedge to fill the inside corner with a 45° gusset. No overhang
//     — an overhanging fill would protrude past the edge's end faces (it adds, not removes).
//   - inward: reflect the legs to the material side (−d1·t1, −d2·t2) and CUT, gouging a recessed
//     relief groove out of the corner; an overhang keeps that cut flush at the ends.
func concaveChamferWedge(edge *topo.Edge, d1, d2 float64, strategy types.ChamferConcaveStrategy, feat string) (*topo.Body, error) {
	fr, err := edgeCornerFrame(edge, "chamfer")
	if err != nil {
		return nil, err
	}
	if strategy == types.ChamferConcaveInward {
		return wedgePrism(fr, -d1, -d2, chamferOverhang(d1, d2), feat), nil
	}
	return wedgePrism(fr, d1, d2, 0, feat), nil
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

// threeEdgeCorner is a vertex where exactly three selected edges meet — the corner that
// gets a flat triangular blend.
type threeEdgeCorner struct {
	vertex *topo.Vertex
	edges  [3]*topo.Edge
}

// cornerCutTools builds the flat-corner blend cut for every three-edge corner among the
// selected edges, using the per-face setbacks (d1, d2) so the blend is correct for both
// symmetric and asymmetric chamfers. A degenerate corner (collinear edges, zero-volume
// tetra) is skipped.
func cornerCutTools(edges []*topo.Edge, d1, d2 float64, feat string) []*topo.Body {
	tools := make([]*topo.Body, 0)
	for i, c := range threeEdgeCorners(edges) {
		if tool, ok := cornerTetra(c, d1, d2, fmt.Sprintf("%s/c%d", feat, i)); ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// threeEdgeCorners groups the selected edges by shared vertex and returns the corners
// where exactly three of them meet, ordered by vertex id so the cut sequence (and the
// lineage it stamps) is reproducible across recomputes.
func threeEdgeCorners(edges []*topo.Edge) []threeEdgeCorner {
	at := map[uint64][]*topo.Edge{}
	verts := map[uint64]*topo.Vertex{}
	for _, e := range edges {
		for _, v := range e.Vertices() {
			at[v.ID()] = append(at[v.ID()], e)
			verts[v.ID()] = v
		}
	}
	corners := make([]threeEdgeCorner, 0)
	for id, es := range at {
		if len(es) == 3 {
			corners = append(corners, threeEdgeCorner{vertex: verts[id], edges: [3]*topo.Edge{es[0], es[1], es[2]}})
		}
	}
	sort.Slice(corners, func(i, j int) bool { return corners[i].vertex.ID() < corners[j].vertex.ID() })
	return corners
}

// cornerTetra builds the tetrahedron cut that flattens a three-edge corner. Once the three
// edge wedges are cut, their chamfer faces meet at a single pointy tip (the three chamfer
// planes' intersection). Subtracting the tetra whose apex is that tip and whose three base
// vertices are the outer ends of the chamfer-pair edges (see cornerBasePoints) trims the
// protruding tip and exposes one flat triangular face. ok is false when any of the defining
// planes are parallel/degenerate.
func cornerTetra(c threeEdgeCorner, d1, d2 float64, feat string) (*topo.Body, bool) {
	planes, ok := cornerChamferPlanes(c, d1, d2)
	if !ok {
		return nil, false
	}
	tip, ok := threePlaneIntersection(planes[0], planes[1], planes[2])
	if !ok {
		return nil, false
	}
	base, ok := cornerBasePoints(c, planes)
	if !ok || degenerateTetra(tip, base) {
		return nil, false
	}
	return buildTetra([4]math.Point3{tip, base[0], base[1], base[2]}, feat), true
}

// cornerBasePoints returns the three vertices of the flat triangular face. For each pair of
// edges, the vertex is where their two chamfer planes meet the original face the two edges
// share — the outer vertex the chamfer-pair edge runs to. Each tetra side face then lands
// on a chamfer plane, so the boolean trims the pointy tip cleanly and exposes the triangle.
func cornerBasePoints(c threeEdgeCorner, planes [3]geom.Plane) ([3]math.Point3, bool) {
	pairs := [3][2]int{{0, 1}, {0, 2}, {1, 2}}
	var base [3]math.Point3
	for k, pr := range pairs {
		shared, ok := sharedFacePlane(c.edges[pr[0]], c.edges[pr[1]], c.vertex)
		if !ok {
			return base, false
		}
		q, ok := threePlaneIntersection(planes[pr[0]], planes[pr[1]], shared)
		if !ok {
			return base, false
		}
		base[k] = q
	}
	return base, true
}

// sharedFacePlane returns the plane of the original face that two corner edges share (each
// edge bounds two faces; two edges meeting at a convex corner share exactly one).
func sharedFacePlane(a, b *topo.Edge, corner *topo.Vertex) (geom.Plane, bool) {
	f := sharedFace(a, b)
	if f == nil {
		return geom.Plane{}, false
	}
	pl, err := geom.NewPlane(corner.Point(), f.Geometry().NormalAt(0, 0))
	if err != nil {
		return geom.Plane{}, false
	}
	return pl, true
}

// sharedFace returns the single face bounding both edges, or nil if they share none.
func sharedFace(a, b *topo.Edge) *topo.Face {
	for _, fa := range a.Faces() {
		for _, fb := range b.Faces() {
			if fa.ID() == fb.ID() {
				return fa
			}
		}
	}
	return nil
}

// cornerChamferPlanes builds the chamfer-face plane of each of the corner's three edges from
// the per-face setbacks (d1, d2).
func cornerChamferPlanes(c threeEdgeCorner, d1, d2 float64) ([3]geom.Plane, bool) {
	var planes [3]geom.Plane
	for i, e := range c.edges {
		pl, ok := chamferPlane(e, c.vertex, d1, d2)
		if !ok {
			return planes, false
		}
		planes[i] = pl
	}
	return planes, true
}

// chamferPlane reconstructs the plane of an edge's chamfer face at this corner: it offsets the
// edge's first adjacent face by d1 and its second by d2 — the SAME per-face setbacks (and the
// same edge.Faces() ordering) chamferWedge cuts with — and runs parallel to the edge. So the
// reconstructed plane is the wedge's hypotenuse for asymmetric chamfers (d1≠d2) as well as the
// symmetric case (d1==d2), which makes the flat-corner blend land on the real chamfer faces.
func chamferPlane(e *topo.Edge, corner *topo.Vertex, d1, d2 float64) (geom.Plane, bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.Plane{}, false
	}
	dir, ok := edgeDirFrom(e, corner)
	if !ok {
		return geom.Plane{}, false
	}
	p := corner.Point()
	t1, t2 := interiorDir(faces[0], p, dir), interiorDir(faces[1], p, dir)
	if t1.LengthSquared() == 0 || t2.LengthSquared() == 0 {
		return geom.Plane{}, false
	}
	a := p.TranslateBy(t1.Scale(d1)) // setback point on the first adjacent face
	b := p.TranslateBy(t2.Scale(d2)) // setback point on the second adjacent face
	pl, err := geom.NewPlaneFromAxes(a, dir.AsVector(), a.VectorTo(b))
	if err != nil {
		return geom.Plane{}, false
	}
	return pl, true
}

// threePlaneIntersection returns the single point common to three planes via Cramer's rule,
// or ok=false when they share no unique point (near-zero normal determinant).
func threePlaneIntersection(a, b, c geom.Plane) (math.Point3, bool) {
	n1, n2, n3 := a.Normal(), b.Normal(), c.Normal()
	det := n1.Dot(n2.Cross(n3))
	if stdmath.Abs(float64(det)) < singularDetTol {
		return math.Point3{}, false
	}
	d1 := a.Origin.AsVector().Dot(n1)
	d2 := b.Origin.AsVector().Dot(n2)
	d3 := c.Origin.AsVector().Dot(n3)
	num := n2.Cross(n3).Scale(d1).Add(n3.Cross(n1).Scale(d2)).Add(n1.Cross(n2).Scale(d3))
	return math.P3(0, 0, 0).TranslateBy(num.Scale(1 / det)), true
}

// edgeDirFrom returns the unit direction leaving v along edge e toward its other vertex.
func edgeDirFrom(e *topo.Edge, v *topo.Vertex) (math.UnitVector3, bool) {
	other := e.EndVertex()
	if other.ID() == v.ID() {
		other = e.StartVertex()
	}
	u, err := math.UnitVector3FromVector(v.Point().VectorTo(other.Point()))
	if err != nil {
		return math.UnitVector3{}, false
	}
	return u, true
}

// degenerateTetra reports whether the apex and the three base points are (near) coplanar,
// so the cut solid would have no volume (scalar triple product ≈ 0).
func degenerateTetra(apex math.Point3, base [3]math.Point3) bool {
	a := apex.VectorTo(base[0])
	b := apex.VectorTo(base[1])
	c := apex.VectorTo(base[2])
	return stdmath.Abs(float64(a.Cross(b).Dot(c))) < singularDetTol
}

// tetraEdges holds the six edges of a tetrahedron keyed by their ascending vertex-index
// pair, so the four triangular faces can share them with the right traversal direction.
type tetraEdges map[[2]int]*topo.Edge

// use returns the oriented use that traverses the tetra edge from vertex i to vertex j.
func (te tetraEdges) use(i, j int) topo.Use {
	if i < j {
		return topo.Fwd(te[[2]int{i, j}])
	}
	return topo.Rev(te[[2]int{j, i}])
}

// buildTetra assembles a solid tetrahedron from four points as a boolean cut tool. Each
// triangular face is planar with its plane oriented outward (away from the opposite
// vertex) and its loop wound to match, so the body is a valid closed solid the boolean can
// subtract. Vertex 0 is the apex; 1..3 the base.
func buildTetra(p [4]math.Point3, feat string) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	var v [4]*topo.Vertex
	for i := range p {
		v[i] = bld.AddVertex(p[i], topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := newTetraEdges(bld, p, v, feat)
	faces := [4][3]int{{1, 2, 3}, {0, 2, 3}, {0, 1, 3}, {0, 1, 2}}
	opposite := [4]int{0, 1, 2, 3}
	for fi, tri := range faces {
		addTetraFace(bld, p, edges, tri, opposite[fi], feat, fi)
	}
	return bld.Build()
}

// newTetraEdges builds the six line-segment edges of the tetrahedron, each stored under
// its ascending vertex-index pair.
func newTetraEdges(bld *topo.Builder, p [4]math.Point3, v [4]*topo.Vertex, feat string) tetraEdges {
	pairs := [6][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	te := make(tetraEdges, 6)
	for k, pr := range pairs {
		te[pr] = bld.AddEdge(geom.NewLineSegment(p[pr[0]], p[pr[1]]), v[pr[0]], v[pr[1]], topo.NewLineage(topo.Tok(feat, "edge", k)))
	}
	return te
}

// addTetraFace adds the triangular face through corner triple tri, flipping the traversal
// so its plane normal points away from the opposite vertex (outward) and winding the loop
// to match. A near-degenerate triangle is dropped (cornerTetra already filters flat
// tetras).
func addTetraFace(bld *topo.Builder, p [4]math.Point3, te tetraEdges, tri [3]int, opp int, feat string, fi int) {
	a, b, c := tri[0], tri[1], tri[2]
	n := p[a].VectorTo(p[b]).Cross(p[a].VectorTo(p[c]))
	if n.Dot(p[a].VectorTo(p[opp])) > 0 { // points toward the interior vertex → flip
		b, c = c, b
		n = n.Negate()
	}
	unit, err := math.UnitVector3FromVector(n)
	if err != nil {
		return
	}
	surf, _ := geom.NewPlane(p[a], unit.AsVector())
	loop := topo.OuterLoop(te.use(a, b), te.use(b, c), te.use(c, a))
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", fi)), loop)
}
