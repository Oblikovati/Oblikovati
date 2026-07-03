// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
)

// Free-form (sub-D) features (M10-F03) wrap a Catmull–Clark control cage (kernel/subd):
// the feature recomputes the B-rep limit surface from the cage at the current
// subdivision level, so cage edits (moving vertices, creasing edges, changing the
// level) smoothly redeform the body — the heart of sub-D modeling. After editing the
// cage, mark the feature dirty and recompute (same protocol as a parameter edit).

// FreeformBody is the editable sub-D control cage plus its subdivision level. Its
// faces/edges/vertices are the handles the free-form UI manipulates.
type FreeformBody struct {
	cage  subd.Mesh
	level int
}

// Level returns the subdivision level; SetLevel changes it (clamped at 0).
func (b *FreeformBody) Level() int { return b.level }

func (b *FreeformBody) SetLevel(n int) {
	if n < 0 {
		n = 0
	}
	b.level = n
}

// Vertices/Edges/Faces expose the cage topology for selection-based editing.
func (b *FreeformBody) Vertices() *FreeformVertices { return &FreeformVertices{body: b} }
func (b *FreeformBody) Edges() *FreeformEdges       { return &FreeformEdges{body: b} }
func (b *FreeformBody) Faces() *FreeformFaces       { return &FreeformFaces{body: b} }

// MoveVertices translates the selected cage vertices by t (a selection transform).
func (b *FreeformBody) MoveVertices(indices []int, t math.Vector3) {
	for _, i := range indices {
		b.cage.Verts[i] = b.cage.Verts[i].TranslateBy(t)
	}
}

// CreaseEdges sets the crease sharpness on the selected cage edges (1 ⇒ fully sharp).
func (b *FreeformBody) CreaseEdges(edges [][2]int, sharpness float64) {
	for _, e := range edges {
		b.cage.SetCrease(e[0], e[1], sharpness)
	}
}

// FreeformVertex is a handle to one cage vertex.
type FreeformVertex struct {
	body  *FreeformBody
	index int
}

// Index returns the vertex's cage index; Point its current position.
func (v FreeformVertex) Index() int         { return v.index }
func (v FreeformVertex) Point() math.Point3 { return v.body.cage.Verts[v.index] }

// Move translates this vertex by t.
func (v FreeformVertex) Move(t math.Vector3) {
	v.body.cage.Verts[v.index] = v.body.cage.Verts[v.index].TranslateBy(t)
}

// FreeformVertices enumerates a body's cage vertices.
type FreeformVertices struct{ body *FreeformBody }

func (vs *FreeformVertices) Count() int { return len(vs.body.cage.Verts) }
func (vs *FreeformVertices) Item(i int) FreeformVertex {
	return FreeformVertex{body: vs.body, index: i}
}

// FreeformEdge is a handle to one cage edge (an undirected vertex-index pair).
type FreeformEdge struct {
	body *FreeformBody
	a, b int
}

// Ends returns the edge's endpoint indices.
func (e FreeformEdge) Ends() (int, int) { return e.a, e.b }

// Crease sets this edge's sharpness.
func (e FreeformEdge) Crease(sharpness float64) { e.body.cage.SetCrease(e.a, e.b, sharpness) }

// FreeformEdges enumerates a body's cage edges.
type FreeformEdges struct{ body *FreeformBody }

func (es *FreeformEdges) Count() int { return len(es.body.cage.EdgeList()) }
func (es *FreeformEdges) Item(i int) FreeformEdge {
	k := es.body.cage.EdgeList()[i]
	return FreeformEdge{body: es.body, a: k[0], b: k[1]}
}

// FreeformFace is a handle to one cage face.
type FreeformFace struct {
	body  *FreeformBody
	index int
}

// VertexIndices returns the face's ordered cage vertex indices.
func (f FreeformFace) VertexIndices() []int {
	return append([]int(nil), f.body.cage.Faces[f.index]...)
}

