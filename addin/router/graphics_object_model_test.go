// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestGraphicsNodeMutations submits a group with a named node, then moves / hides / flags it
// through the targeted retained-mode methods (no geometry resubmit).
func TestGraphicsNodeMutations(t *testing.T) {
	r, s := seededSession(t)
	tri := wire.GraphicsPrimitive{Kind: "triangles", Coordinates: []float64{0, 0, 0, 1, 0, 0, 0, 1, 0}, Indices: []int{0, 1, 2}, Color: []float32{1, 0, 0, 1}}
	call(t, r, s, "clientGraphics.set", mustJSON(t, wire.SetClientGraphicsArgs{
		ClientId: "overlay", Nodes: []wire.GraphicsNode{{Id: "n1", Primitives: []wire.GraphicsPrimitive{tri}}},
	}), &wire.SetClientGraphicsResult{})

	identity := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	call(t, r, s, "graphicsNode.setTransform", mustJSON(t, wire.SetNodeTransformArgs{ClientId: "overlay", NodeId: "n1", Transform: identity}), &wire.OKResult{})
	call(t, r, s, "graphicsNode.setVisible", mustJSON(t, wire.SetNodeVisibleArgs{ClientId: "overlay", NodeId: "n1", Visible: false}), &wire.OKResult{})
	call(t, r, s, "graphicsNode.setSelectable", mustJSON(t, wire.SetNodeSelectableArgs{ClientId: "overlay", NodeId: "n1", Selectable: true}), &wire.OKResult{})

	if err := tryCall(t, r, s, "graphicsNode.setVisible", mustJSON(t, wire.SetNodeVisibleArgs{ClientId: "overlay", NodeId: "ghost", Visible: true})); err == nil {
		t.Error("mutating an unknown node should error")
	}
}

// TestColorMapperRegistryHandlers registers a named mapper and lists it back.
func TestColorMapperRegistryHandlers(t *testing.T) {
	r, s := seededSession(t)
	mapper := wire.GraphicsColorMapper{Values: []float64{0, 1}, Colors: []float32{0, 0, 1, 1, 1, 0, 0, 1}}
	call(t, r, s, "clientGraphics.registerColorMapper", mustJSON(t, wire.RegisterColorMapperArgs{Name: "stress", Mapper: mapper}), &wire.OKResult{})

	var list wire.ColorMappersResult
	call(t, r, s, "clientGraphics.listColorMappers", `{}`, &list)
	if len(list.Mappers) != 1 || list.Mappers[0].Name != "stress" || list.Mappers[0].StopCount != 2 {
		t.Errorf("mappers = %+v, want one stress/2-stop entry", list.Mappers)
	}

	if err := tryCall(t, r, s, "clientGraphics.registerColorMapper", mustJSON(t, wire.RegisterColorMapperArgs{Name: "bad", Mapper: wire.GraphicsColorMapper{Values: []float64{0}, Colors: []float32{1, 2}}})); err == nil {
		t.Error("a malformed mapper (colors != 4*values) should error")
	}
}
