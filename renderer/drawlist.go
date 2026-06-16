// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// Primitive is the topology of a draw item.
type Primitive uint8

const (
	// Triangles: an indexed triangle mesh (shaded surfaces).
	Triangles Primitive = iota
	// Lines: an indexed line list (edge wireframe / overlays).
	Lines
)

// DrawItem is one batch of geometry to draw — geometry-as-data, independent of any
// GPU type. ObjectID is written to the id-buffer for picking; Color is the base
// (albedo) color and Metallic/Roughness/Emissive/Opacity carry the rest of the PBR
// surface (ADR-0022) for the GPU shader — the basic shader uses albedo today and the
// other terms feed a future GGX upgrade.
type DrawItem struct {
	Primitive Primitive
	Positions []math.Point3
	Normals   []math.Vector3
	Indices   []int
	Color     [4]float32
	// Colors, when non-nil, gives a per-vertex color (len == len(Positions)) that
	// overrides the single Color at flatten time — the heatmap/per-vertex-binding path
	// for client graphics. nil keeps the legacy "broadcast Color to every vertex" behavior.
	Colors    [][4]float32
	Metallic  float32
	Roughness float32
	Emissive  [3]float32
	Opacity   float32
	// Shading is how the native pipeline should light this item's faces (flat/PBR/NPR),
	// resolved from the active VisualStyle. Zero value (ShadeNone) is fine for line items.
	Shading Shading
	// Occluder marks a triangle item that writes depth but no color — an invisible occluder
	// used by the hidden-line modes to hide edges behind faces that are not themselves drawn.
	Occluder bool
	// Hidden marks a line item drawn only where it is occluded (reversed depth test) in a
	// dashed style — the hidden-edge half of the wireframe/shaded-with-hidden modes.
	Hidden bool
	// OnTop marks an item that should draw ignoring the depth test (always visible over the
	// model) — the interaction-overlay lane and burn-through markers/labels of client
	// graphics (Inventor's BurnThrough). The viewport routes these to the depth-disabled
	// on-top pass (PBI-067); a headless NullBackend simply records them like any item.
	OnTop bool
	// Biased marks a translucent reference overlay (a work-plane / ground-plane fill) that should
	// render with a small depth bias, so where it is coplanar with solid geometry the solid wins
	// the depth test instead of z-fighting. Display only; routed to the biased tail of the
	// triangle stream (see viewport.Flatten).
	Biased   bool
	ObjectID uint64
}

// Surface is a resolved PBR appearance for one body — what [SurfaceLookup] returns. It is
// the renderer-side value the model's effective Appearance maps onto, keeping the renderer
// independent of the material package.
type Surface struct {
	Albedo    [4]float32
	Metallic  float32
	Roughness float32
	Emissive  [3]float32
	Opacity   float32
}

// SurfaceLookup returns the surface to shade a body with. BuildDrawList calls it per body;
// a nil lookup means "use the neutral default" (the pre-materials look).
type SurfaceLookup func(b *topo.Body) Surface

// defaultSurface is the neutral material used for an un-assigned body — its albedo is the
// pre-materials default gray, so a model with no assignments looks unchanged.
var defaultSurface = Surface{Albedo: defaultSurfaceColor, Metallic: 0, Roughness: 0.6, Opacity: 1}

// TriangleCount / LineCount report the primitive count of an item.
func (d DrawItem) TriangleCount() int {
	if d.Primitive != Triangles {
		return 0
	}
	return len(d.Indices) / 3
}

func (d DrawItem) LineCount() int {
	if d.Primitive != Lines {
		return 0
	}
	return len(d.Indices) / 2
}

// DrawList is the frame's geometry: a flat list of draw items, deterministic in
// body order (the renderer may later sort/batch it without changing semantics).
type DrawList struct {
	Items []DrawItem
}

// Triangles / Lines total the primitives across the list.
func (l DrawList) Triangles() int {
	n := 0
	for _, it := range l.Items {
		n += it.TriangleCount()
	}
	return n
}

func (l DrawList) Lines() int {
	n := 0
	for _, it := range l.Items {
		n += it.LineCount()
	}
	return n
}

// defaultSurfaceColor is the neutral material used until materials land.
var defaultSurfaceColor = [4]float32{0.7, 0.72, 0.75, 1}

// edgeColor is the wireframe color. The document's display settings can override it
// (M16-F07 #643) via [SetEdgeColor]; the head applies the active document's EdgeColor each
// time it (re)builds the styled draw list.
var edgeColor = [4]float32{0.1, 0.1, 0.12, 1}

// defaultEdgeColor is the built-in edge color, restored when a document carries no override.
var defaultEdgeColor = [4]float32{0.1, 0.1, 0.12, 1}

