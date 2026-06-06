// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/api/types"
	"oblikovati/api/wire"
	"oblikovati/model/clientgraphics"
)

// noopTool is a minimal Tool that commits on demand — enough to drive the tool-teardown
// path that clears interaction graphics.
type noopTool struct{ ready bool }

func (t *noopTool) Name() string              { return "noop" }
func (t *noopTool) Start(*Session)            {}
func (t *noopTool) Pick(*Session, Selectable) {}
func (t *noopTool) CanCommit() bool           { return t.ready }
func (t *noopTool) Commit(*Session) error     { return nil }
func (t *noopTool) Cancel(*Session)           {}

// previewLine seeds the preview interaction lane with one line group.
func previewLine(s *Session) {
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{
		ClientId: "x", Lane: string(types.GraphicsLanePreview),
		Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsLines), Coordinates: []float64{0, 0, 0, 1, 0, 0}, Indices: []int{0, 1},
		}}}},
	})
	if err != nil {
		panic(err)
	}
	s.Graphics().ReplaceLane(clientgraphics.LanePreview, []clientgraphics.Group{g})
}

func TestCancelToolClearsInteractionGraphics(t *testing.T) {
	s := NewSession()
	s.StartTool(&noopTool{})
	previewLine(s)
	s.CancelTool()
	if got := len(s.Graphics().Groups()); got != 0 {
		t.Errorf("cancel should clear interaction graphics, got %d groups", got)
	}
}

func TestCommitClearsInteractionGraphics(t *testing.T) {
	s := NewSession()
	s.StartTool(&noopTool{ready: true})
	previewLine(s)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := len(s.Graphics().Groups()); got != 0 {
		t.Errorf("commit should clear interaction graphics, got %d groups", got)
	}
}

func TestStartingNewToolClearsPriorInteractionGraphics(t *testing.T) {
	s := NewSession()
	s.StartTool(&noopTool{})
	previewLine(s)
	s.StartTool(&noopTool{}) // replacing a tool drops its preview
	if got := len(s.Graphics().Groups()); got != 0 {
		t.Errorf("starting a new tool should clear the prior preview, got %d groups", got)
	}
}

// TestPersistentGraphicsSurviveToolTeardown guards that only the interaction lanes are
// cleared — document-owned client graphics outlive commands.
func TestPersistentGraphicsSurviveToolTeardown(t *testing.T) {
	s := NewSession()
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{
		ClientId: "keep", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsPoints), Coordinates: []float64{0, 0, 0},
		}}}},
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	s.Graphics().Set(g)
	s.StartTool(&noopTool{})
	s.CancelTool()
	if got := len(s.Graphics().Groups()); got != 1 {
		t.Errorf("persistent graphics should survive, got %d groups", got)
	}
}
