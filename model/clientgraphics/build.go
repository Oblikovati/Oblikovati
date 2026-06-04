// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"github.com/Oblikovati/api/types"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/renderer"
	"github.com/Oblikovati/oblikovati/scene"
)

// Build turns every visible group in the store into frame geometry and labels: a flat
// list of renderer.DrawItem (triangle meshes with resolved per-vertex colors, line lists,
// and expanded point glyphs) plus the world-anchored text labels the UI head draws.
// Overlay-lane primitives are marked OnTop (drawn over the model). The camera supplies the
// billboard basis and pixel scale for screen-constant point glyphs.
func (s *Store) Build(cam scene.Camera) ([]renderer.DrawItem, []Label) {
	bb := newBillboard(cam)
	wpp := cam.WorldPerPixel()
	var items []renderer.DrawItem
	var labels []Label
	for _, g := range s.Groups() {
		if !g.visible {
			continue
		}
		for i := range g.nodes {
			items, labels = buildNode(items, labels, &g.nodes[i], g.lane, bb, wpp)
		}
	}
	return items, labels
}

// buildNode appends one node's primitives (placed by its transform, gated by its
// visibility) to the running geometry and labels.
func buildNode(items []renderer.DrawItem, labels []Label, n *Node, lane Lane, bb billboard, wpp float64) ([]renderer.DrawItem, []Label) {
	if n.Visible != nil && !*n.Visible {
		return items, labels
	}
	for i := range n.Primitives {
		p := placedPrimitive(&n.Primitives[i], n)
		if p.Kind == types.GraphicsText {
			labels = append(labels, Label{Anchor: p.Anchor, Text: p.Text, Color: p.Color, FontSize: p.FontSize})
			continue
		}
		if item, ok := buildPrimitive(p, lane, bb, wpp); ok {
			items = append(items, item)
		}
	}
	return items, labels
}

// placedPrimitive returns a copy of the primitive with the node transform applied to its
// coordinates, normals and text anchor (identity nodes return the primitive unchanged).
func placedPrimitive(p *Primitive, n *Node) Primitive {
	if !n.HasTransform {
		return *p
	}
	out := *p
	out.Coords = transformPoints(n.Transform, p.Coords)
	out.Normals = transformVectors(n.Transform, p.Normals)
	out.Anchor = n.Transform.TransformPoint(p.Anchor)
	return out
}

// buildPrimitive converts one non-text primitive into a draw item; ok is false for a kind
// that produces no geometry (e.g. an empty primitive).
func buildPrimitive(p Primitive, lane Lane, bb billboard, wpp float64) (renderer.DrawItem, bool) {
	onTop := p.OnTop || lane == LaneOverlay
	switch p.Kind {
	case types.GraphicsPoints:
		return pointItem(p, bb, wpp, onTop)
	case types.GraphicsLines:
		return lineItem(p, p.Indices, onTop)
	case types.GraphicsLineStrip:
		return lineItem(p, stripLineIndices(len(p.Coords)), onTop)
	case types.GraphicsTriangles:
		return triangleItem(p, p.Indices, onTop)
	case types.GraphicsTriangleStrip:
		return triangleItem(p, stripTriangleIndices(len(p.Coords)), onTop)
	default:
		return renderer.DrawItem{}, false
	}
}

// triangleItem builds a shaded triangle draw item, resolving per-vertex colors from the
// color set, scalars+mapper, or the overall color.
func triangleItem(p Primitive, indices []int, onTop bool) (renderer.DrawItem, bool) {
	if len(p.Coords) == 0 || len(indices) == 0 {
		return renderer.DrawItem{}, false
	}
	item := renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: p.Coords,
		Normals:   p.Normals,
		Indices:   indices,
		Color:     applyOpacity(p.Color, p.Opacity),
		Opacity:   opacityOf(p.Color, p.Opacity),
		OnTop:     onTop,
	}
	if colors := vertexColors(p); colors != nil {
		item.Colors = colors
	}
	return item, true
}

