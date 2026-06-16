// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestNamedViewCaptureRestore captures the active camera, moves it, then restores the named
// view and checks the camera returns to the captured frame exactly.
func TestNamedViewCaptureRestore(t *testing.T) {
	r, s := seededSession(t)

	var captured wire.NamedViewInfo
	call(t, r, s, "views.captureNamed", mustJSON(t, wire.CaptureNamedViewArgs{Name: "iso"}), &captured)

	// Move the camera somewhere else.
	call(t, r, s, "view.setCamera", mustJSON(t, wire.SetCameraArgs{
		Eye: types.NewPoint(99, 99, 99), Target: types.NewPoint(0, 0, 0), Up: types.NewVector(0, 1, 0), FOV: 0.8,
	}), &wire.CameraView{})

	var list wire.NamedViewsResult
	call(t, r, s, "views.listNamed", `{}`, &list)
	if len(list.Views) != 1 || list.Views[0].Name != "iso" {
		t.Fatalf("listNamed = %+v, want one view named iso", list.Views)
	}

	var restored wire.CameraView
	call(t, r, s, "views.restoreNamed", mustJSON(t, wire.NamedViewRefArgs{Name: "iso"}), &restored)
	if restored.Eye != captured.Camera.Eye || restored.FOV != captured.Camera.FOV {
		t.Errorf("restored camera = %+v, want captured %+v", restored, captured.Camera)
	}

	// Deleting then restoring should error.
	call(t, r, s, "views.deleteNamed", mustJSON(t, wire.NamedViewRefArgs{Name: "iso"}), &wire.OKResult{})
	if err := tryCall(t, r, s, "views.restoreNamed", mustJSON(t, wire.NamedViewRefArgs{Name: "iso"})); err == nil {
		t.Error("restoring a deleted named view should error")
	}
}

// TestSetViewOrientationMovesCamera jumps to a standard orientation and checks the camera eye
// moves onto the expected octant, and that an undefined orientation errors.
func TestSetViewOrientationMovesCamera(t *testing.T) {
	r, s := seededSession(t)
	var cam wire.CameraView
	call(t, r, s, "view.setOrientation", mustJSON(t, wire.SetOrientationArgs{
		Orientation: types.IsoTopRightViewOrientation, Fit: true,
	}), &cam)
	// Iso top-right looks from +X/+Y/+Z, so the eye is on the positive octant relative to target.
	if !(cam.Eye.X > cam.Target.X && cam.Eye.Y > cam.Target.Y && cam.Eye.Z > cam.Target.Z) {
		t.Errorf("iso-top-right eye %+v not on +++ octant of target %+v", cam.Eye, cam.Target)
	}
	if err := tryCall(t, r, s, "view.setOrientation", `{"orientation":0}`); err == nil {
		t.Error("an undefined orientation should error")
	}
}
