// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/math"
)

// fakeBodyResolver returns one unit triangle for the key "box", else not-found — a named fake
// standing in for the host's body tessellation (CLAUDE.md: named fakes, not inline stubs).
func fakeBodyResolver(key string, _ uint64) ([]math.Point3, []math.Vector3, []int, bool) {
	if key != "box" {
		return nil, nil, nil, false
	}
	return []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2}, true
}

// TestSurfaceOverlayResolvesBodyMesh checks a GraphicsSurface primitive renders the resolved
// body mesh in the override color (M16-F05 #641).
func TestSurfaceOverlayResolvesBodyMesh(t *testing.T) {
	s := NewStore()
	s.SetBodyResolver(fakeBodyResolver)
	s.Set(mustDecode(t, wire.SetClientGraphicsArgs{ClientId: "hl", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind: string(types.GraphicsSurface), BodyKey: "box", Color: []float32{1, 0, 0, 1},
	}}}}}))
	items, _ := s.Build(testCamera())
	if len(items) != 1 || len(items[0].Indices) != 3 {
		t.Fatalf("want one 3-index triangle item, got %+v", items)
	}
	if items[0].Color != ([4]float32{1, 0, 0, 1}) {
		t.Errorf("overlay color = %v, want red", items[0].Color)
	}
}

// TestSurfaceOverlayWithoutResolverDrawsNothing checks a surface primitive is skipped when no
// resolver is injected or the body is unknown.
func TestSurfaceOverlayWithoutResolverDrawsNothing(t *testing.T) {
	s := NewStore() // no resolver
	s.Set(mustDecode(t, wire.SetClientGraphicsArgs{ClientId: "hl", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind: string(types.GraphicsSurface), BodyKey: "box", Color: []float32{1, 0, 0, 1},
	}}}}}))
	if items, _ := s.Build(testCamera()); len(items) != 0 {
		t.Errorf("want nothing without a resolver, got %+v", items)
	}
}
