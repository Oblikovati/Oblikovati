// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Mesh is a triangle mesh for display/export: positions, per-vertex normals, and
// triangle indices (triples). float64 here; the renderer narrows to float32 at the
// GPU boundary (core/03).
type Mesh struct {
	Positions []math.Point3
	Normals   []math.Vector3
	Indices   []int
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int { return len(m.Indices) / 3 }

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int { return len(m.Positions) }

func (m *Mesh) addVertex(p math.Point3, n math.Vector3) int {
	m.Positions = append(m.Positions, p)
	m.Normals = append(m.Normals, n)
	return len(m.Positions) - 1
}

func (m *Mesh) addTriangle(i, j, k int) { m.Indices = append(m.Indices, i, j, k) }

// Quality controls faceting density via a chordal tolerance: the maximum allowed
// deviation between the facets and the true geometry.
type Quality struct {
	ChordTolerance float64
}

// DefaultQuality returns a reasonable display tolerance.
func DefaultQuality() Quality { return Quality{ChordTolerance: 0.05} }

func (q Quality) tol() float64 {
	if q.ChordTolerance <= 0 {
		return DefaultQuality().ChordTolerance
	}
	return q.ChordTolerance
}

// TessellateEdge facets an edge into a polyline honoring the chordal tolerance.
func TessellateEdge(e *topo.Edge, q Quality) []math.Point3 {
	c := e.Geometry()
	lo, hi := c.Domain()
	params := adaptiveParams(c.PointAt, lo, hi, q.tol())
	pts := make([]math.Point3, len(params))
	for i, t := range params {
		pts[i] = c.PointAt(t)
	}
	return pts
}

// TessellateBody facets every face into one mesh and every edge into a polyline.
func TessellateBody(b *topo.Body, q Quality) (*Mesh, [][]math.Point3) {
	mesh := &Mesh{}
	for _, f := range b.Faces() {
		mergeMesh(mesh, TessellateFace(f, q))
	}
	var edges [][]math.Point3
	for _, e := range b.Edges() {
		edges = append(edges, TessellateEdge(e, q))
	}
	return mesh, edges
}

// TessellateFace facets a face. Planar faces are triangulated exactly from their
// outer boundary (watertight); curved faces are sampled on an adaptive UV grid.
func TessellateFace(f *topo.Face, q Quality) *Mesh {
	mesh := tessellateFaceSurface(f, q)
	if f.Reversed() {
		reverseMesh(mesh) // cut wall: surface normal points into the removed material
	}
	return mesh
}

// tessellateFaceSurface meshes a face by its surface kind, ignoring its sense.
func tessellateFaceSurface(f *topo.Face, q Quality) *Mesh {
	if _, planar := f.Geometry().(geom.Plane); planar {
		return tessellatePlanarFace(f, q)
	}
	return tessellateCurvedFace(f, q)
}

// reverseMesh flips a reversed face's sense (see topo.Face.Reversed): every outward normal
// negates and every triangle's winding reverses, so the face presents its true material side
// to shading and to the divergence-theorem volume (mass-properties orient triangles by these
// per-vertex normals — see meshGeometryProperties).
func reverseMesh(m *Mesh) {
	for i := range m.Normals {
		m.Normals[i] = m.Normals[i].Scale(-1)
	}
	for t := 0; t+2 < len(m.Indices); t += 3 {
		m.Indices[t+1], m.Indices[t+2] = m.Indices[t+2], m.Indices[t+1]
	}
}

// mergeMesh appends src's geometry into dst, offsetting indices.
func mergeMesh(dst, src *Mesh) {
	base := dst.VertexCount()
	dst.Positions = append(dst.Positions, src.Positions...)
	dst.Normals = append(dst.Normals, src.Normals...)
	for _, idx := range src.Indices {
		dst.Indices = append(dst.Indices, base+idx)
	}
}

// adaptiveParams returns the parameter breakpoints (lo…hi inclusive) at which
// eval's chord deviates from the true curve by no more than tol, by recursive
// midpoint subdivision.
func adaptiveParams(eval func(float64) math.Point3, lo, hi, tol float64) []float64 {
	out := []float64{lo}
	subdivide(eval, lo, hi, tol, 0, &out)
	return out
}

func subdivide(eval func(float64) math.Point3, lo, hi, tol float64, depth int, out *[]float64) {
	mid := (lo + hi) / 2
	if depth >= 22 || pointToSegment(eval(mid), eval(lo), eval(hi)) <= tol {
		*out = append(*out, hi)
		return
	}
	subdivide(eval, lo, mid, tol, depth+1, out)
	subdivide(eval, mid, hi, tol, depth+1, out)
}

// pointToSegment returns the distance from p to segment [a, b].
func pointToSegment(p, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	denom := ab.LengthSquared()
	if denom < math.DefaultTolerance {
		return p.DistanceTo(a)
	}
	t := clamp01(a.VectorTo(p).Dot(ab) / denom)
	return p.DistanceTo(a.TranslateBy(ab.Scale(t)))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// gridMesh builds a mesh from a UV breakpoint grid.
func gridMesh(s geom.Surface, us, vs []float64) *Mesh {
	m := &Mesh{}
	idx := make([][]int, len(us))
	for i, u := range us {
		idx[i] = make([]int, len(vs))
		for j, v := range vs {
			idx[i][j] = m.addVertex(s.PointAt(u, v), s.NormalAt(u, v))
		}
	}
	for i := 0; i+1 < len(us); i++ {
		for j := 0; j+1 < len(vs); j++ {
			m.addTriangle(idx[i][j], idx[i+1][j], idx[i+1][j+1])
			m.addTriangle(idx[i][j], idx[i+1][j+1], idx[i][j+1])
		}
	}
	return m
}

// clampSpan replaces infinite domain bounds with a finite window.
func clampSpan(lo, hi float64) (float64, float64) {
	const window = 10
	if stdmath.IsInf(lo, 0) {
		lo = -window
	}
	if stdmath.IsInf(hi, 0) {
		hi = window
	}
	return lo, hi
}
