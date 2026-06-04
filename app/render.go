// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/clientgraphics"
	"github.com/Oblikovati/oblikovati/renderer"
)

// Client/preview graphics (M05-F05) + frame assembly. The session collects overlay
// primitives (persistent client graphics) and a tool may contribute a transient
// preview; RenderFrame assembles the active part's geometry plus these into one draw
// list and submits it to a backend. With the NullBackend this is fully headless, so
// a test asserts exactly what the viewport would show after a UI action.

// Previewable is an optional Tool capability: a tool that can show a live preview of
// its result before commit implements it (Inventor's in-canvas preview).
type Previewable interface {
	Preview(s *Session) []renderer.DrawItem
}

// AddOverlay adds a persistent client-graphics primitive (overlay line/points drawn
// over the model). ClearOverlays removes them; Overlays returns a snapshot.
func (s *Session) AddOverlay(item renderer.DrawItem) { s.overlays = append(s.overlays, item) }
func (s *Session) ClearOverlays()                    { s.overlays = nil }
func (s *Session) Overlays() []renderer.DrawItem {
	out := make([]renderer.DrawItem, len(s.overlays))
	copy(out, s.overlays)
	return out
}

// Graphics returns the add-in client/interaction graphics store (M05-F05), the seam the
// router mutates from clientGraphics.* / interactionGraphics.* calls.
func (s *Session) Graphics() *clientgraphics.Store { return s.graphics }

// GraphicsLabels returns the world-anchored text labels of the live client graphics, for
// the UI head to draw via its projected-ImGui label path (text is not draw-list geometry).
func (s *Session) GraphicsLabels() []clientgraphics.Label {
	_, labels := s.graphics.Build(s.camera)
	return labels
}

// RenderFrame builds the frame draw list — the active part's bodies, the persistent
// overlays, the active tool's transient preview, and the add-in client/interaction
// graphics — and submits it to the backend.
func (s *Session) RenderFrame(backend renderer.Backend) {
	list := renderer.BuildDrawListStyled(s.sceneBodies(), s.camera, ops.DefaultQuality(), s.SurfaceLookup(), s.visualStyle)
	list.Items = append(list.Items, s.overlays...)
	if s.tool != nil {
		if p, ok := s.tool.tool.(Previewable); ok {
			list.Items = append(list.Items, p.Preview(s)...)
		}
	}
	graphics, _ := s.graphics.Build(s.camera)
	list.Items = append(list.Items, graphics...)
	backend.Render(list)
}

// sceneBodies returns the bodies to draw — the active part's, if any.
func (s *Session) sceneBodies() []*topo.Body {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	return part.SurfaceBodies().All()
}
