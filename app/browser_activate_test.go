// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/drawing"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The NodeActivatable capability performs a browser node's double-click action (#1630). These
// tests drive Activate on each handle kind and assert its real side effect, so head/ui's
// switch-free openEditOnDoubleClick has parity with the old per-type dispatch.

// TestSketchHandleActivate: double-clicking a 2D sketch re-enters the sketch environment.
func TestSketchHandleActivate(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	SketchHandle{Sketch: sk}.Activate(s)
	if !s.InSketch() {
		t.Error("activating a sketch node should enter the sketch environment")
	}
}

// TestWorkPlaneHandleActivate: double-clicking a user offset plane opens its redefine tool.
func TestWorkPlaneHandleActivate(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()
	WorkPlaneHandle{Plane: wp}.Activate(s)
	if s.ActiveWorkPlaneEdit() == nil {
		t.Error("activating a user work plane should open its edit tool")
	}
}

// TestAssemblyFeatureHandleActivate: double-clicking a committed machining feature opens its
// parameter-edit tool (#766).
func TestAssemblyFeatureHandleActivate(t *testing.T) {
	t.Parallel()
	s, asm, _ := chamferedAssembly(t)
	AssemblyFeatureHandle{Feature: asm.Features().Item(0)}.Activate(s)
	if s.ActiveTool() == nil {
		t.Error("activating an assembly feature should open its edit tool")
	}
}

// TestOccurrenceHandleActivate: double-clicking a placed occurrence opens the placed component's
// document as the active tab (#764).
func TestOccurrenceHandleActivate(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)}.Activate(s)
	if _, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition); !ok {
		t.Errorf("activating an occurrence should open its part document, active is %T",
			s.ActiveDocument().Content())
	}
}

// capturedLODAssembly returns an assembly with one occurrence suppressed and captured as a LOD
// representation, then un-suppressed — the fixture the representation/model-state activate tests
// use to prove activation re-applies the captured suppression (M12-F04).
func capturedLODAssembly(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition) {
	t.Helper()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	b := asm.Occurrences().Item(1)
	b.SetSuppressed(true)
	if err := s.CaptureLOD(); err != nil {
		t.Fatalf("CaptureLOD: %v", err)
	}
	b.SetSuppressed(false)
	return s, asm
}

// TestRepresentationHandleActivate: activating a captured LOD representation re-applies its
// suppression (M12-F04).
func TestRepresentationHandleActivate(t *testing.T) {
	t.Parallel()
	s, asm := capturedLODAssembly(t)
	folder := findBrowserNode(BuildBrowser(s), "representations", "Representations")
	if folder == nil || len(folder.Children) != 1 {
		t.Fatalf("Representations folder = %v, want one LOD row", folder)
	}
	h := folder.Children[0].Select.(RepresentationHandle)
	h.Activate(s)
	if !asm.Occurrences().Item(1).Suppressed() {
		t.Error("activating the LOD representation did not re-suppress the occurrence")
	}
}

// TestModelStateHandleActivate: activating a captured model state re-applies its representation
// families (M12-F04).
func TestModelStateHandleActivate(t *testing.T) {
	t.Parallel()
	s, asm := capturedLODAssembly(t)
	if _, err := asm.Representations().ActivateLOD(asm.Representations().AllLODs()[0].ID()); err != nil {
		t.Fatalf("ActivateLOD: %v", err)
	}
	if err := s.NewModelState(); err != nil {
		t.Fatalf("NewModelState: %v", err)
	}
	asm.Occurrences().Item(1).SetSuppressed(false)
	activateFirstModelState(t, s)
	if !asm.Occurrences().Item(1).Suppressed() {
		t.Error("activating the model state did not re-apply the LOD representation")
	}
}

// activateFirstModelState finds the single Model States browser row and activates it through the
// NodeActivatable capability.
func activateFirstModelState(t *testing.T, s *Session) {
	t.Helper()
	folder := findBrowserNode(BuildBrowser(s), "modelStates", "Model States")
	if folder == nil || len(folder.Children) != 1 {
		t.Fatalf("Model States folder = %v, want one row", folder)
	}
	folder.Children[0].Select.(ModelStateHandle).Activate(s)
}

// TestDrawingViewHandleActivate: double-clicking a drawing view opens its settings-edit tool
// (M14-F02).
func TestDrawingViewHandleActivate(t *testing.T) {
	t.Parallel()
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	spec := drawing.BaseViewSpec{Orientation: types.BaseViewIso, Scale: 1, CenterX: 150, CenterY: 100}
	if _, err := c.Sheets().Active().Views().AddBase(spec); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	h, ok := s.PickDrawingViewAt(150, 100)
	if !ok {
		t.Fatal("PickDrawingViewAt(150,100) found no view")
	}
	h.Activate(s)
	if s.ActiveTool() == nil {
		t.Error("activating a drawing view should open its edit tool")
	}
}
