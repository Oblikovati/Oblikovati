// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/renderer"
)

// callGraphics dispatches a graphics method through the router and fails on error.
func callGraphics(t *testing.T, r *Router, s *app.Session, method string, args any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal %s args: %v", method, err)
	}
	out, err := r.Handle(s, method, raw)
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	return out
}

// heatmapArgs is a one-triangle group colored per vertex through a blue→red mapper — a
// minimal simulation-result overlay.
func heatmapArgs() wire.SetClientGraphicsArgs {
	return wire.SetClientGraphicsArgs{ClientId: "fea", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind:         string(types.GraphicsTriangles),
		Coordinates:  []float64{0, 0, 0, 1, 0, 0, 0, 1, 0},
		Indices:      []int{0, 1, 2},
		Scalars:      []float64{0, 0.5, 1},
		ColorBinding: string(types.GraphicsColorPerVertex),
		ColorMapper:  &wire.GraphicsColorMapper{Values: []float64{0, 1}, Colors: []float32{0, 0, 1, 1, 1, 0, 0, 1}},
	}}}}}
}

// TestGraphicsEndToEndAppearsInFrame is the e2e: an add-in pushes a heatmap mesh, lines and
// a label through the router, and they show up in the rendered frame (heatmap with
// per-vertex colors) and in the label channel.
func TestGraphicsEndToEndAppearsInFrame(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()

	var set wire.SetClientGraphicsResult
	if err := json.Unmarshal(callGraphics(t, r, s, wire.MethodClientGraphicsSet, heatmapArgs()), &set); err != nil {
		t.Fatalf("decode set result: %v", err)
	}
	if set.PrimitiveCount != 1 {
		t.Errorf("set primitiveCount = %d, want 1", set.PrimitiveCount)
	}
	callGraphics(t, r, s, wire.MethodClientGraphicsSet, wire.SetClientGraphicsArgs{
		ClientId: "vectors", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsLines), Coordinates: []float64{0, 0, 0, 1, 1, 1}, Indices: []int{0, 1},
		}}}},
	})
	callGraphics(t, r, s, wire.MethodClientGraphicsSet, wire.SetClientGraphicsArgs{
		ClientId: "label", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsText), Text: "12 MPa", Anchor: []float64{0, 0, 0},
		}}}},
	})

	be := &renderer.NullBackend{}
	s.RenderFrame(be)
	frame := be.LastFrame()

	heatmap := findHeatmap(frame)
	if heatmap == nil {
		t.Fatal("heatmap triangle item missing from frame")
	}
	if len(heatmap.Colors) != 3 || heatmap.Colors[0] != ([4]float32{0, 0, 1, 1}) {
		t.Errorf("heatmap per-vertex colors = %v, want blue→red", heatmap.Colors)
	}
	if frame.Lines() < 1 {
		t.Error("vector line item missing from frame")
	}
	if labels := s.GraphicsLabels(); len(labels) != 1 || labels[0].Text != "12 MPa" {
		t.Errorf("labels = %+v, want one '12 MPa'", labels)
	}
}

// findHeatmap returns the per-vertex-colored triangle item in the frame, or nil.
func findHeatmap(frame renderer.DrawList) *renderer.DrawItem {
	for i := range frame.Items {
		if frame.Items[i].Primitive == renderer.Triangles && len(frame.Items[i].Colors) > 0 {
			return &frame.Items[i]
		}
	}
	return nil
}

func TestGraphicsListAndDelete(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	callGraphics(t, r, s, wire.MethodClientGraphicsSet, heatmapArgs())

	var list wire.ListClientGraphicsResult
	if err := json.Unmarshal(callGraphics(t, r, s, wire.MethodClientGraphicsList, struct{}{}), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Groups) != 1 || list.Groups[0].ClientId != "fea" {
		t.Fatalf("list = %+v, want one group 'fea'", list.Groups)
	}

	callGraphics(t, r, s, wire.MethodClientGraphicsDelete, wire.DeleteClientGraphicsArgs{ClientId: "fea"})
	if got := len(s.Graphics().Groups()); got != 0 {
		t.Errorf("group count after delete = %d, want 0", got)
	}
}

// TestInteractionGraphicsUpdateThenClear checks a transient preview shows in the frame and
// the clear method drops it (the command-end path).
func TestInteractionGraphicsUpdateThenClear(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	callGraphics(t, r, s, wire.MethodInteractionGraphicsUpdate, wire.UpdateInteractionGraphicsArgs{
		Lane: string(types.GraphicsLanePreview),
		Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsLines), Coordinates: []float64{0, 0, 0, 2, 0, 0}, Indices: []int{0, 1},
		}}}},
	})
	be := &renderer.NullBackend{}
	s.RenderFrame(be)
	if be.LastFrame().Lines() < 1 {
		t.Fatal("interaction preview line missing from frame")
	}

	callGraphics(t, r, s, wire.MethodInteractionGraphicsClear, struct{}{})
	if got := len(s.Graphics().Groups()); got != 0 {
		t.Errorf("groups after clear = %d, want 0", got)
	}
}
