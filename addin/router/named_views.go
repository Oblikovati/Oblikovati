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
	r.readOnly(wire.MethodViewsCaptureNamed, typed(captureNamedView))
	r.readOnly(wire.MethodViewsListNamed, listNamedViews)
	r.readOnly(wire.MethodViewsRestoreNamed, typed(restoreNamedView))
	r.readOnly(wire.MethodViewsDeleteNamed, typed(deleteNamedView))
	r.readOnly(wire.MethodViewSetOrientation, typed(setViewOrientation))
}

// captureNamedView saves the active view's camera under a name (wire.MethodViewsCaptureNamed).
func captureNamedView(s *app.Session, in wire.CaptureNamedViewArgs) (wire.NamedViewInfo, error) {
	nv, err := s.CaptureNamedView(in.Name)
	if err != nil {
		return wire.NamedViewInfo{}, err
	}
	return namedViewInfo(nv), nil
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
func restoreNamedView(s *app.Session, in wire.NamedViewRefArgs) (wire.CameraView, error) {
	if err := s.RestoreNamedView(in.Name); err != nil {
		return wire.CameraView{}, err
	}
	return cameraView(s.CameraFrame()), nil
}

// deleteNamedView removes a saved named view (wire.MethodViewsDeleteNamed).
func deleteNamedView(s *app.Session, in wire.NamedViewRefArgs) (wire.OKResult, error) {
	if err := s.DeleteNamedView(in.Name); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setViewOrientation jumps the active view to a standard orientation, returning the resulting
// camera (wire.MethodViewSetOrientation).
func setViewOrientation(s *app.Session, in wire.SetOrientationArgs) (wire.CameraView, error) {
	if !in.Orientation.IsValid() {
		return wire.CameraView{}, fmt.Errorf("setViewOrientation: %d is not a defined ViewOrientationTypeEnum", int32(in.Orientation))
	}
	if err := s.SetViewOrientation(in.Orientation, in.Fit); err != nil {
		return wire.CameraView{}, err
	}
	return cameraView(s.CameraFrame()), nil
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
