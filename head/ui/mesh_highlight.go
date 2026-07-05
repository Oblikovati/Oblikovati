//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

// meshSelection is the ≤1-method view selectedMeshFacetOverlay needs — the current selection — so the
// highlight does not couple head/ui to the whole *app.Session (archguard ratchet).
type meshSelection interface {
	Selection() *app.Selection
}

// selectedMeshFacetOverlay highlights the picked facet of a placed mesh reference (#1776): the facet
// as an on-top triangle in the selection colour, so a mesh pick reads the same way a face/edge pick
// does. Nil when the selection is not a mesh facet.
func selectedMeshFacetOverlay(sel meshSelection) []renderer.DrawItem {
	h, ok := sel.Selection().First().(app.MeshFaceHandle)
	if !ok {
		return nil
	}
	verts := h.Mesh.Geometry().Vertices
	loop := h.Face().VertexIndices()
	if len(loop) < 3 {
		return nil
	}
	pos := make([]math.Point3, 0, len(loop))
	for _, vi := range loop {
		pos = append(pos, verts[vi])
	}
	idx := make([]int, 0, (len(pos)-2)*3)
	for k := 2; k < len(pos); k++ { // fan-triangulate a polygon facet; a triangle is one fan
		idx = append(idx, 0, k-1, k)
	}
	return []renderer.DrawItem{{
		Primitive: renderer.Triangles,
		Positions: pos,
		Indices:   idx,
		Color:     chromeTheme.selectionHighlight,
		Opacity:   1,
		OnTop:     true, // depth-disabled so the picked facet always shows, no z-fight with the mesh
	}}
}
