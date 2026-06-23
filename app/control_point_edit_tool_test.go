// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// partWithNurbsGrid creates a part holding a flat 5×5 NURBS plane patch (2×2 in size), built
// through the real (serializable) NurbsPlane tool so the recipe-stream undo can snapshot it — the
// editable base surface for the CV-edit tests.
func partWithNurbsGrid(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, def := emptyPartSession(t)
	tool := NewNurbsPlaneTool()
	tool.width, tool.height, tool.uCount, tool.vCount = 2, 2, 5, 5
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("create NURBS plane: %v", err)
	}
	return s, def
}

// armCVDrag puts the tool into an active drag on control index (i,j) without needing pixel math.
func armCVDrag(s *Session, surf geom.BSplineSurface, i, j int) {
	s.cvEdit = cvEditDrag{cvU: i, cvV: j, origin: surf.Ctrl[i][j], normal: math.V3(0, 0, 1), from: surf.Ctrl[i][j], surf: surf, active: true}
}

func TestControlPointDragLiftsSurfaceAndIsUndoable(t *testing.T) {
	s, def := partWithNurbsGrid(t)
	s.StartTool(NewControlPointEditTool())
	surf, _ := activeEditableSurface(s)

	armCVDrag(s, surf, 2, 2)
	s.applyCVMove(math.V3(0, 0, 1)) // lift the centre CV by 1
	s.CommitCVDrag()

	edited, _ := activeEditableSurface(s)
	if edited.UDegree != 3 || len(edited.Ctrl) != 5 || len(edited.Ctrl[0]) != 5 {
		t.Errorf("edit changed structure: degree %d net %dx%d", edited.UDegree, len(edited.Ctrl), len(edited.Ctrl[0]))
	}
	if z := edited.PointAt(0.5, 0.5).Z; z <= 0.1 {
		t.Errorf("limit surface did not rise after lifting the centre CV: z=%g", z)
	}
	if !s.CanUndo() {
		t.Fatal("a control-point edit should be undoable")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if z := mustEditableSurface(t, s).PointAt(0.5, 0.5).Z; z > 1e-9 {
		t.Errorf("undo should flatten the surface again, z=%g", z)
	}
	_ = def
}

func TestControlPointRowDragWithFalloff(t *testing.T) {
	s, _ := partWithNurbsGrid(t)
	tool := NewControlPointEditTool()
	tool.mode = cvModeRow
	tool.radius = 0.6 // reaches the neighbouring rows
	s.StartTool(tool)
	surf, _ := activeEditableSurface(s)

	armCVDrag(s, surf, 2, 2)
	s.applyCVMove(math.V3(0, 0, 1))
	s.CommitCVDrag()

	edited, _ := activeEditableSurface(s)
	// The driven row (u=2) lifted fully; an adjacent row (u=1) lifted partially; the far edge (u=0)
	// barely moved — the falloff signature.
	mid := edited.Ctrl[2][2].Z
	near := edited.Ctrl[1][2].Z
	far := edited.Ctrl[0][2].Z
	if !(mid > near && near > far) {
		t.Errorf("row falloff not monotonic: mid=%g near=%g far=%g", mid, near, far)
	}
	if mid <= 0.9 {
		t.Errorf("driven row should lift by ~1, got %g", mid)
	}
}

func TestControlPointSymmetryMirrorsEdit(t *testing.T) {
	s, _ := partWithNurbsGrid(t)
	tool := NewControlPointEditTool()
	tool.symmetry = true
	s.StartTool(tool)
	surf, _ := activeEditableSurface(s)

	armCVDrag(s, surf, 1, 2) // off-centre, so the mirror (u=3) is a different CV
	s.applyCVMove(math.V3(0, 0, 1))
	s.CommitCVDrag()

	edited, _ := activeEditableSurface(s)
	if z := edited.Ctrl[1][2].Z; z <= 0.9 {
		t.Errorf("driven CV should lift, z=%g", z)
	}
	if z := edited.Ctrl[3][2].Z; z <= 0.9 {
		t.Errorf("symmetry should lift the mirrored CV (u=3) too, z=%g", z)
	}
}

func TestControlPointEditViaRibbonCommand(t *testing.T) {
	s, _ := partWithNurbsGrid(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.EditCV"); err != nil {
		t.Fatalf("execute Surface.EditCV: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Edit Control Points" {
		t.Errorf("Surface.EditCV started tool %q, want Edit Control Points", got)
	}
}

func TestNearestControlPointPicksClosest(t *testing.T) {
	s, _ := partWithNurbsGrid(t)
	surf, _ := activeEditableSurface(s)
	// A ray straight down −Z through CV (2,2) at the patch centre (1,1,0).
	o := math.P3(1, 1, 5)
	d := math.V3(0, 0, -1)
	i, j, ok := nearestControlPoint(surf, o, d, 0.1)
	if !ok || i != 2 || j != 2 {
		t.Errorf("nearestControlPoint = (%d,%d,%v), want (2,2,true)", i, j, ok)
	}
	// A ray far from any CV misses.
	if _, _, ok := nearestControlPoint(surf, math.P3(99, 99, 5), d, 0.1); ok {
		t.Error("a ray far from the net should hit no control point")
	}
}

func TestControlPointPreviewDrawsNetHandles(t *testing.T) {
	s, _ := partWithNurbsGrid(t)
	tool := NewControlPointEditTool()
	s.StartTool(tool)
	items := tool.Preview(s)
	if len(items) < 2 {
		t.Fatalf("preview should draw net lines + handle markers, got %d items", len(items))
	}
}

// mustEditableSurface fetches the active editable surface or fails.
func mustEditableSurface(t *testing.T, s *Session) geom.BSplineSurface {
	t.Helper()
	surf, ok := activeEditableSurface(s)
	if !ok {
		t.Fatal("no editable surface")
	}
	return surf
}
