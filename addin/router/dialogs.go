// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerDialogHandlers wires the host-dialog methods (M05-F08, #615).
func (r *Router) registerDialogHandlers() {
	r.readOnly(wire.MethodDialogsShowFileDialog, typed(showFileDialog))
	r.readOnly(wire.MethodDialogsShowWebDialog, typed(showWebDialog))
	r.readOnly(wire.MethodDialogsCloseWebDialog, typed(closeWebDialog))
	r.readOnly(wire.MethodDialogsListWebViews, listWebViews)
}

// showFileDialog queues a file-dialog ask; the choice arrives as dialog.fileChosen.
func showFileDialog(s *app.Session, in wire.ShowFileDialogArgs) (wire.OKResult, error) {
	err := s.RequestFileDialog(app.FileDialogRequest{
		ID: in.ID, Title: in.Title, Save: in.Save, Filter: in.Filter,
		FilterIndex: in.FilterIndex, InitialDir: in.InitialDir, MultiSelect: in.MultiSelect,
	})
	if err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func showWebDialog(s *app.Session, in wire.ShowWebDialogArgs) (wire.OKResult, error) {
	if err := s.ShowWebDialog(in.Dialog); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func closeWebDialog(s *app.Session, in wire.CloseWebDialogArgs) (wire.OKResult, error) {
	if err := s.CloseWebDialog(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func listWebViews(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListWebViewsResult{Views: s.WebViews()})
}
