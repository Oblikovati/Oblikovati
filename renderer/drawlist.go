// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/scene"
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
// shading color.
type DrawItem struct {
	Primitive Primitive
	Positions []math.Point3
	Normals   []math.Vector3
	Indices   []int
	Color     [4]float32
	ObjectID  uint64
}

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

// edgeColor is the wireframe color.
var edgeColor = [4]float32{0.1, 0.1, 0.12, 1}

// BuildDrawList is the pure function that turns the visible scene bodies into a draw
// list at the given quality: each visible body contributes a shaded triangle item
// (its tessellation) and a wireframe line item (its edges), tagged with the body's
// object id for picking. Bodies outside the view are culled.
func BuildDrawList(bodies []*topo.Body, cam scene.Camera, q ops.Quality) DrawList {
	var items []DrawItem
	for _, b := range bodies {
		if !visible(cam, b.RangeBox()) {
			continue
		}
		mesh, edges := ops.TessellateBody(b, q)
		if mesh.TriangleCount() > 0 {
			items = append(items, triangleItem(b.ID(), mesh))
		}
		if line := lineItem(b.ID(), edges); line != nil {
			items = append(items, *line)
		}
	}
	return DrawList{Items: items}
}

// triangleItem builds the shaded surface item for a body's mesh.
func triangleItem(objectID uint64, mesh *ops.Mesh) DrawItem {
	return DrawItem{
		Primitive: Triangles,
		Positions: mesh.Positions,
		Normals:   mesh.Normals,
		Indices:   mesh.Indices,
		Color:     defaultSurfaceColor,
		ObjectID:  objectID,
	}
}

// lineItem builds the wireframe item from a body's edge polylines, or nil if there
// are no edges.
func lineItem(objectID uint64, edges [][]math.Point3) *DrawItem {
	item := DrawItem{Primitive: Lines, Color: edgeColor, ObjectID: objectID}
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
