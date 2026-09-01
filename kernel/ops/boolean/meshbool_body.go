// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The OUT half of the exact-boolean adapter: turn a meshbool.Boolean result (a
// watertight triangle soup) back into a topo.Body. meshbool.MergeFaces first
// recovers the maximal planar faces (with holes); this builds one shared vertex per
// distinct position, one shared edge per distinct vertex pair, and one planar face
// per merged region (outer loop + hole loops — which subd.ToBody cannot express).
// The exact rational coordinates round to float64 here, the single rounding the
// ADR-0052 design defers to output. PRECONDITION: soup is a closed solid result.
func soupToBody(soup [][3]meshbool.Point, feat string) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	vb := &vertexBank{bld: bld, feat: feat, byKey: map[string]*topo.Vertex{}}
	eb := &edgeBank{bld: bld, feat: feat, byKey: map[[2]uint64]*topo.Edge{}}
	for fi, f := range meshbool.MergeFaces(soup) {
		loops := []topo.LoopSpec{buildLoopSpec(vb, eb, f.Outer, true)}
		for _, hole := range f.Holes {
			loops = append(loops, buildLoopSpec(vb, eb, hole, false))
		}
		bld.AddFace(planeThrough(f.Outer), topo.NewLineage(topo.Tok(feat, "face", fi)), loops...)
	}
	return bld.Build()
}

// booleanViaMeshbool computes `a op b` through the exact mesh-arrangement core
// (ADR-0052): tessellate both operands to soups, run meshbool.Boolean, and rebuild
// a body from the watertight result. The result is exact-volume and watertight but
// FACETED (#2153), so it is used only as a robustness fallback (booleanGeneral),
// never in preference to a valid analytic/planar result.
func booleanViaMeshbool(a, b *topo.Body, op meshbool.Op, q Quality, feat string) *topo.Body {
	return soupToBody(meshbool.Boolean(bodyToSoup(a, q), bodyToSoup(b, q), op), feat)
}

// toMeshboolOp maps a part-feature operation to the mesh-arrangement operation, and
// reports false for operations (NewBody, NewBody-like) that are not a set operation.
func toMeshboolOp(op PartFeatureOperation) (meshbool.Op, bool) {
	switch op {
	case Join:
		return meshbool.Union, true
	case Cut:
		return meshbool.Difference, true
	case Intersect:
		return meshbool.Intersection, true
	default:
		return 0, false
	}
}

// buildLoopSpec creates the oriented edge uses for one boundary loop, sharing
// vertices and edges through the banks.
func buildLoopSpec(vb *vertexBank, eb *edgeBank, loop []meshbool.Point, outer bool) topo.LoopSpec {
	n := len(loop)
	uses := make([]topo.Use, n)
	for i := range n {
		e, reversed := eb.get(vb, loop[i], loop[(i+1)%n])
		uses[i] = topo.Use{Edge: e, Reversed: reversed}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// vertexBank returns one shared topo.Vertex per exact position.
type vertexBank struct {
	bld   *topo.Builder
	feat  string
	byKey map[string]*topo.Vertex
	n     int
}

func (vb *vertexBank) get(p meshbool.Point) *topo.Vertex {
	k := pointKey(p)
	if v, ok := vb.byKey[k]; ok {
		return v
	}
	v := vb.bld.AddVertex(p.Round(), topo.NewLineage(topo.Tok(vb.feat, "vertex", vb.n)))
	vb.byKey[k] = v
	vb.n++
	return v
}

// edgeBank returns one shared topo.Edge per undirected vertex pair, plus whether the
// requested direction runs against the edge's stored orientation.
type edgeBank struct {
	bld   *topo.Builder
	feat  string
	byKey map[[2]uint64]*topo.Edge
	n     int
}

func (eb *edgeBank) get(vb *vertexBank, pa, pb meshbool.Point) (*topo.Edge, bool) {
	a, b := vb.get(pa), vb.get(pb)
	reversed := a.ID() > b.ID()
	v0, v1, p0, p1 := a, b, pa, pb
	if reversed {
		v0, v1, p0, p1 = b, a, pb, pa
	}
	key := [2]uint64{v0.ID(), v1.ID()}
	if e, ok := eb.byKey[key]; ok {
		return e, reversed
	}
	e := eb.bld.AddEdge(geom.NewLineSegment(p0.Round(), p1.Round()), v0, v1, topo.NewLineage(topo.Tok(eb.feat, "edge", eb.n)))
	eb.byKey[key] = e
	eb.n++
	return e, reversed
}

// planeThrough fits a plane through a face's outer loop (Newell normal at the
// centroid), falling back to +Z for a degenerate loop.
func planeThrough(loop []meshbool.Point) geom.Surface {
	pts := make([]math.Point3, len(loop))
	for i, p := range loop {
		pts[i] = p.Round()
	}
	p, err := geom.NewPlane(loopMean(pts), newellNormalOf(pts))
	if err != nil {
		p, _ = geom.NewPlane(loopMean(pts), math.V3(0, 0, 1))
	}
	return p
}

func loopMean(pts []math.Point3) math.Point3 {
	var x, y, z float64
	for _, p := range pts {
		x, y, z = x+p.X, y+p.Y, z+p.Z
	}
	n := float64(len(pts))
	return math.P3(x/n, y/n, z/n)
}

func newellNormalOf(pts []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	n := len(pts)
	for i := range n {
		cur, nxt := pts[i], pts[(i+1)%n]
		nx += (cur.Y - nxt.Y) * (cur.Z + nxt.Z)
		ny += (cur.Z - nxt.Z) * (cur.X + nxt.X)
		nz += (cur.X - nxt.X) * (cur.Y + nxt.Y)
	}
	return math.V3(nx, ny, nz)
}

func pointKey(p meshbool.Point) string {
	return p.X.RatString() + "|" + p.Y.RatString() + "|" + p.Z.RatString()
}
