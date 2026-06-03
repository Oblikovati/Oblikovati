// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/renderer"
)

// Edge selection has no per-body recolor (an edge is not a body), so picked edges are
// highlighted by drawing them as overlay lines in the selection colour. This covers the
// edges an active edge tool (chamfer, fillet, …) has gathered and a lone selected edge, so
// the user sees their pick land.

// edgeLister is any tool that exposes the edges it has picked (e.g. the Chamfer tool).
type edgeLister interface {
	Edges() []app.EdgeHandle
}

// selectedEdgeOverlay draws the currently picked edges in the selection colour: the active
// edge tool's gathered edges, or a single edge selected outside a tool. Returns nil when
// there are none.
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
	return []renderer.DrawItem{lineItem(acc, selectionHighlight)}
}

// highlightedEdges returns the edges to highlight: the active edge tool's picks, else a lone
// selected edge.
func highlightedEdges(s *app.Session) []*topo.Edge {
	if at := s.ActiveTool(); at != nil {
		if el, ok := at.Tool().(edgeLister); ok {
			return edgesOf(el.Edges())
		}
	}
	if h, ok := s.Selection().First().(app.EdgeHandle); ok {
		return []*topo.Edge{h.Edge}
	}
	return nil
}

// edgesOf unwraps edge handles to their topo edges.
func edgesOf(handles []app.EdgeHandle) []*topo.Edge {
	out := make([]*topo.Edge, 0, len(handles))
	for _, h := range handles {
		out = append(out, h.Edge)
	}
	return out
}
