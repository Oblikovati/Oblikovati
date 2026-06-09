// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"sort"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// singularDetTol is the magnitude below which a normal determinant (three planes' triple
// product) or a scalar triple product of edge vectors is treated as zero: the three planes
// share no unique meeting point, or the four tetra points are coplanar so the cut has no
// volume. It sits below the linear DefaultTolerance because it bounds a product of three
// (roughly unit) vectors, not a length.
const singularDetTol = 1e-12

// chamferEdges bevels each selected edge by cutting a triangular wedge tool that runs
// along it. All cut tools are built from the original body up front (a boolean rebuilds
// topology with new lineage, so a reference key would not survive the first cut), then
// each is subtracted in turn. Convex edges only in phase A — a concave edge would add
// material, which is a follow-up.
//
// When flatCorners is set, every vertex where exactly three selected edges meet also gets
// a tetrahedron cut that trims the pointy three-plane intersection into one flat
// triangular face — the way Inventor blends such a corner by default. With it clear the
// three chamfer planes are left to meet at a point.
func chamferEdges(in Input, keys [][]byte, dist float64, feat string, flatCorners bool) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if dist <= 0 {
		return Output{}, fmt.Errorf("chamfer: distance %g must be > 0", dist)
	}
	edges, err := resolveEdges(body, keys)
	if err != nil {
		return Output{}, err
	}
	// A curved body (analytic cylinder) is re-faceted and the selected edges remapped to its faceted
	// segments, so the wedge cut works instead of hitting a degenerate closed edge (#129/#127).
	work, edges := planarizeForEdges(body, edges, feat)
	tools, err := chamferWedges(edges, dist, feat)
	if err != nil {
		return Output{}, err
	}
	if flatCorners {
		tools = append(tools, cornerCutTools(edges, dist, feat)...)
	}
	result := work
	for _, tool := range tools {
		if result, err = ops.Boolean(ops.Cut, result, tool); err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
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

// chamferWedges builds the bevel wedge for each resolved edge.
func chamferWedges(edges []*topo.Edge, dist float64, feat string) ([]*topo.Body, error) {
	tools := make([]*topo.Body, 0, len(edges))
	for i, edge := range edges {
		tool, err := chamferWedge(edge, dist, fmt.Sprintf("%s/w%d", feat, i))
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// chamferWedge builds the triangular prism removed to bevel an edge: a right-triangle
// cross-section with legs `dist` along each adjacent face's interior, swept along the
// edge (with a small overhang past each end so the boolean is clean).
func chamferWedge(edge *topo.Edge, dist float64, feat string) (*topo.Body, error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return nil, fmt.Errorf("chamfer: edge bounds %d faces, need 2", len(faces))
	}
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, fmt.Errorf("chamfer: degenerate edge")
	}
	mid := v0.Midpoint(v1)
	t1 := interiorDir(faces[0], mid, e)
	t2 := interiorDir(faces[1], mid, e)
	if t1.LengthSquared() == 0 || t2.LengthSquared() == 0 {
		return nil, fmt.Errorf("chamfer: cannot orient the wedge against the edge faces")
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	proj := func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) }
	poly := []math.Point2{{X: 0, Y: 0}, proj(t1.Scale(dist)), proj(t2.Scale(dist))}
	return buildPrism(poly, plane, span{near: -cutterOverhang, far: v0.DistanceTo(v1) + cutterOverhang}, 0, feat), nil
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
// selected edges. A degenerate corner (collinear edges, zero-volume tetra) is skipped.
func cornerCutTools(edges []*topo.Edge, dist float64, feat string) []*topo.Body {
	tools := make([]*topo.Body, 0)
	for i, c := range threeEdgeCorners(edges) {
		if tool, ok := cornerTetra(c, dist, fmt.Sprintf("%s/c%d", feat, i)); ok {
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
func cornerTetra(c threeEdgeCorner, dist float64, feat string) (*topo.Body, bool) {
	planes, ok := cornerChamferPlanes(c, dist)
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

// cornerChamferPlanes builds the chamfer-face plane of each of the corner's three edges.
func cornerChamferPlanes(c threeEdgeCorner, dist float64) ([3]geom.Plane, bool) {
	var planes [3]geom.Plane
	for i, e := range c.edges {
		pl, ok := chamferPlane(e, c.vertex, dist)
		if !ok {
			return planes, false
		}
		planes[i] = pl
	}
	return planes, true
}

// chamferPlane reconstructs the plane of an edge's chamfer face at this corner: through the
// face's two setback offsets (dist along each adjacent face interior) and parallel to the
// edge — the same hypotenuse plane chamferWedge cuts with.
func chamferPlane(e *topo.Edge, corner *topo.Vertex, dist float64) (geom.Plane, bool) {
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
	pl, err := geom.NewPlaneFromAxes(p.TranslateBy(t1.Scale(dist)), dir.AsVector(), t2.Sub(t1))
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
