// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerDialogHandlers wires the host-dialog methods (M05-F08, #615).
func (r *Router) registerDialogHandlers() {
	r.readOnly(wire.MethodDialogsShowFileDialog, showFileDialog)
	r.readOnly(wire.MethodDialogsShowWebDialog, showWebDialog)
	r.readOnly(wire.MethodDialogsCloseWebDialog, closeWebDialog)
	r.readOnly(wire.MethodDialogsListWebViews, listWebViews)
}

// showFileDialog queues a file-dialog ask; the choice arrives as dialog.fileChosen.
func showFileDialog(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowFileDialogArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	err := s.RequestFileDialog(app.FileDialogRequest{
		ID: req.ID, Title: req.Title, Save: req.Save, Filter: req.Filter,
		FilterIndex: req.FilterIndex, InitialDir: req.InitialDir, MultiSelect: req.MultiSelect,
	})
	if err != nil {
		return nil, err
	}
	return ok()
}

func showWebDialog(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowWebDialogArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ShowWebDialog(req.Dialog); err != nil {
		return nil, err
	}
	return ok()
}

func closeWebDialog(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.CloseWebDialogArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.CloseWebDialog(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

func listWebViews(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListWebViewsResult{Views: s.WebViews()})
}