// FreeformFaces enumerates a body's cage faces.
type FreeformFaces struct{ body *FreeformBody }

func (fs *FreeformFaces) Count() int              { return len(fs.body.cage.Faces) }
func (fs *FreeformFaces) Item(i int) FreeformFace { return FreeformFace{body: fs.body, index: i} }

// FreeformFeature creates and edits a sub-D free-form body, recomputing its B-rep
// limit surface from the cage at the current subdivision level (PBI-113/114).
type FreeformFeature struct {
	body     *FreeformBody
	featName string
}

// FreeformBody returns the editable control cage handle.
func (f *FreeformFeature) FreeformBody() *FreeformBody { return f.body }

// Kind implements [Feature].
func (f *FreeformFeature) Kind() string { return "freeform" }

// Recompute subdivides the cage to the current level and converts it to a B-rep body.
func (f *FreeformFeature) Recompute(in Input) (Output, error) {
	body := subd.ToBody(subd.SubdivideN(f.body.cage, f.body.level), f.featName)
	return Output{Bodies: appendBody(in.Bodies, body)}, nil
}

// AliasFreeformFeature is an imported Autodesk Alias sub-D body (M17 translation)
// participating as an editable free-form feature; it shares the cage edit/recompute
// behavior, distinguished only by provenance.
type AliasFreeformFeature struct{ FreeformFeature }

// Kind implements [Feature].
func (a *AliasFreeformFeature) Kind() string { return "alias-freeform" }

// FreeformFeatures adds free-form features into the engine.
type FreeformFeatures struct{ engine *PartFeatures }

// NewFreeformFeatures binds the collection to a feature engine.
func NewFreeformFeatures(engine *PartFeatures) *FreeformFeatures {
	return &FreeformFeatures{engine: engine}
}

// AddBox/AddPlane/AddQuadBall start a free-form body from a primitive cage at the
// given subdivision level.
func (c *FreeformFeatures) AddBox(sx, sy, sz float64, level int) *PartFeature {
	return c.add(subd.Box(sx, sy, sz), level)
}

func (c *FreeformFeatures) AddPlane(sx, sy float64, level int) *PartFeature {
	return c.add(subd.Plane(sx, sy), level)
}

func (c *FreeformFeatures) AddQuadBall(radius float64, level int) *PartFeature {
	return c.add(subd.QuadBall(radius), level)
}

// add registers a free-form feature over the given cage.
func (c *FreeformFeatures) add(cage subd.Mesh, level int) *PartFeature {
	ff := &FreeformFeature{body: &FreeformBody{cage: cage.Clone(), level: level}, featName: "Freeform"}
	pf := c.engine.Add(ff)
	ff.featName = pf.name
	return pf
}

// AliasFreeformFeatures adds imported Alias free-form bodies into the engine.
type AliasFreeformFeatures struct{ engine *PartFeatures }

// NewAliasFreeformFeatures binds the collection to a feature engine.
func NewAliasFreeformFeatures(engine *PartFeatures) *AliasFreeformFeatures {
	return &AliasFreeformFeatures{engine: engine}
}

// AddFromCage wraps an imported sub-D cage (vertices + polygon faces) as an editable
// free-form feature.
func (c *AliasFreeformFeatures) AddFromCage(verts []math.Point3, faces [][]int, level int) *PartFeature {
	return c.add(subd.Mesh{Verts: verts, Faces: faces}, level)
}

// add registers an imported Alias free-form feature over the given cage, preserving the
// cage's creases and subdivision level — the seam the serialization restore path uses so a
// round-trip keeps sharp edges (AddFromCage's caller passes a crease-free cage).
func (c *AliasFreeformFeatures) add(cage subd.Mesh, level int) *PartFeature {
	af := &AliasFreeformFeature{FreeformFeature{body: &FreeformBody{cage: cage.Clone(), level: level}, featName: "AliasFreeform"}}
	pf := c.engine.Add(af)
	af.featName = pf.name
	return pf
}
