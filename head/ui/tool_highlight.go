//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/renderer"
)

// Unified tool highlighting: every interactive tool gives the SAME feedback — the selectable
// under the cursor (that the tool's filter would pick) is highlighted in the candidate colour, and
// what the tool has already picked (reported through the app.Picking contract) is highlighted in
// the selection colour. Both run off one per-kind renderer (drawSelectable), so a tool that
// accepts a kind highlights it identically with no head-side per-tool code (ADR-0041). Work planes
// and axes keep their own overlays (planesOverlay/hoveredPlane), but go through the same
// preselect path via hoveredPlane.

// drawSelectable returns the overlay for one selectable in colour, keyed on its kind — the single
// place a selection kind's highlight appearance is defined. A face is both outlined AND tinted
// with a translucent on-top fill, because a face's outline coincides with the model's own edges
// and z-fights into invisibility on its own. Kinds drawn by dedicated overlays (work planes/axes,
// sketch entities) return nil here.
func drawSelectable(sel app.Selectable, color [4]float32) []renderer.DrawItem {
	switch h := sel.(type) {
	case app.ProfileHandle:
		return profileOutline(h, color)
	case app.EdgeHandle:
		return edgeWire([]*topo.Edge{h.Edge}, color)
	case app.FaceHandle:
		return append(edgeWire(h.Face.Edges(), color), faceFill(h, color))
	}
	return nil
}

// edgeWire draws the given edges as overlay lines in colour (the shared edge/face wireframe).
func edgeWire(edges []*topo.Edge, color [4]float32) []renderer.DrawItem {
	acc := &segAccum{}
	for _, e := range edges {
		pts := ops.TessellateEdge(e, ops.DefaultQuality())
		for i := 0; i+1 < len(pts); i++ {
			acc.addSegment(pts[i], pts[i+1])
		}
	}
	if len(acc.pos) == 0 {
		return nil
	}
	return []renderer.DrawItem{lineItem(acc, color)}
}

// highlightSetItems draws the session's add-in highlight sets (#157): each set's references are
// resolved against the live geometry and outlined in the set's colour — a non-selecting emphasis
// the renderer reads every frame, so the highlight follows edits (a lost reference isn't drawn).
func highlightSetItems(s *app.Session) []renderer.DrawItem {
	var items []renderer.DrawItem
	for _, set := range s.HighlightSets().All() {
		rgba := set.Color().Rgba()
		color := [4]float32{rgba.R, rgba.G, rgba.B, rgba.A}
		for _, ref := range set.Refs() {
			if sel, ok := s.ResolveReference(ref); ok {
				items = append(items, drawSelectable(sel, color)...)
			}
		}
	}
	return items
}

// toolHoverHighlight highlights the selectable under the cursor that the active tool would pick,
// in the candidate colour — the same preselect feedback for every tool, through the one per-kind
// renderer.
func toolHoverHighlight(s *app.Session) []renderer.DrawItem {
	if s.ActiveTool() == nil || !native.IsItemHovered() {
		return nil
	}
	filter := s.Selection().Filter()
	if filter == nil {
		return nil
	}
	x, y := viewportCursor()
	sel, ok := s.PickAt(x, y, filter)
	if !ok {
		return nil
	}
	return drawSelectable(sel, chromeTheme.sketchCandidateColor)
}

// faceFill returns a translucent, always-on-top tint of a face's tessellation — the visible part
// of a face highlight (the highlighted face is the front-most hit, so drawing it on top reads
// cleanly without z-fighting the model's own edges). Used for both preselect and selected faces.
func faceFill(fh app.FaceHandle, color [4]float32) renderer.DrawItem {
	mesh := ops.TessellateFace(fh.Face, ops.DefaultQuality())
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: mesh.Positions,
		Normals:   mesh.Normals,
		Indices:   mesh.Indices,
		Color:     color,
		Opacity:   faceFillOpacity,
		OnTop:     true,
	}
}

// faceFillOpacity keeps the face tint translucent so the face's shading still reads through.
const faceFillOpacity = 0.5

// toolSelectedHighlight highlights everything the active tool has picked, in the selection colour —
// the picks come from the uniform app.Picking contract (s.ToolPicks), so every tool highlights its
// selection through the same renderer with no per-tool head code.
func toolSelectedHighlight(s *app.Session) []renderer.DrawItem {
	var items []renderer.DrawItem
	for _, sel := range s.ToolPicks() {
		items = append(items, drawSelectable(sel, chromeTheme.selectionHighlight)...)
	}
	return items
}
