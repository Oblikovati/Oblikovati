// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/renderer"
)

func TestRenderFrameDrawsTheModeledSolid(t *testing.T) {
	// Build a box through the UI, then render — the null backend records the frame.
	s := extrudedBox(t, 2, 4)
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if null.FrameCount() != 1 {
		t.Fatalf("frames = %d, want 1", null.FrameCount())
	}
	if tris := null.LastFrame().Triangles(); tris != 12 {
		t.Errorf("rendered triangles = %d, want 12 (the box)", tris)
	}
	if null.LastFrame().Lines() == 0 {
		t.Error("rendered frame has no wireframe lines")
	}
}

func TestPersistentOverlayInFrame(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	before := frameItems(s)
	s.AddOverlay(renderer.DrawItem{Primitive: renderer.Lines, Positions: nil, Indices: []int{0, 1}})
	if len(s.Overlays()) != 1 {
		t.Fatal("overlay not stored")
	}
	if frameItems(s) != before+1 {
		t.Error("overlay not included in the frame")
	}
	s.ClearOverlays()
	if frameItems(s) != before {
		t.Error("ClearOverlays did not remove the overlay")
	}
}

func TestExtrudeToolLivePreview(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)

	// No preview before a profile + distance are gathered.
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	preCount := len(null.LastFrame().Items)

	s.Click(0, 0)
	ext.SetDistance(5)
	// Now the tool contributes a transient wireframe preview to the frame.
	null.Reset()
	s.RenderFrame(null)
	if len(null.LastFrame().Items) <= preCount {
		t.Fatal("no preview item added once the extrude is ready")
	}
	// The preview is a line item with the expected vertex count (2n ring points).
	var preview *renderer.DrawItem
	for i := range null.LastFrame().Items {
		if null.LastFrame().Items[i].Primitive == renderer.Lines {
			preview = &null.LastFrame().Items[i]
		}
	}
	if preview == nil || len(preview.Positions) != 8 { // square → 4 bottom + 4 top
		t.Errorf("extrude preview wireframe wrong: %+v", preview)
	}

	// After OK the preview is gone and the real solid is rendered.
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	null.Reset()
	s.RenderFrame(null)
	if null.LastFrame().Triangles() != 12 {
		t.Errorf("post-commit frame triangles = %d, want 12", null.LastFrame().Triangles())
	}
}

func TestRenderFrameNoActivePart(t *testing.T) {
	s := NewSession() // no document
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if null.FrameCount() != 1 || len(null.LastFrame().Items) != 0 {
		t.Error("empty session should render an empty frame")
	}
}

func TestRenderFrameSkipsHiddenBody(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	body := s.VisibleBodies()[0]
	s.SetBodyVisible(body, false)
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if tris := null.LastFrame().Triangles(); tris != 0 {
		t.Fatalf("hidden body rendered %d triangles, want 0", tris)
	}
}

func frameItems(s *Session) int {
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	return len(null.LastFrame().Items)
}
