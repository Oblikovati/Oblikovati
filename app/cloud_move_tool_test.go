// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
)

// TestCloudMoveDragTranslatesAndDatumFollows: with the Move tool active, a left-drag translates the
// cloud and a datum anchored on it follows live (the drag recomputes the part) (#645).
func TestCloudMoveDragTranslatesAndDatumFollows(t *testing.T) {
	s, def := emptyPartSession(t) // camera looks down +Z at the XY plane, 200×200
	pc, wp := anchorWorkPointOnScan(t, s, def, math.P3(0, 0, 0))

	// Arm the Move tool on the selected cloud.
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if !canMoveSelectedCloud(s) {
		t.Fatal("Move should be enabled with a cloud selected")
	}
	if err := s.StartMoveSelectedCloud(); err != nil {
		t.Fatalf("StartMoveSelectedCloud: %v", err)
	}
	if !s.CloudMoveActive() {
		t.Fatal("the Move tool should be the active tool")
	}

	// Drag from the screen centre to the right: the cloud (and the work point on it) move in x.
	if !s.BeginCloudDrag(100, 100) {
		t.Fatal("BeginCloudDrag should start a drag")
	}
	s.UpdateCloudDrag(150, 100)
	moved := wp.Point()
	const eps = 1e-9
	if stdmath.Abs(float64(moved.X)) < eps {
		t.Errorf("dragged work point = %v, want a non-trivial x move (screen-parallel, top view)", moved)
	}
	if stdmath.Abs(float64(moved.Y)) > eps || stdmath.Abs(float64(moved.Z)) > eps {
		t.Errorf("dragged work point = %v, want movement in x only", moved)
	}
	s.CommitCloudDrag()
	if s.CloudDragActive() {
		t.Error("commit should end the drag")
	}
}

// TestCloudMoveGating: the drag only runs while the Move tool is active and a cloud is selected.
func TestCloudMoveGating(t *testing.T) {
	s, def := emptyPartSession(t)

	if canMoveSelectedCloud(s) {
		t.Error("Move should be disabled with nothing selected")
	}
	if err := s.StartMoveSelectedCloud(); err == nil {
		t.Error("StartMoveSelectedCloud should error with no cloud selected")
	}
	if s.BeginCloudDrag(100, 100) {
		t.Error("BeginCloudDrag should not start without the Move tool active")
	}

	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if err := s.StartMoveSelectedCloud(); err != nil {
		t.Fatalf("StartMoveSelectedCloud: %v", err)
	}
	// UpdateCloudDrag and CommitCloudDrag with no active drag are harmless no-ops.
	s.UpdateCloudDrag(120, 120)
	s.CommitCloudDrag()
}

// TestCloudMoveToolTrivialsAndNoCloudBranch covers the tool's no-op accessors and the Begin path
// when the tool is active but no cloud is selected (#645).
func TestCloudMoveToolTrivialsAndNoCloudBranch(t *testing.T) {
	s, def := emptyPartSession(t)
	tool := NewCloudMoveTool()
	if tool.Name() == "" {
		t.Error("tool should have a name")
	}
	tool.Start(s)
	tool.Pick(s, nil)
	if tool.CanCommit() {
		t.Error("CanCommit should be false (drag-driven)")
	}
	if err := tool.Commit(s); err != nil {
		t.Errorf("Commit: %v", err)
	}
	tool.Cancel(s)

	// Arm the tool with a cloud, then change the selection to a non-cloud: Begin must not start.
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if err := s.StartMoveSelectedCloud(); err != nil {
		t.Fatalf("StartMoveSelectedCloud: %v", err)
	}
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(0, 0, 0)}) // a scan point, not the cloud
	if s.BeginCloudDrag(100, 100) {
		t.Error("BeginCloudDrag should not start when no cloud is selected")
	}
}

// TestActiveToolConsumesClicks: the predicate that makes box-select stand down — true for a
// PlaneClickTool (the Crop Box tool), false otherwise (#645).
func TestActiveToolConsumesClicks(t *testing.T) {
	s, _ := emptyPartSession(t)
	if s.ActiveToolConsumesClicks() {
		t.Error("no active tool should not consume clicks")
	}
	s.StartTool(NewCropBoxTool(nil)) // a PlaneClickTool (ClickAt)
	if !s.ActiveToolConsumesClicks() {
		t.Error("the Crop Box tool should consume clicks")
	}
	s.StartTool(NewCloudMoveTool()) // drag-driven, not a PlaneClickTool
	if s.ActiveToolConsumesClicks() {
		t.Error("the Move tool is drag-driven and should not consume clicks (updateCloudDrag handles it)")
	}
}