// lineItem builds a line-list draw item from the given index pairs.
func lineItem(p Primitive, indices []int, onTop bool) (renderer.DrawItem, bool) {
	if len(p.Coords) == 0 || len(indices) == 0 {
		return renderer.DrawItem{}, false
	}
	item := renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: p.Coords,
		Indices:   indices,
		Color:     applyOpacity(p.Color, p.Opacity),
		OnTop:     onTop,
	}
	if colors := vertexColors(p); colors != nil {
		item.Colors = colors
	}
	return item, true
}

// pointItem expands a points primitive into a line-list draw item of glyph segments.
func pointItem(p Primitive, bb billboard, wpp float64, onTop bool) (renderer.DrawItem, bool) {
	segs := pointGlyphs(p, bb, wpp)
	if len(segs) == 0 {
		return renderer.DrawItem{}, false
	}
	item := renderer.DrawItem{Primitive: renderer.Lines, Color: applyOpacity(p.Color, p.Opacity), OnTop: onTop}
	for _, s := range segs {
		base := len(item.Positions)
		item.Positions = append(item.Positions, s[0], s[1])
		item.Indices = append(item.Indices, base, base+1)
	}
	return item, true
}

// vertexColors resolves a primitive's per-vertex color array, or nil to fall back to the
// single broadcast Color. Explicit Colors win; otherwise Scalars through a ColorMapper.
func vertexColors(p Primitive) [][4]float32 {
	if p.ColorBinding == types.GraphicsColorOverall {
		return nil
	}
	if len(p.Colors) == len(p.Coords) && len(p.Colors) > 0 {
		return withOpacity(p.Colors, p.Opacity)
	}
	if p.Mapper != nil && len(p.Scalars) == len(p.Coords) && len(p.Scalars) > 0 {
		out := make([][4]float32, len(p.Scalars))
		for i, v := range p.Scalars {
			out[i] = applyOpacity(p.Mapper.At(v), p.Opacity)
		}
		return out
	}
	return nil
}

// withOpacity returns colors with the primitive opacity folded into alpha (or the input
// when opacity is unset).
func withOpacity(colors [][4]float32, opacity float32) [][4]float32 {
	if opacity <= 0 {
		return colors
	}
	out := make([][4]float32, len(colors))
	for i, c := range colors {
		out[i] = applyOpacity(c, opacity)
	}
	return out
}

// applyOpacity returns color with alpha replaced by opacity when opacity is set (>0).
func applyOpacity(color [4]float32, opacity float32) [4]float32 {
	if opacity <= 0 {
		return color
	}
	return [4]float32{color[0], color[1], color[2], opacity}
}

// opacityOf returns the triangle item's Opacity field (the primitive opacity, else the
// color's alpha) so translucent meshes blend.
func opacityOf(color [4]float32, opacity float32) float32 {
	if opacity > 0 {
		return opacity
	}
	return color[3]
}

// stripLineIndices builds consecutive-pair indices for a line strip of n vertices.
func stripLineIndices(n int) []int {
	if n < 2 {
		return nil
	}
	out := make([]int, 0, (n-1)*2)
	for i := 0; i+1 < n; i++ {
		out = append(out, i, i+1)
	}
	return out
}

// stripTriangleIndices builds triangle indices for a triangle strip of n vertices,
// alternating winding so the mesh stays consistently oriented.
func stripTriangleIndices(n int) []int {
	if n < 3 {
		return nil
	}
	out := make([]int, 0, (n-2)*3)
	for i := 0; i+2 < n; i++ {
		if i%2 == 0 {
			out = append(out, i, i+1, i+2)
		} else {
			out = append(out, i+1, i, i+2)
		}
	}
	return out
}

// transformPoints applies a transform to a copy of pts.
func transformPoints(t math.Matrix4, pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = t.TransformPoint(p)
	}
	return out
}

// transformVectors applies a transform's linear part to a copy of vecs.
func transformVectors(t math.Matrix4, vecs []math.Vector3) []math.Vector3 {
	out := make([]math.Vector3, len(vecs))
	for i, v := range vecs {
		out[i] = t.TransformVector(v)
	}
	return out
}
