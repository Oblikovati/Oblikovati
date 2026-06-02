// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bufio"
	"fmt"
	"io"
	"strconv"

	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Mesh features (M10-F04, PBI-115) wrap imported tessellated geometry (e.g. an STL)
// as a first-class feature with selectable facet/edge/vertex topology. A mesh is
// reference geometry — the feature passes the running solid state through unchanged
// while exposing the mesh for selection, measurement and (later) conversion.

// MeshGeometry is imported tessellated geometry: shared vertices and facets (each an
// ordered loop of vertex indices, typically a triangle).
type MeshGeometry struct {
	Vertices []math.Point3
	Facets   [][]int
}

// ParseSTL reads an ASCII STL stream into a MeshGeometry, welding coincident vertices
// (on a tolerance grid) so facets share vertices/edges. It errors on malformed input,
// naming the offending token. Binary STL is a separate decoder (not yet wired).
func ParseSTL(r io.Reader) (*MeshGeometry, error) {
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)
	g := &MeshGeometry{}
	if err := scanFacets(sc, g); err != nil {
		return nil, err
	}
	if len(g.Facets) == 0 {
		return nil, fmt.Errorf("STL: no facets parsed")
	}
	return g, nil
}

// scanFacets consumes the token stream, welding each facet's vertices into g.
func scanFacets(sc *bufio.Scanner, g *MeshGeometry) error {
	index := map[[3]int64]int{}
	var facet []int
	for sc.Scan() {
		switch sc.Text() {
		case "vertex":
			p, err := readPoint(sc)
			if err != nil {
				return err
			}
			facet = append(facet, weldVertex(g, index, p))
		case "endfacet":
			if len(facet) >= 3 {
				g.Facets = append(g.Facets, facet)
			}
			facet = nil
		}
	}
	return sc.Err()
}

// readPoint reads three whitespace-separated floats following a "vertex" token.
func readPoint(sc *bufio.Scanner) (math.Point3, error) {
	var c [3]float64
	for i := 0; i < 3; i++ {
		if !sc.Scan() {
			return math.Point3{}, fmt.Errorf("STL: vertex truncated at coord %d", i)
		}
		v, err := strconv.ParseFloat(sc.Text(), 64)
		if err != nil {
			return math.Point3{}, fmt.Errorf("STL: bad coordinate %q: %w", sc.Text(), err)
		}
		c[i] = v
	}
	return math.P3(c[0], c[1], c[2]), nil
}

// weldVertex returns the index of p in g, reusing a coincident vertex (1e-6 grid).
func weldVertex(g *MeshGeometry, index map[[3]int64]int, p math.Point3) int {
	const tol = 1e-6
	k := [3]int64{int64(stdmath.Round(p.X / tol)), int64(stdmath.Round(p.Y / tol)), int64(stdmath.Round(p.Z / tol))}
	if i, ok := index[k]; ok {
		return i
	}
	i := len(g.Vertices)
	g.Vertices = append(g.Vertices, p)
	index[k] = i
	return i
}

// MeshVertex/MeshEdge/MeshFace are selectable handles into a mesh's topology.
type MeshVertex struct {
	geom  *MeshGeometry
	index int
}

func (v MeshVertex) Index() int         { return v.index }
func (v MeshVertex) Point() math.Point3 { return v.geom.Vertices[v.index] }

// MeshFace is one facet.
type MeshFace struct {
	geom  *MeshGeometry
	index int
}

// VertexIndices returns the facet's ordered vertex indices; Centroid its center.
func (f MeshFace) VertexIndices() []int { return append([]int(nil), f.geom.Facets[f.index]...) }

func (f MeshFace) Centroid() math.Point3 {
	pts := make([]math.Point3, 0, len(f.geom.Facets[f.index]))
	for _, vi := range f.geom.Facets[f.index] {
		pts = append(pts, f.geom.Vertices[vi])
	}
	return meshCentroid(pts)
}

// MeshEdge is one undirected facet edge.
type MeshEdge struct {
	geom *MeshGeometry
	a, b int
}

func (e MeshEdge) Ends() (int, int) { return e.a, e.b }

// MeshVertices/MeshFaces/MeshEdges enumerate a mesh's selectable topology.
type (
	MeshVertices struct{ geom *MeshGeometry }
	MeshFaces    struct{ geom *MeshGeometry }
	MeshEdges    struct{ geom *MeshGeometry }
)

func (vs *MeshVertices) Count() int            { return len(vs.geom.Vertices) }
func (vs *MeshVertices) Item(i int) MeshVertex { return MeshVertex{geom: vs.geom, index: i} }

func (fs *MeshFaces) Count() int          { return len(fs.geom.Facets) }
func (fs *MeshFaces) Item(i int) MeshFace { return MeshFace{geom: fs.geom, index: i} }

func (es *MeshEdges) Count() int { return len(es.geom.edgeKeys()) }
func (es *MeshEdges) Item(i int) MeshEdge {
	k := es.geom.edgeKeys()[i]
	return MeshEdge{geom: es.geom, a: k[0], b: k[1]}
}

// edgeKeys returns the mesh's distinct undirected facet edges in deterministic order.
func (g *MeshGeometry) edgeKeys() [][2]int {
	seen := map[[2]int]bool{}
	var out [][2]int
	for _, f := range g.Facets {
		for i := range f {
			a, b := f[i], f[(i+1)%len(f)]
			if a > b {
				a, b = b, a
			}
			if k := [2]int{a, b}; !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// MeshFeature wraps imported tessellated geometry as a feature (PBI-115).
type MeshFeature struct {
	geom     *MeshGeometry
	featName string
}

// Geometry returns the imported mesh; Vertices/Edges/Faces expose its topology.
func (m *MeshFeature) Geometry() *MeshGeometry { return m.geom }
func (m *MeshFeature) Vertices() *MeshVertices { return &MeshVertices{geom: m.geom} }
func (m *MeshFeature) Edges() *MeshEdges       { return &MeshEdges{geom: m.geom} }
func (m *MeshFeature) Faces() *MeshFaces       { return &MeshFaces{geom: m.geom} }

// Kind implements [Feature].
func (m *MeshFeature) Kind() string { return "mesh" }

// Recompute passes the running solid state through — a mesh is reference geometry.
func (m *MeshFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// MeshFeatures adds mesh features into the engine.
type MeshFeatures struct{ engine *PartFeatures }

// NewMeshFeatures binds the collection to a feature engine.
func NewMeshFeatures(engine *PartFeatures) *MeshFeatures { return &MeshFeatures{engine: engine} }

// Add wraps imported mesh geometry as a feature.
func (c *MeshFeatures) Add(g *MeshGeometry) *PartFeature {
	mf := &MeshFeature{geom: g, featName: "Mesh"}
	pf := c.engine.Add(mf)
	mf.featName = pf.name
	return pf
}

// MeshFeatureSet groups related mesh features (e.g. one STL's shells) under one node.
type MeshFeatureSet struct {
	name  string
	items []*MeshFeature
}

// NewMeshFeatureSet creates a named, empty mesh feature set.
func NewMeshFeatureSet(name string) *MeshFeatureSet { return &MeshFeatureSet{name: name} }

// Name returns the set's name; Add appends a mesh feature; Count/Item enumerate.
func (s *MeshFeatureSet) Name() string            { return s.name }
func (s *MeshFeatureSet) Add(m *MeshFeature)      { s.items = append(s.items, m) }
func (s *MeshFeatureSet) Count() int              { return len(s.items) }
func (s *MeshFeatureSet) Item(i int) *MeshFeature { return s.items[i] }

// meshCentroid averages a set of points.
func meshCentroid(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}
