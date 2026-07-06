// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strconv"
	"strings"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// Placed-mesh display (#1773): a Mesh ▸ Place Mesh reference (#1764/#700) has no B-rep body, so the
// body draw list never sees it and it renders as nothing. It is drawn instead as its own display
// object. The head flattens these items ONCE and caches them on MeshDisplaySignature, so a dense
// scan is not re-flattened every frame — putting it on the per-frame overlay path would reintroduce
// the O(triangles)-per-frame cost #1771 removed.

// meshRefColor is the neutral surface colour a placed mesh reference is shaded with — a light,
// faintly cool grey, so a reference mesh reads as reference rather than as a modelled body.
var meshRefColor = [4]float32{0.72, 0.74, 0.78, 1}

// MeshDrawItems returns one shaded-triangle draw item per visible placed mesh reference on the
// active part. Each facet is flat-shaded — its triangles carry their own vertices and the facet's
// geometric normal — because a scan/visualization mesh has no smoothing groups to average across.
// Empty when the active document is not a part or carries no mesh references. Building a dense
// mesh's item is O(triangles); the head calls this only when MeshDisplaySignature changes.
//
//	items := s.MeshDrawItems() // one renderer.DrawItem per placed STL
func (s *Session) MeshDrawItems() []renderer.DrawItem {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	feats := part.Features()
	var items []renderer.DrawItem
	for i := 0; i < feats.Count(); i++ {
		pf := feats.Item(i)
		if pf.Suppressed() {
			continue
		}
		if mf, ok := pf.Definition().(*feature.MeshFeature); ok {
			items = append(items, meshTriangleItem(mf.Geometry(), uint64(pf.ID())))
		}
	}
	return items
}

// MeshDisplaySignature is a cheap identity for the current placed-mesh SET — the visible mesh
// features' ids — so the head's flatten cache rebuilds only when a mesh is placed, removed or
// suppressed, not on an unrelated edit or an orbit (which leave the set unchanged). It walks the
// feature list without building any geometry. ok is false when there is nothing to display.
func (s *Session) MeshDisplaySignature() (string, bool) {
	part, err := activePart(s)
	if err != nil {
		return "", false
	}
	feats := part.Features()
	var sig strings.Builder
	for i := 0; i < feats.Count(); i++ {
		pf := feats.Item(i)
		if pf.Suppressed() {
			continue
		}
		if _, ok := pf.Definition().(*feature.MeshFeature); ok {
			sig.WriteString(strconv.FormatUint(uint64(pf.ID()), 10))
			sig.WriteByte(';')
		}
	}
	if sig.Len() == 0 {
		return "", false
	}
	return sig.String(), true
}

// meshTriangleItem builds the flat-shaded triangle draw item for one placed mesh geometry.
func meshTriangleItem(g *feature.MeshGeometry, objectID uint64) renderer.DrawItem {
	pos, norm, idx := triangulateFacets(g)
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: pos,
		Normals:   norm,
		Indices:   idx,
		Color:     meshRefColor,
		Roughness: 0.6,
		Opacity:   1,
		Shading:   renderer.ShadeFlat, // a reference mesh has no extracted edges — always lit
		ObjectID:  objectID,
	}
}

// triangulateFacets fan-triangulates every facet into flat-shaded, UNSHARED-vertex streams — each
// triangle carries its own three vertices and the facet's geometric normal, so neighbours never
// average and the faceting reads crisply.
func triangulateFacets(g *feature.MeshGeometry) (pos []math.Point3, norm []math.Vector3, idx []int) {
	tris := 0
	for _, f := range g.Facets {
		if len(f) >= 3 {
			tris += len(f) - 2
		}
	}
	pos = make([]math.Point3, 0, tris*3)
	norm = make([]math.Vector3, 0, tris*3)
	idx = make([]int, 0, tris*3)
	for _, f := range g.Facets {
		for k := 2; k < len(f); k++ { // a triangle contributes one iteration
			a, b, c := g.Vertices[f[0]], g.Vertices[f[k-1]], g.Vertices[f[k]]
			n := faceNormal(a, b, c)
			base := len(pos)
			pos = append(pos, a, b, c)
			norm = append(norm, n, n, n)
			idx = append(idx, base, base+1, base+2)
		}
	}
	return pos, norm, idx
}

// faceNormal is the unit geometric normal of triangle a-b-c by the right-hand rule over its vertex
// winding, or +Z for a degenerate (zero-area) triangle so shading stays defined.
func faceNormal(a, b, c math.Point3) math.Vector3 {
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	l := n.Length()
	if l < 1e-12 {
		return math.V3(0, 0, 1)
	}
	return n.Scale(1 / l)
}
