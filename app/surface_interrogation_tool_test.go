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
	s.StartTool(NewSurfaceInterrogationTool())
	items := s.SurfaceInterrogationItems()
	if len(items) == 0 {
		t.Fatal("the active overlay should draw interrogation lines on a curved surface")
	}
	if items[0].Primitive != renderer.Lines || len(items[0].Positions) == 0 {
		t.Errorf("interrogation item should be a non-empty line batch, got %+v", items[0].Primitive)
	}
}

func TestSurfaceInterrogationModes(t *testing.T) {
	s, _ := partWithCurvedSurface(t)
	for _, mode := range []int{interrogIsophote, interrogReflection, interrogHighlight} {
		tool := NewSurfaceInterrogationTool()
		tool.mode = mode
		s.StartTool(tool)
		if len(s.SurfaceInterrogationItems()) == 0 {
			t.Errorf("mode %d should produce interrogation lines on a curved surface", mode)
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

func TestSurfaceAnalysisViaRibbonCommand(t *testing.T) {
	s, _ := partWithCurvedSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Inspect.SurfaceAnalysis"); err != nil {
		t.Fatalf("execute Inspect.SurfaceAnalysis: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Surface Analysis" {
		t.Errorf("Inspect.SurfaceAnalysis started tool %q, want Surface Analysis", got)
	}
}

func TestTrianglesFromIndices(t *testing.T) {
	got := trianglesFromIndices([]int{0, 1, 2, 3, 4, 5})
	if len(got) != 2 || got[0] != [3]int{0, 1, 2} || got[1] != [3]int{3, 4, 5} {
		t.Errorf("trianglesFromIndices = %v, want [[0 1 2] [3 4 5]]", got)
	}
}
