// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FilletEdges rounds the selected convex straight edges of a planar solid with a constant-
// radius rolling-ball blend: each edge between two planar faces is replaced by a cylinder
// face of radius r tangent to both, the two faces are retrimmed back to the tangent lines,
// and the end faces gain a quarter-arc at the rounded corner. All edges are resolved and
// solved on the original body, then applied in a single rebuild, so independent edges that
// share a face (e.g. the four verticals of a box) all retrim that face correctly. Convex,
// straight edges with one extra face at each end (box/prism edges); chains, corners where
// fillets meet, and concave edges are follow-ups.
func FilletEdges(body *topo.Body, edgeKeys [][]byte, r float64) (*topo.Body, error) {
	if r <= 0 {
		return nil, fmt.Errorf("fillet: radius %g must be > 0", r)
	}
	edges, err := resolveFilletEdges(body, edgeKeys)
	if err != nil {
		return nil, err
	}
	blends, err := computeBlends(edges, r)
	if err != nil {
		return nil, err
	}
	fils := make([]edgeFillet, 0, len(edges))
	for _, e := range edges {
		ef, err := computeEdgeFillet(body, e, r, blends)
		if err != nil {
			return nil, err
		}
		fils = append(fils, ef)
	}
	res := assembleBody(filletResultFaces(body, fils, blends), "fillet")
	if rep := Validate(res); !rep.Valid || !res.IsSolid() {
		return nil, fmt.Errorf("fillet: result is not a valid solid %v", rep.Issues)
	}
	return res, nil
}

// resolveFilletEdges resolves the edge reference keys against the body, erroring on a lost key.
func resolveFilletEdges(body *topo.Body, keys [][]byte) ([]*topo.Edge, error) {
	edges := make([]*topo.Edge, 0, len(keys))
	for _, k := range keys {
		e, ok := body.FindEdgeByKey(k)
		if !ok {
			return nil, fmt.Errorf("fillet: edge reference lost: %x", k)
		}
		edges = append(edges, e)
	}
	return edges, nil
}

// corner is one rounded end of a filleted edge: the cylinder centre at that end, the tangent
// points on faces a/b, and the arc midpoint (the cylinder point nearest the sharp corner).
// At a blend corner the centre is the corner sphere's centre and the tangent points are the
// sphere's tangents (the cylinder ends there and its arc joins the sphere patch).
type corner struct {
	a, b    *topo.Face
	cen     math.Point3 // cylinder centre at this end (sphere centre when blended)
	ta, tb  math.Point3
	mid     math.Point3
	endFace *topo.Face // the flat end cap to arc (nil at a blend corner)
	vertex  *topo.Vertex
	blend   bool
}

// tOf returns the tangent point on face f (a or b).
func (c corner) tOf(f *topo.Face) math.Point3 {
	if f == c.a {
		return c.ta
	}
	return c.tb
}

// edgeFillet is a fully solved fillet of one edge: its two faces, the cylinder, and the two
// rounded corners.
type edgeFillet struct {
	a, b   *topo.Face
	cyl    geom.Cylinder
	c0, c1 corner
	edge   *topo.Edge
}

// computeEdgeFillet solves the rolling-ball geometry for one convex straight edge, using a
// corner blend at either endpoint that is a shared corner.
func computeEdgeFillet(body *topo.Body, e *topo.Edge, r float64, blends map[uint64]*cornerBlend) (edgeFillet, error) {
	a, b, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		return edgeFillet{}, err
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return edgeFillet{}, fmt.Errorf("fillet: degenerate edge")
	}
	off := nA.Add(nB).Scale(-r / (1 + nA.Dot(nB))) // centre offset from the edge into the solid
	if mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point()); !PointInsideBody(body, mid.TranslateBy(off)) {
		return edgeFillet{}, fmt.Errorf("fillet: edge is not convex (only convex edges are supported)")
	}
	in := cornerInputs{a: a, b: b, nA: nA, nB: nB, off: off, r: r, axis: axis.AsVector()}
	c0, c1, err := edgeCorners(e, in, blends)
	if err != nil {
		return edgeFillet{}, err
	}
	cyl, err := geom.NewCylinder(c0.cen, axis.AsVector(), r)
	if err != nil {
		return edgeFillet{}, err
	}
	return edgeFillet{a: a, b: b, cyl: cyl, c0: c0, c1: c1, edge: e}, nil
}

// edgeCorners solves the rounded corners at both endpoints of an edge (each blended when its
// vertex is a shared corner).
func edgeCorners(e *topo.Edge, in cornerInputs, blends map[uint64]*cornerBlend) (c0, c1 corner, err error) {
	if c0, err = cornerAt(e.StartVertex(), in, blends[e.StartVertex().ID()]); err != nil {
		return corner{}, corner{}, err
	}
	c1, err = cornerAt(e.EndVertex(), in, blends[e.EndVertex().ID()])
	return c0, c1, err
}

