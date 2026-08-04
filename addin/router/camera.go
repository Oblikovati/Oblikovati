// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
)

// getCamera returns a document's active-view camera as a look-at frame (Document 0 ⇒ the
// active document) — wire.MethodViewGetCamera.
func getCamera(s *app.Session, a wire.GetCameraArgs) (wire.CameraView, error) {
	cam, err := s.ViewCamera(a.Document)
	if err != nil {
		return wire.CameraView{}, err
	}
	return cameraViewOf(s, a.Document, cam)
}

// setCamera moves a document's active-view camera to the requested look-at frame, keeping
// the viewport size, and echoes the result (Document 0 ⇒ active document) —
// wire.MethodViewSetCamera. It rejects a non-finite or degenerate frame so a bad presenter
// packet can never corrupt the view.
func setCamera(s *app.Session, a wire.SetCameraArgs) (wire.CameraView, error) {
	if err := validateCameraArgs(a); err != nil {
		return wire.CameraView{}, err
	}
	cam, err := s.ViewCamera(a.Document) // preserve Width/Height; override the look-at frame
	if err != nil {
		return wire.CameraView{}, err
	}
	cam.Eye = math.P3(a.Eye.X, a.Eye.Y, a.Eye.Z)
	cam.Target = math.P3(a.Target.X, a.Target.Y, a.Target.Z)
	cam.Up = math.V3(a.Up.X, a.Up.Y, a.Up.Z)
	cam.FOV = a.FOV
	if err := s.SetViewCamera(a.Document, cam); err != nil {
		return wire.CameraView{}, err
	}
	// A zero Projection means "leave it": moving the camera must not silently reset how the view
	// projects just because the caller did not restate it.
	if a.Projection != 0 {
		if err := s.SetViewProjection(a.Document, a.Projection); err != nil {
			return wire.CameraView{}, err
		}
	}
	out, err := s.ViewCamera(a.Document)
	if err != nil {
		return wire.CameraView{}, err
	}
	return cameraViewOf(s, a.Document, out)
}

// cameraViewOf is cameraView with the view's projection attached — the projection lives on the
// view, not on the look-at frame, so it is read separately.
func cameraViewOf(s *app.Session, docID uint64, c app.CameraFrame) (wire.CameraView, error) {
	proj, err := s.ViewProjection(docID)
	if err != nil {
		return wire.CameraView{}, err
	}
	v := cameraView(c)
	v.Projection = proj
	return v, nil
}

// cameraView projects an app camera frame onto the wire look-at DTO.
func cameraView(c app.CameraFrame) wire.CameraView {
	return wire.CameraView{
		Eye:    types.Point{X: c.Eye.X, Y: c.Eye.Y, Z: c.Eye.Z},
		Target: types.Point{X: c.Target.X, Y: c.Target.Y, Z: c.Target.Z},
		Up:     types.Vector{X: c.Up.X, Y: c.Up.Y, Z: c.Up.Z},
		FOV:    c.FOV,
	}
}

// validateCameraArgs rejects a camera frame that would produce an invalid view: any
// non-finite component, a non-positive or ≥π field of view, a zero eye→target distance,
// a zero-length up vector, or an up parallel to the view direction.
func validateCameraArgs(a wire.SetCameraArgs) error {
	for _, v := range []float64{a.Eye.X, a.Eye.Y, a.Eye.Z, a.Target.X, a.Target.Y, a.Target.Z, a.Up.X, a.Up.Y, a.Up.Z, a.FOV} {
		if stdmath.IsNaN(v) || stdmath.IsInf(v, 0) {
			return fmt.Errorf("view.setCamera: non-finite camera component in %+v", a)
		}
	}
	if a.FOV <= 0 || a.FOV >= stdmath.Pi {
		return fmt.Errorf("view.setCamera: fov %v out of range (expected 0 < fov < π)", a.FOV)
	}
	eye := math.P3(a.Eye.X, a.Eye.Y, a.Eye.Z)
	target := math.P3(a.Target.X, a.Target.Y, a.Target.Z)
	up := math.V3(a.Up.X, a.Up.Y, a.Up.Z)
	const eps = 1e-9
	if eye.DistanceTo(target) < eps {
		return fmt.Errorf("view.setCamera: eye and target coincide at %v", a.Eye)
	}
	if up.Length() < eps {
		return fmt.Errorf("view.setCamera: up vector is zero %v", a.Up)
	}
	if up.IsParallelTo(eye.VectorTo(target), eps) {
		return fmt.Errorf("view.setCamera: up %v is parallel to the view direction", a.Up)
	}
	return nil
}
