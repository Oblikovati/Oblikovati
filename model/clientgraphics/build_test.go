// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati/api/types"
	"oblikovati/api/wire"

	"oblikovati/renderer"
	"oblikovati/scene"
)

func testCamera() scene.Camera { return scene.NewCamera(100, 100) }

func buildOne(t *testing.T, args wire.SetClientGraphicsArgs) ([]renderer.DrawItem, []Label) {
	t.Helper()
	s := NewStore()
	s.Set(mustDecode(t, args))
	return s.Build(testCamera())
}

// TestBuildHeatmapResolvesPerVertexColors is the headline case: scalars through a mapper
// become per-vertex DrawItem.Colors, so the renderer shades the FEA gradient.
func TestBuildHeatmapResolvesPerVertexColors(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "fea", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:         string(types.GraphicsTriangles),
		Coordinates:  []float64{0, 0, 0, 1, 0, 0, 0, 1, 0},
		Indices:      []int{0, 1, 2},
		Scalars:      []float64{0, 0.5, 1},
		ColorBinding: string(types.GraphicsColorPerVertex),
		ColorMapper:  &wire.GraphicsColorMapper{Values: []float64{0, 1}, Colors: []float32{0, 0, 1, 1, 1, 0, 0, 1}},
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 || len(items[0].Colors) != 3 {
		t.Fatalf("want one item with 3 per-vertex colors, got %d items", len(items))
	}
	// vertex 0 scalar 0 → blue, vertex 2 scalar 1 → red.
	if items[0].Colors[0] != ([4]float32{0, 0, 1, 1}) || items[0].Colors[2] != ([4]float32{1, 0, 0, 1}) {
		t.Errorf("endpoint colors = %v, want blue→red", items[0].Colors)
	}
}

func TestBuildOverallColorLeavesPerVertexNil(t *testing.T) {
	args := meshArgs("solid", LanePersistent)
	args.Nodes[0].Primitives[0].Color = []float32{0.2, 0.4, 0.6, 1}
	args.Nodes[0].Primitives[0].ColorBinding = string(types.GraphicsColorOverall)
	items, _ := buildOne(t, args)
	if len(items) != 1 || items[0].Colors != nil {
		t.Fatalf("overall color should leave Colors nil, got %+v", items)
	}
	if items[0].Color != ([4]float32{0.2, 0.4, 0.6, 1}) {
		t.Errorf("overall color = %v, want set", items[0].Color)
	}
}

func TestBuildLinesUsesIndices(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "ln", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:        string(types.GraphicsLines),
		Coordinates: []float64{0, 0, 0, 1, 1, 1},
		Indices:     []int{0, 1},
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 || items[0].Primitive != renderer.Lines || items[0].LineCount() != 1 {
		t.Fatalf("want one line item with 1 segment, got %+v", items)
	}
}

func TestBuildLineStripDerivesPairs(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "strip", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:        string(types.GraphicsLineStrip),
		Coordinates: []float64{0, 0, 0, 1, 0, 0, 2, 0, 0}, // 3 vertices → 2 segments
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 || items[0].LineCount() != 2 {
		t.Fatalf("line strip of 3 should give 2 segments, got %+v", items)
	}
}

func TestBuildPointsExpandToGlyphSegments(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "pts", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:        string(types.GraphicsPoints),
		Coordinates: []float64{0, 0, 0, 5, 5, 5}, // 2 points
		PointStyle:  string(types.GraphicsPointPlus),
	}}}}}
	items, _ := buildOne(t, args)
	// A "plus" is 2 segments per point → 4 segments for 2 points.
	if len(items) != 1 || items[0].LineCount() != 4 {
		t.Fatalf("two plus glyphs should give 4 segments, got %+v", items)
	}
}

func TestBuildTextBecomesLabelNotGeometry(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "lbl", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:   string(types.GraphicsText),
		Text:   "12.3 MPa",
		Anchor: []float64{1, 2, 3},
	}}}}}
	items, labels := buildOne(t, args)
	if len(items) != 0 {
		t.Errorf("text should produce no draw items, got %d", len(items))
	}
	if len(labels) != 1 || labels[0].Text != "12.3 MPa" {
		t.Fatalf("want one label '12.3 MPa', got %+v", labels)
	}
	if labels[0].Anchor.X != 1 || labels[0].Anchor.Y != 2 || labels[0].Anchor.Z != 3 {
		t.Errorf("label anchor = %v, want (1,2,3)", labels[0].Anchor)
	}
}

func TestBuildOverlayLaneMarksOnTop(t *testing.T) {
	items, _ := buildOne(t, meshArgs("o", LaneOverlay))
	if len(items) != 1 || !items[0].OnTop {
		t.Errorf("overlay-lane item should be OnTop, got %+v", items)
	}
}

func TestBuildInvisibleGroupProducesNothing(t *testing.T) {
	s := NewStore()
	g := mustDecode(t, meshArgs("h", LanePersistent))
	s.Set(g)
	if err := s.SetVisible("h", false); err != nil {
		t.Fatalf("SetVisible: %v", err)
	}
	items, _ := s.Build(testCamera())
	if len(items) != 0 {
		t.Errorf("hidden group should produce no items, got %d", len(items))
	}
}

func TestBuildAppliesNodeTransform(t *testing.T) {
	args := meshArgs("xf", LanePersistent)
	// Translate by (10,0,0) (row-major 4x4 with last column the translation).
	args.Nodes[0].Transform = []float64{
		1, 0, 0, 10,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	items, _ := buildOne(t, args)
	if len(items) != 1 {
		t.Fatalf("want one item, got %d", len(items))
	}
	// First coordinate (0,0,0) should now sit at x=10.
	if got := items[0].Positions[0].X; got != 10 {
		t.Errorf("transformed x = %v, want 10", got)
	}
}