// edgePlanarFaces returns the edge's two faces and their outward normals, erroring unless
// the edge bounds exactly two planar faces.
func edgePlanarFaces(e *topo.Edge) (a, b *topo.Face, nA, nB math.Vector3, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, nA, nB, fmt.Errorf("fillet: edge bounds %d faces, need 2", len(faces))
	}
	pa, oka := faces[0].Geometry().(geom.Plane)
	pb, okb := faces[1].Geometry().(geom.Plane)
	if !oka || !okb {
		return nil, nil, nA, nB, fmt.Errorf("fillet: both faces of the edge must be planar")
	}
	return faces[0], faces[1], pa.Normal(), pb.Normal(), nil
}

// cornerInputs bundles the per-edge data a corner needs.
type cornerInputs struct {
	a, b   *topo.Face
	nA, nB math.Vector3
	off    math.Vector3
	axis   math.Vector3
	r      float64
}

// cornerAt solves a fillet corner at vertex v. Without a blend it is a simple end: centre
// v+off, tangent points r along each face normal, an arc on the end face. With a blend (v is
// a shared corner) the centre is the blend sphere's centre and the tangent points are the
// sphere's tangents on the two faces; the corner-end arc joins the sphere patch (no end
// face), and the arc is registered on the blend.
func cornerAt(v *topo.Vertex, in cornerInputs, blend *cornerBlend) (corner, error) {
	cen := v.Point().TranslateBy(in.off)
	ta := cen.TranslateBy(in.nA.Scale(in.r))
	tb := cen.TranslateBy(in.nB.Scale(in.r))
	var end *topo.Face
	if blend != nil {
		cen, ta, tb = blend.center, blend.tan[in.a.ID()], blend.tan[in.b.ID()]
	} else if end = endFaceAt(v, in.a, in.b); end == nil {
		return corner{}, fmt.Errorf("fillet: edge endpoint has no end face to round")
	}
	mid := cen.TranslateBy(perpToward(cen, v.Point(), in.axis).Scale(in.r))
	c := corner{a: in.a, b: in.b, endFace: end, vertex: v, cen: cen, ta: ta, tb: tb, mid: mid, blend: blend != nil}
	if blend != nil {
		blend.arcs = append(blend.arcs, blendArc{ta: ta, tb: tb, mid: mid})
	}
	return c, nil
}

// perpToward returns the unit direction from cen toward p projected into the plane
// perpendicular to axis — the in-cross-section direction to the rounded corner.
func perpToward(cen, p math.Point3, axis math.Vector3) math.Vector3 {
	d := cen.VectorTo(p)
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return d
	}
	return u.AsVector()
}

// endFaceAt returns the face meeting at v that is neither a nor b (the end cap the fillet
// rounds), or nil if there is none.
func endFaceAt(v *topo.Vertex, a, b *topo.Face) *topo.Face {
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if f != a && f != b {
				return f
			}
		}
	}
	return nil
}

// blendArc is one boundary arc of a corner sphere patch (shared with a cylinder fillet):
// from ta to tb through mid, all on the sphere.
type blendArc struct{ ta, tb, mid math.Point3 }

// cornerBlend is a spherical corner patch where several filleted edges meet at one vertex:
// the rolling-ball sphere tangent to the corner's faces, its tangent point on each face
// (keyed by face id), and the arcs (filled in as the edges are solved) that bound the patch.
type cornerBlend struct {
	vertex *topo.Vertex
	center math.Point3
	sphere geom.Sphere
	tan    map[uint64]math.Point3
	arcs   []blendArc
}

// computeBlends finds the shared corners of the filleted edge set and solves a sphere patch
// for each. A vertex where ≥2 filleted edges meet must be a fully-filleted trihedral corner
// (exactly 3 of the selected edges, 3 faces) — the supported case (e.g. a box corner);
// anything else errors clearly. Returns a map keyed by corner vertex id.
func computeBlends(edges []*topo.Edge, r float64) (map[uint64]*cornerBlend, error) {
	groups := map[uint64][]*topo.Edge{}
	for _, e := range edges {
		groups[e.StartVertex().ID()] = append(groups[e.StartVertex().ID()], e)
		groups[e.EndVertex().ID()] = append(groups[e.EndVertex().ID()], e)
	}
	out := map[uint64]*cornerBlend{}
	for vid, es := range groups {
		if len(es) < 2 {
			continue
		}
		v := vertexByID(es, vid)
		faces := facesAtVertex(v)
		if len(es) != 3 || len(faces) != 3 {
			return nil, fmt.Errorf("fillet: corner blend needs exactly 3 mutually filleted edges at a 3-face vertex (got %d edges, %d faces); other corner configs are not yet supported", len(es), len(faces))
		}
		cb, err := solveBlend(v, faces, r)
		if err != nil {
			return nil, err
		}
		out[vid] = cb
	}
	return out, nil
}

// solveBlend builds the corner sphere from the three planar faces meeting at v (the point
// at distance r from all three, inside) and its tangent points on each.
func solveBlend(v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	var a [3][3]float64
	var b [3]float64
	for i, f := range faces {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return nil, fmt.Errorf("fillet: corner face must be planar")
		}
		n := pl.Normal()
		a[i] = [3]float64{n.X, n.Y, n.Z}
		b[i] = n.Dot(pl.Origin.AsVector()) - r // distance r on the inside of each face
	}
	x, ok := solve3(a, b)
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
		tan[f.ID()] = s.TranslateBy(f.Geometry().(geom.Plane).Normal().Scale(r))
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
