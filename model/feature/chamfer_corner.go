// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chamfer — the FLAT THREE-EDGE-CORNER blend (M48 #2232 split of chamfer.go). Where exactly three
// selected edges meet at a vertex, the wedge cuts leave a small notch; this fills it with a flat
// triangular tetra cut whose three faces lie on the three edges' chamfer planes. Solved from the
// planes' triple intersection (declining when they are singular/degenerate). The wedge-cut driver and
// setback/mode resolution live in chamfer.go; the tetra body builder in chamfer_tetra.go.

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
