//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/renderer"
)

// Unified tool highlighting: every interactive tool gives the SAME feedback — the selectable
// under the cursor (that the tool's filter would pick) is outlined in the candidate colour, and
// what the tool has already picked is outlined in the selection colour. Both run off one per-kind
// dispatcher (drawSelectable) and the tool's existing pick accessors (toolPicks), so a new tool
// needs no head-side highlight code. Work planes and sketch curves keep their own overlays
// (planesOverlay/hoveredPlane and revolveCenterlineHighlight) since they are not B-rep selectables.

// drawSelectable returns the overlay wireframe for one selectable in colour — the kinds tools
// pick in the 3D view (a sketch profile region, a B-rep edge, a B-rep face). Other kinds (work
// planes, sketch entities) are drawn by their dedicated overlays and return nil here.
func drawSelectable(sel app.Selectable, color [4]float32) []renderer.DrawItem {
	switch h := sel.(type) {
	case app.ProfileHandle:
		return profileOutline(h, color)
	case app.EdgeHandle:
		return edgeWire([]*topo.Edge{h.Edge}, color)
	case app.FaceHandle:
		return edgeWire(h.Face.Edges(), color)
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

// toolHoverHighlight outlines the selectable under the cursor that the active tool would pick,
// in the candidate colour — the same hover feedback for every tool.
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
	return drawSelectable(sel, sketchCandidateColor)
}

// toolSelectedHighlight outlines everything the active tool has picked, in the selection colour —
// gathered from the tool's existing accessors (toolPicks), so all tools highlight their picks.
func toolSelectedHighlight(s *app.Session) []renderer.DrawItem {
	at := s.ActiveTool()
	if at == nil {
		return nil
	}
	var items []renderer.DrawItem
	for _, sel := range toolPicks(at.Tool()) {
		items = append(items, drawSelectable(sel, selectionHighlight)...)
	}
	return items
}

// toolPicks gathers the selectables a tool has picked from whichever accessor interfaces it
// already implements (Edges/Faces/PickedProfiles/PickedProfile/PickedFace) — no per-tool method.
func toolPicks(t app.Tool) []app.Selectable {
	var picks []app.Selectable
	if el, ok := t.(interface{ Edges() []app.EdgeHandle }); ok {
		for _, e := range el.Edges() {
			picks = append(picks, e)
		}
	}
	if fl, ok := t.(interface{ Faces() []app.FaceHandle }); ok {
		for _, f := range fl.Faces() {
			picks = append(picks, f)
		}
	}
	if pl, ok := t.(interface{ PickedProfiles() []app.ProfileHandle }); ok {
		for _, p := range pl.PickedProfiles() {
			picks = append(picks, p)
		}
	} else if pp, ok := t.(interface {
		PickedProfile() (app.ProfileHandle, bool)
	}); ok {
		if ph, has := pp.PickedProfile(); has {
			picks = append(picks, ph)
		}
	}
	if hf, ok := t.(interface{ PickedFace() (app.FaceHandle, bool) }); ok {
		if fh, has := hf.PickedFace(); has {
			picks = append(picks, fh)
		}
	}
	return picks
}
