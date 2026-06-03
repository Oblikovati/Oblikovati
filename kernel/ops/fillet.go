// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
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
	fils := make([]edgeFillet, 0, len(edgeKeys))
	for _, k := range edgeKeys {
		e, ok := body.FindEdgeByKey(k)
		if !ok {
			return nil, fmt.Errorf("fillet: edge reference lost: %x", k)
		}
		ef, err := computeEdgeFillet(body, e, r)
		if err != nil {
			return nil, err
		}
		fils = append(fils, ef)
	}
	res := assembleBody(filletResultFaces(body, fils), "fillet")
	if rep := Validate(res); !rep.Valid || !res.IsSolid() {
		return nil, fmt.Errorf("fillet: result is not a valid solid %v", rep.Issues)
	}
	return res, nil
}

// corner is one rounded end of a filleted edge: the cylinder centre at that end, the tangent
// points on faces a/b, and the arc midpoint (the cylinder point nearest the sharp corner).
type corner struct {
	a, b    *topo.Face
	cen     math.Point3 // cylinder centre at this end
	ta, tb  math.Point3
	mid     math.Point3
	endFace *topo.Face
	vertex  *topo.Vertex
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

// computeEdgeFillet solves the rolling-ball geometry for one convex straight edge.
func computeEdgeFillet(body *topo.Body, e *topo.Edge, r float64) (edgeFillet, error) {
	a, b, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		return edgeFillet{}, err
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return edgeFillet{}, fmt.Errorf("fillet: degenerate edge")
	}
	off := nA.Add(nB).Scale(-r / (1 + nA.Dot(nB))) // centre offset from the edge into the solid
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	if !PointInsideBody(body, mid.TranslateBy(off)) { // convex ⇒ the rolling-ball centre is inside
		return edgeFillet{}, fmt.Errorf("fillet: edge is not convex (only convex edges are supported)")
	}
	c0, err := cornerAt(e.StartVertex(), a, b, nA, nB, off, r)
	if err != nil {
		return edgeFillet{}, err
	}
	c1, err := cornerAt(e.EndVertex(), a, b, nA, nB, off, r)
	if err != nil {
		return edgeFillet{}, err
	}
	cyl, err := geom.NewCylinder(c0.cen, axis.AsVector(), r)
	if err != nil {
		return edgeFillet{}, err
	}
	return edgeFillet{a: a, b: b, cyl: cyl, c0: c0, c1: c1, edge: e}, nil
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

// cornerAt solves a fillet corner at vertex v: the centre (v + off), the tangent points on
// the two faces, the arc midpoint, and the end face (the face at v other than a/b).
func cornerAt(v *topo.Vertex, a, b *topo.Face, nA, nB math.Vector3, off math.Vector3, r float64) (corner, error) {
	c := v.Point().TranslateBy(off)
	end := endFaceAt(v, a, b)
	if end == nil {
		return corner{}, fmt.Errorf("fillet: edge endpoint has no end face to round")
	}
	toward, err := math.UnitVector3FromVector(c.VectorTo(v.Point()))
	if err != nil {
		return corner{}, fmt.Errorf("fillet: degenerate corner")
	}
	return corner{
		a: a, b: b, endFace: end, vertex: v, cen: c,
		ta:  c.TranslateBy(nA.Scale(r)),
		tb:  c.TranslateBy(nB.Scale(r)),
		mid: c.TranslateBy(toward.AsVector().Scale(r)),
	}, nil
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
