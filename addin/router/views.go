// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// listViews enumerates a document's views with their cameras, the active index, and the
// tiling layout (Document 0 ⇒ active) — wire.MethodViewsList.
func listViews(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.ListViewsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(listViewsResult(d))
}

// addView creates a new view of a document and makes it active, returning it
// (wire.MethodViewsAdd).
func addView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.AddViewArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	i, err := s.AddView(a.Document, a.Name, a.CopyActiveCamera)
	if err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(viewInfo(i, d.Views().All()[i], i == d.Views().ActiveIndex()))
}

// activateView makes the indexed view active, returning the updated collection
// (wire.MethodViewsActivate).
func activateView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.ActivateViewArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	if err := d.Views().Activate(a.Index); err != nil {
		return nil, err
	}
	return json.Marshal(listViewsResult(d))
}

// closeView removes the indexed view (refused for the last view), returning the updated
// collection (wire.MethodViewsClose).
func closeView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.CloseViewArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	if err := d.Views().Close(a.Index); err != nil {
		return nil, err
	}
	return json.Marshal(listViewsResult(d))
}

// renameView sets the indexed view's name, returning it (wire.MethodViewsRename).
func renameView(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.RenameViewArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	if err := d.Views().Rename(a.Index, a.Name); err != nil {
		return nil, err
	}
	return json.Marshal(viewInfo(a.Index, d.Views().All()[a.Index], a.Index == d.Views().ActiveIndex()))
}

// getLayout returns a document's tiling layout (wire.MethodViewsGetLayout).
func getLayout(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.ListViewsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.LayoutResult{Document: a.Document, Layout: d.Views().Layout()})
}

// setLayout chooses how a document's views are tiled (wire.MethodViewsSetLayout).
func setLayout(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetLayoutArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	d, err := s.DocumentByID(a.Document)
	if err != nil {
		return nil, err
	}
	d.Views().SetLayout(a.Layout)
	return json.Marshal(wire.LayoutResult{Document: a.Document, Layout: d.Views().Layout()})
}

// listViewsResult renders a document's whole view collection into the wire DTO.
func listViewsResult(d *doc.Document) wire.ListViewsResult {
	vs := d.Views()
	all := vs.All()
	out := make([]wire.ViewInfo, len(all))
	for i, v := range all {
		out[i] = viewInfo(i, v, i == vs.ActiveIndex())
	}
	return wire.ListViewsResult{Views: out, ActiveIndex: vs.ActiveIndex(), Layout: vs.Layout()}
}

// viewInfo renders one view (index, name, active flag, camera) into the wire DTO.
func viewInfo(i int, v *doc.View, active bool) wire.ViewInfo {
	return wire.ViewInfo{
		Index:  i,
		Name:   v.Name,
		Active: active,
		Camera: wire.CameraView{
			Eye:    [3]float64{v.Eye.X, v.Eye.Y, v.Eye.Z},
			Target: [3]float64{v.Target.X, v.Target.Y, v.Target.Z},
			Up:     [3]float64{v.Up.X, v.Up.Y, v.Up.Z},
			FOV:    v.FOV,
		},
	}
}
