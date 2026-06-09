// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	stdmath "math"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// getCamera returns a document's active-view camera as a look-at frame (Document 0 ⇒ the
// active document) — wire.MethodViewGetCamera.
func getCamera(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.GetCameraArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	cam, err := s.ViewCamera(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cameraView(cam))
}

// setCamera moves a document's active-view camera to the requested look-at frame, keeping
// the viewport size, and echoes the result (Document 0 ⇒ active document) —
// wire.MethodViewSetCamera. It rejects a non-finite or degenerate frame so a bad presenter
// packet can never corrupt the view.
func setCamera(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetCameraArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := validateCameraArgs(a); err != nil {
		return nil, err
	}
	cam, err := s.ViewCamera(a.Document) // preserve Width/Height; override the look-at frame
	if err != nil {
		return nil, err
	}
	cam.Eye = math.P3(a.Eye[0], a.Eye[1], a.Eye[2])
	cam.Target = math.P3(a.Target[0], a.Target[1], a.Target[2])
	cam.Up = math.V3(a.Up[0], a.Up[1], a.Up[2])
	cam.FOV = a.FOV
	if err := s.SetViewCamera(a.Document, cam); err != nil {
		return nil, err
	}
	out, err := s.ViewCamera(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cameraView(out))
}

// cameraView projects a scene.Camera onto the wire look-at DTO.
func cameraView(c scene.Camera) wire.CameraView {
	return wire.CameraView{
		Eye:    [3]float64{c.Eye.X, c.Eye.Y, c.Eye.Z},
		Target: [3]float64{c.Target.X, c.Target.Y, c.Target.Z},
		Up:     [3]float64{c.Up.X, c.Up.Y, c.Up.Z},
		FOV:    c.FOV,
	}
}

// validateCameraArgs rejects a camera frame that would produce an invalid view: any
// non-finite component, a non-positive or ≥π field of view, a zero eye→target distance,
// a zero-length up vector, or an up parallel to the view direction.
func validateCameraArgs(a wire.SetCameraArgs) error {
	for _, v := range []float64{a.Eye[0], a.Eye[1], a.Eye[2], a.Target[0], a.Target[1], a.Target[2], a.Up[0], a.Up[1], a.Up[2], a.FOV} {
		if stdmath.IsNaN(v) || stdmath.IsInf(v, 0) {
			return fmt.Errorf("view.setCamera: non-finite camera component in %+v", a)
		}
	}
	if a.FOV <= 0 || a.FOV >= stdmath.Pi {
		return fmt.Errorf("view.setCamera: fov %v out of range (expected 0 < fov < π)", a.FOV)
	}
	eye := math.P3(a.Eye[0], a.Eye[1], a.Eye[2])
	target := math.P3(a.Target[0], a.Target[1], a.Target[2])
	up := math.V3(a.Up[0], a.Up[1], a.Up[2])
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