// SetEdgeColor overrides the wireframe/edge color used by subsequently-built draw lists.
func SetEdgeColor(c [4]float32) { edgeColor = c }

// EdgeColor returns the current wireframe/edge color.
func EdgeColor() [4]float32 { return edgeColor }

// DefaultEdgeColor returns the built-in edge color (the head's reset value).
func DefaultEdgeColor() [4]float32 { return defaultEdgeColor }

// outlineColor is the near-black "ink" used for NPR illustration outlines.
var outlineColor = [4]float32{0.04, 0.04, 0.06, 1}

// BuildDrawList turns the visible scene bodies into a draw list at the given quality in the
// default Shaded-with-Edges style. Bodies outside the view are culled.
func BuildDrawList(bodies []*topo.Body, cam scene.Camera, q ops.Quality, lookup SurfaceLookup) DrawList {
	return BuildDrawListStyled(bodies, cam, q, lookup, ShadedWithEdges)
}

// BuildDrawListStyled is BuildDrawList honoring a visual style. The style resolves to a
// [PassSet] (PassSetFor): a body contributes a shaded triangle item when the style draws
// faces, and/or an edge line item when it draws edges, each tagged with the body's object id
// for picking. The triangle item carries the style's [Shading] so the native pipeline can
// pick the shader (flat/PBR/NPR); hidden-line removal of edges (EdgesVisible*) is applied by
// the viewport pass, so at the draw-list level the occluded-edge styles still emit every
// edge segment and the pass classifies them.
func BuildDrawListStyled(bodies []*topo.Body, cam scene.Camera, q ops.Quality, lookup SurfaceLookup, style VisualStyle) DrawList {
	pass := PassSetFor(style)
	dashWorld := cam.WorldPerPixel() * hiddenDashPixels
	var items []DrawItem
	for _, b := range bodies {
		if !visible(cam, b.RangeBox()) {
			continue
		}
		mesh, _ := ops.TessellateBody(b, q)
		// Crease-angle smooth shading + tangent-edge suppression (display only): a loft/sweep skin
		// is built as many planar facets, which flat-shade into stripes with a web of facet lines.
		// Average normals across facets meeting below the crease angle, and drop the tangent edges
		// between them, so the skin reads as one smooth surface while genuine sharp edges stay
		// crisp. Mass properties keep the raw per-face mesh (BodyGeometryProperties calls
		// TessellateBody directly), so volume/orientation are unaffected.
		mesh.Normals = ops.SmoothShadeNormals(mesh, ops.DefaultCreaseAngle())
		edges := ops.VisibleEdges(b, q, ops.DefaultCreaseAngle())
		items = appendBodyItems(items, b.ID(), mesh, edges, surfaceFor(b, lookup), pass, dashWorld)
	}
	return DrawList{Items: items}
}

// hiddenDashPixels is the on-screen dash period (in pixels) for occluded edges; the dash is
// rendered as CPU-split geometry so the GPU line pipeline needs no dash pattern.
const hiddenDashPixels = 7.0

// appendBodyItems appends one body's draw items for the resolved pass: shaded faces, or a
// depth-only occluder when the mode hides edges but does not draw faces; the solid edge set
// (depth-tested so only the visible parts show); and, for the with-hidden modes, a dashed
// occluded-edge set drawn with the reversed depth test.
func appendBodyItems(items []DrawItem, id uint64, mesh *ops.Mesh, edges [][]math.Point3,
	surf Surface, pass PassSet, dashWorld float64,
) []DrawItem {
	if mesh.TriangleCount() > 0 {
		switch {
		case pass.Faces != ShadeNone:
			items = append(items, triangleItem(id, mesh, surf, pass.Faces))
		case pass.HidesEdges():
			items = append(items, occluderItem(id, mesh)) // hide edges behind unseen faces
		}
	}
	if pass.Edges != EdgesNone {
		if line := lineItem(id, edges); line != nil {
			items = append(items, *line) // depth-tested ⇒ shows the visible portions
		}
	}
	if pass.Edges == EdgesVisiblePlusHidden {
		if dash := dashedHiddenItem(id, edges, dashWorld); dash != nil {
			items = append(items, *dash)
		}
	}
	if pass.Outline {
		if o := outlineItem(id, edges); o != nil {
			items = append(items, *o) // dark "ink" outline over NPR faces (depth-tested ⇒ visible only)
		}
	}
	return items
}

// surfaceFor returns the body's resolved surface, or the neutral default when no lookup is
// supplied (headless tests, or before materials are wired).
func surfaceFor(b *topo.Body, lookup SurfaceLookup) Surface {
	if lookup == nil {
		return defaultSurface
	}
	return lookup(b)
}

