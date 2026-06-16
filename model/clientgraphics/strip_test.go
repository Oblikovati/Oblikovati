// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestBuildTriangleStripExpandsIndices checks a triangle-strip primitive becomes a shaded
// triangle item with alternating-winding corner indices (a 4-vertex strip → 2 triangles).
func TestBuildTriangleStripExpandsIndices(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "strip", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:        string(types.GraphicsTriangleStrip),
		Coordinates: []float64{0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0},
		Color:       []float32{0, 1, 0, 1},
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 || len(items[0].Indices) != 6 {
		t.Fatalf("triangle strip of 4 should give 2 triangles (6 indices), got %+v", items)
	}
}

// TestBuildPerVertexColorsFoldOpacity checks a primitive carrying per-vertex colors AND an
// opacity folds the opacity into each vertex alpha (the withOpacity path).
func TestBuildPerVertexColorsFoldOpacity(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "fea", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:         string(types.GraphicsTriangles),
		Coordinates:  []float64{0, 0, 0, 1, 0, 0, 0, 1, 0},
		Indices:      []int{0, 1, 2},
		Colors:       []float32{1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 1},
		ColorBinding: string(types.GraphicsColorPerVertex),
		Opacity:      0.5,
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 || len(items[0].Colors) != 3 {
		t.Fatalf("want one item with 3 per-vertex colors, got %+v", items)
	}
	for i, c := range items[0].Colors {
		if c[3] != 0.5 {
			t.Errorf("vertex %d alpha = %v, want 0.5 (opacity folded in)", i, c[3])
		}
	}
}
