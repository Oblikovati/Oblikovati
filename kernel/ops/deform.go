// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DeformBody returns a copy of b with every vertex moved by the non-affine map fn, preserving
// the exact combinatorial topology. Unlike [TransformBody] (which carries an affine matrix on
// curves/surfaces), DeformBody is for POLYHEDRAL bodies: every edge is rebuilt as a straight
// segment between its moved endpoints and every face's plane is re-fitted through its moved
// loop. It is the engine behind sheet-metal develop/refold, where a bend is unrolled or rolled
// by a cylindrical point map — any cut made while flat moves with its vertices, so it is carried
// through the bend without a separate entity map.
//
// fn must be applied consistently to coincident points (it is a pure function of position), so
// shared vertices stay welded. Faces that are badly non-planar after the map yield a degenerate
// plane error — subdivide the bend finely enough (small facet steps) that each face stays
// near-planar, exactly as the faceted bend arc already is.
func DeformBody(b *topo.Body, fn func(math.Point3) math.Point3, derive func(topo.Lineage) topo.Lineage) (*topo.Body, error) {
	bld := topo.NewBuilder(b.IsSolid(), derive(b.Lineage()))

	verts := make(map[*topo.Vertex]*topo.Vertex, len(b.Vertices()))
	for _, v := range b.Vertices() {
		verts[v] = bld.AddVertex(fn(v.Point()), derive(v.Lineage()))
	}
	edges := make(map[*topo.Edge]*topo.Edge, len(b.Edges()))
	for _, e := range b.Edges() {
		s, t := verts[e.StartVertex()], verts[e.EndVertex()]
		if s.Point().DistanceTo(t.Point()) < deformTol {
			return nil, fmt.Errorf("ops.DeformBody: edge %d collapsed to a point under the map", e.ID())
		}
		edges[e] = bld.AddEdge(geom.NewLineSegment(s.Point(), t.Point()), s, t, derive(e.Lineage()))
	}
	for _, f := range b.Faces() {
		if err := deformFaceInto(bld, f, edges, derive); err != nil {
			return nil, err
		}
	}
	return bld.Build(), nil
}

// deformTol is the minimum moved-edge length below which the map is treated as collapsing an
// edge (a degenerate result the boolean kernel cannot use).
const deformTol = 1e-9 // tol:calibrated — collapse guard used as BOTH a moved-edge length and a degenerate-normal magnitude; a clean split is deferred

// deformFaceInto clones one face with a plane re-fitted through its moved outer loop, preserving
// the face's material sense (a reversed cut/bore wall must stay reversed or its tessellation
// winds inward and the divergence-theorem volume flips on it).
func deformFaceInto(bld *topo.Builder, f *topo.Face, edges map[*topo.Edge]*topo.Edge, derive func(topo.Lineage) topo.Lineage) error {
	plane, err := refitPlane(f, edges)
	if err != nil {
		return fmt.Errorf("ops.DeformBody: face %d: %w", f.ID(), err)
	}
	if f.Reversed() {
		bld.AddReversedFace(plane, derive(f.Lineage()), loopSpecs(f, edges, false)...)
		return nil
	}
	bld.AddFace(plane, derive(f.Lineage()), loopSpecs(f, edges, false)...)
	return nil
}

// refitPlane fits a plane through a face's moved outer-loop vertices. The normal follows the
// loop winding (Newell's method) so it stays consistent with the preserved loop orientation,
// and the origin is the loop's centroid. It errors when the moved loop is degenerate (all
// points collinear ⇒ zero-area normal).
func refitPlane(f *topo.Face, edges map[*topo.Edge]*topo.Edge) (geom.Plane, error) {
	pts := movedOuterLoopPoints(f, edges)
	if len(pts) < 3 {
		return geom.Plane{}, fmt.Errorf("outer loop has %d moved vertices, need ≥3", len(pts))
	}
	var n math.Vector3
	var c math.Vector3
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		n = n.Add(math.V3((a.Y-b.Y)*(a.Z+b.Z), (a.Z-b.Z)*(a.X+b.X), (a.X-b.X)*(a.Y+b.Y)))
		c = c.Add(a.AsVector())
	}
	if n.Length() < deformTol {
		return geom.Plane{}, fmt.Errorf("moved outer loop is degenerate (zero-area)")
	}
	origin := math.P3(c.X/float64(len(pts)), c.Y/float64(len(pts)), c.Z/float64(len(pts)))
	return geom.NewPlane(origin, n)
}

// movedOuterLoopPoints walks a face's outer loop and returns its vertices' positions on the
// already-moved (cloned) edges, in loop order.
func movedOuterLoopPoints(f *topo.Face, edges map[*topo.Edge]*topo.Edge) []math.Point3 {
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		uses := l.EdgeUses()
		pts := make([]math.Point3, 0, len(uses))
		for _, u := range uses {
			clone := edges[u.Edge()]
			v := clone.StartVertex()
			if u.Reversed() {
				v = clone.EndVertex()
			}
			pts = append(pts, v.Point())
		}
		return pts
	}
	return nil
}
