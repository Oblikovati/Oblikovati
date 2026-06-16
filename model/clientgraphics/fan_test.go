// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestBuildTriangleFanExpandsIndices checks a triangle-fan primitive (M16-F05 #641) becomes a
// shaded triangle item with v0-anchored corner indices (a 4-vertex fan → 2 triangles).
func TestBuildTriangleFanExpandsIndices(t *testing.T) {
	args := wire.SetClientGraphicsArgs{ClientId: "fan", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:        string(types.GraphicsTriangleFan),
		Coordinates: []float64{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0},
		Color:       []float32{0, 1, 0, 1},
	}}}}}
	items, _ := buildOne(t, args)
	if len(items) != 1 {
		t.Fatalf("want one item, got %d", len(items))
	}
	want := []int{0, 1, 2, 0, 2, 3}
	if len(items[0].Indices) != len(want) {
		t.Fatalf("fan indices = %v, want %v", items[0].Indices, want)
	}
	for i := range want {
		if items[0].Indices[i] != want[i] {
			t.Errorf("fan indices = %v, want %v", items[0].Indices, want)
			break
		}
	}
}

// TestFanTriangleIndicesDegenerate checks fewer than 3 vertices yields no triangles.
func TestFanTriangleIndicesDegenerate(t *testing.T) {
	if fanTriangleIndices(2) != nil {
		t.Error("a 2-vertex fan should produce no triangles")
	}
}
