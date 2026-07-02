// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/renderer"
)

// Edge selection has no per-body recolor (an edge is not a body), so a lone edge selected
// outside any tool is highlighted by drawing it as an overlay line in the selection colour.
// Edges gathered by an active tool (chamfer, fillet, …) are highlighted by the unified
// toolSelectedHighlight (toolPicks/drawSelectable), so they are not handled here.

// selectedEdgeOverlay draws a single edge selected outside a tool in the selection colour.
// Returns nil when there is none.
func selectedEdgeOverlay(s *app.Session) []renderer.DrawItem {
	edges := highlightedEdges(s)
	if len(edges) == 0 {
		return nil
	}
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
	return []renderer.DrawItem{lineItem(acc, chromeTheme.selectionHighlight)}
}

// highlightedEdges returns a lone selected edge (outside any tool) to highlight. An active
// tool's gathered edges flow through the unified toolSelectedHighlight, so they are skipped
// here to avoid drawing them twice.
func highlightedEdges(s *app.Session) []*topo.Edge {
	if s.ActiveTool() != nil {
		return nil
	}
	if h, ok := s.Selection().First().(app.EdgeHandle); ok {
		return []*topo.Edge{h.Edge}
	}
	return nil
}
