//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
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
		return appendProfileFill(profileOutline(h, color), h, color)
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
		pts := tessellate.TessellateEdge(e, ops.DefaultQuality())
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

// appendProfileFill adds a translucent, always-on-top tint of a sketch region (its outer loop minus
// its holes, ear-clipped) to items — the filled part of a region highlight, so hovering or selecting a
// profile lights the WHOLE area, not only its outline, matching Inventor's region highlight (#2165). A
// stale index or a region whose triangulation is empty adds nothing.
func appendProfileFill(items []renderer.DrawItem, ph app.ProfileHandle, color [4]float32) []renderer.DrawItem {
	if ph.ProfileIndex >= ph.Sketch.Profiles().Count() {
		return items
	}
	prof := ph.Sketch.Profiles().Item(ph.ProfileIndex)
	if !prof.IsClosed() {
		return items // an open profile bounds no area to fill
	}
	outer := prof.OuterLoop().Polygon()
	verts := append([]math.Point2(nil), outer...)
	holes := make([][]math.Point2, 0, len(prof.InnerLoops()))
	for _, h := range prof.InnerLoops() {
		holes = append(holes, h.Polygon())
		verts = append(verts, h.Polygon()...)
	}
	tris := ops.FillTriangles(outer, holes)
	if len(tris) == 0 {
		return items
	}
	return append(items, regionFillItem(ph.Sketch.Plane(), verts, tris, color))
}

// regionFillItem lifts a region's 2D triangulation onto its sketch plane as one translucent on-top
// Triangles item (the fill shares the profile-outline colour, at the face-fill opacity).
func regionFillItem(plane sketch.Plane, verts []math.Point2, tris [][3]int, color [4]float32) renderer.DrawItem {
	pos := make([]math.Point3, len(verts))
	normals := make([]math.Vector3, len(verts))
	n := plane.Normal().AsVector()
	for i, p := range verts {
		pos[i], normals[i] = plane.ToModel(p), n
	}
	idx := make([]int, 0, len(tris)*3)
	for _, t := range tris {
		idx = append(idx, t[0], t[1], t[2])
	}
	return renderer.DrawItem{
		Primitive: renderer.Triangles, Positions: pos, Normals: normals, Indices: idx,
		Color: color, Opacity: faceFillOpacity, OnTop: true,
	}
}

// faceFill returns a translucent, always-on-top tint of a face's tessellation — the visible part
// of a face highlight (the highlighted face is the front-most hit, so drawing it on top reads
// cleanly without z-fighting the model's own edges). Used for both preselect and selected faces.
func faceFill(fh app.FaceHandle, color [4]float32) renderer.DrawItem {
	mesh := tessellate.TessellateFace(fh.Face, ops.DefaultQuality())
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

// regionBadgeSession is drawRegionModifierBadge's slim view of the session (audit I5): just the
// add/remove decision for the region under the cursor. RegionModifierHint already returns show=false
// when no region tool is active, so the badge needs nothing else.
type regionBadgeSession interface {
	RegionModifierHint(x, y float64, mods app.Modifier) (show, add bool)
}

var _ regionBadgeSession = (*app.Session)(nil)

// drawRegionModifierBadge draws a +/− glyph beside the cursor when a modified click during area
// selection would ADD or REMOVE the region under the cursor — Inventor's add/remove cursor feedback
// (#2165). It renders in screen space in the viewport-overlay pass (not part of the 3D draw list), so
// it must be called while the viewport item is current. Nothing is drawn unless a region multi-select
// tool is active, a modifier is held, and a region sits under the cursor (RegionModifierHint).
func drawRegionModifierBadge(s regionBadgeSession) {
	if !native.IsItemHovered() {
		return
	}
	var mods app.Modifier
	if native.KeyCtrl() {
		mods |= app.CtrlMod
	}
	if native.KeyShift() {
		mods |= app.ShiftMod
	}
	cx, cy := viewportCursor()
	show, add := s.RegionModifierHint(cx, cy, mods)
	if !show {
		return
	}
	mx, my := native.MousePos()
	drawCursorPlusMinus(mx, my, add)
}

// drawCursorPlusMinus paints a small badge offset from the cursor: a green plus for ADD, a red minus
// for REMOVE — the horizontal bar is common to both, the vertical bar makes the plus.
func drawCursorPlusMinus(mx, my float32, add bool) {
	const off, half = 13.0, 5.0
	bx, by := mx+off, my+off
	native.DrawRectFilled(bx-half-2, by-half-2, bx+half+2, by+half+2, [4]float32{0.11, 0.11, 0.12, 0.88})
	fg := [4]float32{0.45, 0.95, 0.45, 1} // add → green
	if !add {
		fg = [4]float32{0.98, 0.5, 0.5, 1} // remove → red
	}
	native.DrawLine(bx-half, by, bx+half, by, fg, 2)
	if add {
		native.DrawLine(bx, by-half, bx, by+half, fg, 2)
	}
}

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
