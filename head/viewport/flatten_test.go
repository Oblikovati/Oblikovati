// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/renderer"
)

func triItem(z float64) renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: []math.Point3{math.P3(0, 0, z), math.P3(1, 0, z), math.P3(0, 1, z)},
		Normals:   []math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		Indices:   []int{0, 1, 2},
		Color:     [4]float32{1, 0, 0, 1},
	}
}

func lineItem() renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1)},
		Indices:   []int{0, 1},
		Color:     [4]float32{0, 1, 0, 1},
	}
}

func TestFlattenSplitsByPrimitive(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(0), lineItem()}})
	if m.TriVCount != 3 || len(m.TriIndices) != 3 {
		t.Errorf("triangles: vcount=%d idx=%d, want 3/3", m.TriVCount, len(m.TriIndices))
	}
	if m.LineVCount != 2 || len(m.LineIndices) != 2 {
		t.Errorf("lines: vcount=%d idx=%d, want 2/2", m.LineVCount, len(m.LineIndices))
	}
	if len(m.TriVerts) != 3*VertexFloats || len(m.LineVerts) != 2*VertexFloats {
		t.Errorf("vertex float counts wrong: tri=%d line=%d", len(m.TriVerts), len(m.LineVerts))
	}
}

// TestFlattenRebasesIndicesPerItem verifies a second triangle item's indices are
// offset by the running vertex count (so two items share one buffer correctly).
func TestFlattenRebasesIndicesPerItem(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(0), triItem(5)}})
	if m.TriVCount != 6 {
		t.Fatalf("vcount = %d, want 6", m.TriVCount)
	}
	// The second item's indices {0,1,2} must have been rebased to {3,4,5}.
	want := []uint32{0, 1, 2, 3, 4, 5}
	for i, v := range want {
		if m.TriIndices[i] != v {
			t.Errorf("TriIndices = %v, want %v", m.TriIndices, want)
			break
		}
	}
}

func TestFlattenInterleavesPositionNormalColor(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(2)}})
	// First vertex: pos (0,0,2), normal (0,0,1), color (1,0,0,1).
	got := m.TriVerts[:VertexFloats]
	want := []float32{0, 0, 2, 0, 0, 1, 1, 0, 0, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("interleaved vertex = %v, want %v", got, want)
		}
	}
}

func TestFlattenLinesTolerateMissingNormals(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{lineItem()}})
	// Normal slot (floats 3..6) should be zero, not a panic / garbage.
	if m.LineVerts[3] != 0 || m.LineVerts[4] != 0 || m.LineVerts[5] != 0 {
		t.Errorf("missing normals should default to zero, got %v", m.LineVerts[3:6])
	}
}
