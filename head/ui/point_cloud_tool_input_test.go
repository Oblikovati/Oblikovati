//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// armCloudSession attaches a small scan to a framed model session with box-select enabled (a region
// picker + an always-empty point picker), selects the cloud, and returns it.
func armCloudSession(t *testing.T) (*app.Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := framedSession()
	s.SetPicker(fakeEmptyPicker{})                          // every press is on empty space
	s.SetRegionPicker(fakeRegionHit{sel: app.BodyHandle{}}) // box-select is available (model env)
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte("0 0 0\n1 0 0\n0 1 0\n1 1 0\n"), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	pc, _, err := s.AttachPointCloud("Scan", path)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	s.Select(app.PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	return s, def
}

// TestInWindowCropToolOwnsClicks: while the Crop Box tool is active, a viewport press must feed the
// tool (a crop corner) and NOT start a box-select (which would swallow the click and clear the
// selection) (#645).
func TestInWindowCropToolOwnsClicks(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s, def := armCloudSession(t)
	if err := s.StartCropSelectedCloud(); err != nil {
		t.Fatalf("StartCropSelectedCloud: %v", err)
	}

	cx, cy := float32(inWinW/2), float32(inWinH/2)
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true) // first crop corner
	viewportFrame(win, s)
	if s.BoxSelectActive() {
		t.Fatal("box-select armed while the Crop tool is active — the tool's clicks are intercepted")
	}
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)

	// Second corner elsewhere → the crop commits.
	before := def.PointClouds().Item(0).Crops().Count()
	native.InjectMousePos(cx+60, cy+40)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
	if got := def.PointClouds().Item(0).Crops().Count(); got != before+1 {
		t.Errorf("two viewport clicks added %d crops, want 1 (the tool should own the clicks)", got-before)
	}
}

// TestInWindowMoveToolOwnsDrag: while the Move tool is active, a viewport press starts the cloud
// drag, not a box-select (#645).
func TestInWindowMoveToolOwnsDrag(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s, _ := armCloudSession(t)
	if err := s.StartMoveSelectedCloud(); err != nil {
		t.Fatalf("StartMoveSelectedCloud: %v", err)
	}
	cx, cy := float32(inWinW/2), float32(inWinH/2)
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	if s.BoxSelectActive() {
		t.Error("box-select armed while the Move tool is active")
	}
	if !s.CloudDragActive() {
		t.Error("the Move tool did not begin a cloud drag on press")
	}
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
}

// TestInWindowBoxSelectRestoredAfterTool: once a click tool deactivates, an empty-space press
// begins box-select again (the guard only stands box-select down while a click tool is active).
func TestInWindowBoxSelectRestoredAfterTool(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s, _ := armCloudSession(t)
	if err := s.StartCropSelectedCloud(); err != nil {
		t.Fatalf("StartCropSelectedCloud: %v", err)
	}
	s.CancelTool() // leave the Crop tool

	cx, cy := float32(inWinW/2), float32(inWinH/2)
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	if !s.BoxSelectActive() {
		t.Error("box-select should arm again once the click tool is gone")
	}
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
}
