// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// registerNamedViewHandlers wires named-view capture/restore and the standard-orientation jump
// (M16-F03, #404/#409).
func (r *Router) registerNamedViewHandlers() {
	r.handlers[wire.MethodViewsCaptureNamed] = captureNamedView
	r.handlers[wire.MethodViewsListNamed] = listNamedViews
	r.handlers[wire.MethodViewsRestoreNamed] = restoreNamedView
	r.handlers[wire.MethodViewsDeleteNamed] = deleteNamedView
	r.handlers[wire.MethodViewSetOrientation] = setViewOrientation
}

// captureNamedView saves the active view's camera under a name (wire.MethodViewsCaptureNamed).
func captureNamedView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.CaptureNamedViewArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	nv, err := s.CaptureNamedView(a.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(namedViewInfo(nv))
}

// listNamedViews enumerates the active document's saved named views (wire.MethodViewsListNamed).
func listNamedViews(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	views := s.NamedViews()
	out := make([]wire.NamedViewInfo, len(views))
	for i, nv := range views {
		out[i] = namedViewInfo(nv)
	}
	return json.Marshal(wire.NamedViewsResult{Views: out})
}

// restoreNamedView applies a saved named view's camera to the active view, returning the
// resulting frame (wire.MethodViewsRestoreNamed).
func restoreNamedView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.NamedViewRefArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.RestoreNamedView(a.Name); err != nil {
		return nil, err
	}
	return json.Marshal(cameraView(s.Camera()))
}

// deleteNamedView removes a saved named view (wire.MethodViewsDeleteNamed).
func deleteNamedView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.NamedViewRefArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.DeleteNamedView(a.Name); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// setViewOrientation jumps the active view to a standard orientation, returning the resulting
// camera (wire.MethodViewSetOrientation).
func setViewOrientation(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetOrientationArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if !a.Orientation.IsValid() {
		return nil, fmt.Errorf("setViewOrientation: %d is not a defined ViewOrientationTypeEnum", int32(a.Orientation))
	}
	if err := s.SetViewOrientation(a.Orientation, a.Fit); err != nil {
		return nil, err
	}
	return json.Marshal(cameraView(s.Camera()))
}

// namedViewInfo projects a saved named view into its wire shape.
func namedViewInfo(nv doc.NamedView) wire.NamedViewInfo {
	return wire.NamedViewInfo{
		Name: nv.Name,
		Camera: wire.CameraView{
			Eye:    types.Point{X: nv.Home.Eye.X, Y: nv.Home.Eye.Y, Z: nv.Home.Eye.Z},
			Target: types.Point{X: nv.Home.Target.X, Y: nv.Home.Target.Y, Z: nv.Home.Target.Z},
			Up:     types.Vector{X: nv.Home.Up.X, Y: nv.Home.Up.Y, Z: nv.Home.Up.Z},
			FOV:    nv.Home.FOV,
		},
	}
}
