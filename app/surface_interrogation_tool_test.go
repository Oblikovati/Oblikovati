// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// partWithCurvedSurface creates a part holding a NURBS plane curved by a control-point edit, so its
// normals vary and interrogation lines are non-empty.
func partWithCurvedSurface(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, def := emptyPartSession(t)
	feature.NewNurbsPlaneFeatures(def.Features()).Add(10, 10, 6, 6)
	feature.NewControlPointEditFeatures(def.Features()).Add([]geom.ControlPointDelta{
		{U: 2, V: 2, Delta: math.V3(0, 0, 3)},
		{U: 3, V: 3, Delta: math.V3(0, 0, -2)},
	})
	def.Recompute()
	return s, def
}

func TestSurfaceInterrogationOverlayActiveOnlyWithTool(t *testing.T) {
	s, _ := partWithCurvedSurface(t)
	if items := s.SurfaceInterrogationItems(); items != nil {
		t.Error("no overlay should draw when the Surface Analysis tool is inactive")
	}
	s.StartTool(NewSurfaceInterrogationToolMode(interrogReflection))
	items := s.SurfaceInterrogationItems()
	if len(items) == 0 {
		t.Fatal("the active overlay should draw interrogation lines on a curved surface")
	}
	if items[0].Primitive != renderer.Lines || len(items[0].Positions) == 0 {
		t.Errorf("interrogation item should be a non-empty line batch, got %+v", items[0].Primitive)
	}
}

// TestSurfaceInterrogationZebraFillsTriangles: the zebra map is a filled (Triangles) overlay with
// per-vertex black/white band colours, not contour lines.
func TestSurfaceInterrogationZebraFillsTriangles(t *testing.T) {
	s, _ := partWithCurvedSurface(t)
	s.StartTool(NewSurfaceInterrogationToolMode(interrogZebra))
	items := s.SurfaceInterrogationItems()
	if len(items) == 0 {
		t.Fatal("zebra should draw a filled overlay on a curved surface")
	}
	if items[0].Primitive != renderer.Triangles || len(items[0].Colors) != len(items[0].Positions) {
		t.Errorf("zebra item should be per-vertex-coloured triangles, got %+v with %d colors / %d positions", items[0].Primitive, len(items[0].Colors), len(items[0].Positions))
	}
}

func TestSurfaceInterrogationModes(t *testing.T) {
	s, _ := partWithCurvedSurface(t)
	for _, mode := range []int{interrogZebra, interrogIsophote, interrogReflection, interrogHighlight} {
		s.StartTool(NewSurfaceInterrogationToolMode(mode))
		if len(s.SurfaceInterrogationItems()) == 0 {
			t.Errorf("mode %d should produce an interrogation overlay on a curved surface", mode)
		}
		s.CancelTool()
	}
}

func TestSurfaceInterrogationToolParams(t *testing.T) {
	tool := NewSurfaceInterrogationTool()
	if tool.Prompt(nil) == "" {
		t.Error("prompt should be non-empty")
	}
	if tool.CanCommit() {
		t.Error("CanCommit should be false (display overlay)")
	}
	if err := tool.Commit(nil); err != nil {
		t.Errorf("Commit is a no-op, got %v", err)
	}
	p := tool.Params()
	p.Ints[0].Set(20)
	p.Choices[0].Set(interrogReflection)
	for i, v := range []float64{0.5, -0.3, 0.8} { // exercise all three direction closures
		p.Floats[i].Set(v)
		if p.Floats[i].Get() != v {
			t.Errorf("float param %d get/set mismatch", i)
		}
	}
	if p.Ints[0].Get() != 20 || p.Choices[0].Get() != interrogReflection {
		t.Error("param get/set round-trip mismatch")
	}
}

func TestSurfaceAnalysisRibbonCommands(t *testing.T) {
	cases := map[string]int{
		"Inspect.Zebra":      interrogZebra,
		"Inspect.Isophotes":  interrogIsophote,
		"Inspect.Reflection": interrogReflection,
		"Inspect.Highlight":  interrogHighlight,
	}
	for id, wantMode := range cases {
		s, _ := partWithCurvedSurface(t)
		if err := RegisterStandardCommands(s); err != nil {
			t.Fatalf("register commands: %v", err)
		}
		if err := s.Execute(id); err != nil {
			t.Fatalf("execute %s: %v", id, err)
		}
		tool, ok := s.ActiveTool().Tool().(*SurfaceInterrogationTool)
		if !ok {
			t.Fatalf("%s did not start the Surface Analysis tool", id)
		}
		if tool.mode != wantMode {
			t.Errorf("%s started mode %d, want %d", id, tool.mode, wantMode)
		}
	}
}

func TestTrianglesFromIndices(t *testing.T) {
	got := trianglesFromIndices([]int{0, 1, 2, 3, 4, 5})
	if len(got) != 2 || got[0] != [3]int{0, 1, 2} || got[1] != [3]int{3, 4, 5} {
		t.Errorf("trianglesFromIndices = %v, want [[0 1 2] [3 4 5]]", got)
	}
}
