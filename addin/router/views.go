// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// listViews enumerates a document's views with their cameras, the active index, and the
// tiling layout (Document 0 ⇒ active) — wire.MethodViewsList.
func listViews(s *app.Session, a wire.ListViewsArgs) (wire.ListViewsResult, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.ListViewsResult{}, err
	}
	return listViewsResult(d, s.DisplayMode()), nil
}

// addView creates a new view of a document and makes it active, returning it
// (wire.MethodViewsAdd).
func addView(s *app.Session, a wire.AddViewArgs) (wire.ViewInfo, error) {
	i, err := s.AddView(a.Document, a.Name, a.CopyActiveCamera)
	if err != nil {
		return wire.ViewInfo{}, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.ViewInfo{}, err
	}
	return viewInfo(i, d.Views().All()[i], i == d.Views().ActiveIndex(), s.DisplayMode()), nil
}

// activateView makes the indexed view active, returning the updated collection
// (wire.MethodViewsActivate).
func activateView(s *app.Session, a wire.ActivateViewArgs) (wire.ListViewsResult, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.ListViewsResult{}, err
	}
	if err := d.Views().Activate(a.Index); err != nil {
		return wire.ListViewsResult{}, err
	}
	return listViewsResult(d, s.DisplayMode()), nil
}

// closeView removes the indexed view (refused for the last view), returning the updated
// collection (wire.MethodViewsClose).
func closeView(s *app.Session, a wire.CloseViewArgs) (wire.ListViewsResult, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.ListViewsResult{}, err
	}
	if err := d.Views().Close(a.Index); err != nil {
		return wire.ListViewsResult{}, err
	}
	return listViewsResult(d, s.DisplayMode()), nil
}

// renameView sets the indexed view's name, returning it (wire.MethodViewsRename).
func renameView(s *app.Session, a wire.RenameViewArgs) (wire.ViewInfo, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.ViewInfo{}, err
	}
	if err := d.Views().Rename(a.Index, a.Name); err != nil {
		return wire.ViewInfo{}, err
	}
	return viewInfo(a.Index, d.Views().All()[a.Index], a.Index == d.Views().ActiveIndex(), s.DisplayMode()), nil
}

// getLayout returns a document's tiling layout (wire.MethodViewsGetLayout).
func getLayout(s *app.Session, a wire.ListViewsArgs) (wire.LayoutResult, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.LayoutResult{}, err
	}
	return wire.LayoutResult{Document: a.Document, Layout: d.Views().Layout()}, nil
}

// setLayout chooses how a document's views are tiled (wire.MethodViewsSetLayout).
func setLayout(s *app.Session, a wire.SetLayoutArgs) (wire.LayoutResult, error) {
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return wire.LayoutResult{}, err
	}
	d.Views().SetLayout(a.Layout)
	return wire.LayoutResult{Document: a.Document, Layout: d.Views().Layout()}, nil
}

// listViewsResult renders a document's whole view collection into the wire DTO. The display
// mode is the session-wide current mode (views share one renderer today).
func listViewsResult(d *doc.Document, mode types.DisplayModeEnum) wire.ListViewsResult {
	vs := d.Views()
	all := vs.All()
	out := make([]wire.ViewInfo, len(all))
	for i, v := range all {
		out[i] = viewInfo(i, v, i == vs.ActiveIndex(), mode)
	}
	return wire.ListViewsResult{Views: out, ActiveIndex: vs.ActiveIndex(), Layout: vs.Layout()}
}

// viewInfo renders one view (index, name, active flag, type, camera, display mode) into the
// wire DTO — the camera + display-mode pair a client view carries (#409).
func viewInfo(i int, v *doc.View, active bool, mode types.DisplayModeEnum) wire.ViewInfo {
	return wire.ViewInfo{
		Index:    i,
		Name:     v.Name,
		Active:   active,
		ViewType: types.GraphicsViewType,
		Camera: wire.CameraView{
			Eye:    types.Point{X: v.Eye.X, Y: v.Eye.Y, Z: v.Eye.Z},
			Target: types.Point{X: v.Target.X, Y: v.Target.Y, Z: v.Target.Z},
			Up:     types.Vector{X: v.Up.X, Y: v.Up.Y, Z: v.Up.Z},
			FOV:    v.FOV,
		},
		DisplayMode: mode,
	}
}