// triangleItem builds the shaded surface item for a body's mesh with its PBR surface and the
// active style's shading mode (which the native pipeline reads to pick the face shader).
func triangleItem(objectID uint64, mesh *ops.Mesh, s Surface, shading Shading) DrawItem {
	return DrawItem{
		Primitive: Triangles,
		Positions: mesh.Positions,
		Normals:   mesh.Normals,
		Indices:   mesh.Indices,
		Color:     s.Albedo,
		Metallic:  s.Metallic,
		Roughness: s.Roughness,
		Emissive:  s.Emissive,
		Opacity:   s.Opacity,
		Shading:   shading,
		ObjectID:  objectID,
	}
}

// lineItem builds the wireframe item from a body's edge polylines, or nil if there
// are no edges.
func lineItem(objectID uint64, edges [][]math.Point3) *DrawItem {
	return coloredLineItem(objectID, edges, edgeColor)
}

// outlineItem builds the NPR "ink" outline item — the body's edges in a near-black color,
// composited over the stylized faces to give the illustration outline.
func outlineItem(objectID uint64, edges [][]math.Point3) *DrawItem {
	return coloredLineItem(objectID, edges, outlineColor)
}

// coloredLineItem builds a line item from a body's edge polylines in the given color, or nil
// if there are no edges.
func coloredLineItem(objectID uint64, edges [][]math.Point3, color [4]float32) *DrawItem {
	item := DrawItem{Primitive: Lines, Color: color, ObjectID: objectID}
	for _, poly := range edges {
		base := len(item.Positions)
		item.Positions = append(item.Positions, poly...)
		for i := 0; i+1 < len(poly); i++ {
			item.Indices = append(item.Indices, base+i, base+i+1)
		}
	}
	if len(item.Indices) == 0 {
		return nil
	}
	return &item
}

// hiddenEdgeColor is the dim color of dashed occluded edges (distinct from the solid edge
// color so hidden lines read as hidden).
var hiddenEdgeColor = [4]float32{0.45, 0.46, 0.52, 1}

// occluderItem builds a depth-only triangle item for a body's mesh: it writes depth (to hide
// edges behind it) but no color, so the faces themselves are not seen.
func occluderItem(objectID uint64, mesh *ops.Mesh) DrawItem {
	return DrawItem{
		Primitive: Triangles,
		Positions: mesh.Positions,
		Normals:   mesh.Normals,
		Indices:   mesh.Indices,
		Occluder:  true,
		ObjectID:  objectID,
	}
}

// dashedHiddenItem builds the dashed occluded-edge line item: every edge segment split into
// on/off dashes of dashWorld length, drawn (by the native pipeline) only where it is behind
// geometry. Returns nil if there are no edges.
func dashedHiddenItem(objectID uint64, edges [][]math.Point3, dashWorld float64) *DrawItem {
	item := DrawItem{Primitive: Lines, Color: hiddenEdgeColor, Hidden: true, ObjectID: objectID}
	for _, poly := range edges {
		for i := 0; i+1 < len(poly); i++ {
			appendDashes(&item, poly[i], poly[i+1], dashWorld)
		}
	}
	if len(item.Indices) == 0 {
		return nil
	}
	return &item
}

// appendDashes appends the "on" dash sub-segments of a→b (period = 2·dashWorld) to a line
// item. With no usable scale (dashWorld ≤ 0) it appends the segment solid, so hidden edges
// still show.
func appendDashes(item *DrawItem, a, b math.Point3, dashWorld float64) {
	seg := a.VectorTo(b)
	length := seg.Length()
	if length == 0 {
		return
	}
	if dashWorld <= 0 {
		base := len(item.Positions)
		item.Positions = append(item.Positions, a, b)
		item.Indices = append(item.Indices, base, base+1)
		return
	}
	dir := seg.Scale(1 / length)
	for start := 0.0; start < length; start += dashWorld * 2 {
		end := start + dashWorld
		if end > length {
			end = length
		}
		base := len(item.Positions)
		item.Positions = append(item.Positions, a.TranslateBy(dir.Scale(start)), a.TranslateBy(dir.Scale(end)))
		item.Indices = append(item.Indices, base, base+1)
	}
}

// visible reports whether any corner of a bounding box is in front of the camera —
// a cheap frustum-front cull (a full frustum test is a later optimization).
func visible(cam scene.Camera, box math.Box) bool {
	if box.IsEmpty() {
		return false
	}
	forward := cam.Forward()
	for _, corner := range box.Corners() {
		if cam.Eye.VectorTo(corner).Dot(forward) > 0 {
			return true
		}
	}
	return false
}
