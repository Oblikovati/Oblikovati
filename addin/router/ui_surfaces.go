// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerUISurfaceHandlers wires the add-in UI surfaces of M05-F03: browser panes
// (#256), dockable windows and the environment listing (#247).
func (r *Router) registerUISurfaceHandlers() {
	r.handlers[wire.MethodBrowserSetPane] = setBrowserPane
	r.handlers[wire.MethodBrowserDeletePane] = deleteBrowserPane
	r.handlers[wire.MethodBrowserListPanes] = listBrowserPanes
	r.handlers[wire.MethodDockableWindowsSet] = setDockableWindow
	r.handlers[wire.MethodDockableWindowsSetVisible] = setDockableWindowVisible
	r.handlers[wire.MethodDockableWindowsDelete] = deleteDockableWindow
	r.handlers[wire.MethodDockableWindowsList] = listDockableWindows
	r.handlers[wire.MethodUIListEnvironments] = listEnvironments
}

// setBrowserPane creates or replaces an add-in browser pane (wire browser.setPane).
func setBrowserPane(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetBrowserPaneArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.BrowserPanes().Set(req.Pane); err != nil {
		return nil, err
	}
	return ok()
}

// deleteBrowserPane removes an add-in browser pane (wire browser.deletePane).
func deleteBrowserPane(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.DeleteBrowserPaneArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.BrowserPanes().Delete(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

// listBrowserPanes returns the add-in panes in creation order (wire browser.listPanes).
func listBrowserPanes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListBrowserPanesResult{Panes: s.BrowserPanes().List()})
}

// setDockableWindow creates or replaces an add-in dockable window
// (wire dockableWindows.set).
func setDockableWindow(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetDockableWindowArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetDockableWindow(req.Window); err != nil {
		return nil, err
	}
	return ok()
}

// setDockableWindowVisible shows/hides an add-in window (wire dockableWindows.setVisible).
func setDockableWindowVisible(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetDockableWindowVisibleArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetDockableWindowVisible(req.ID, req.Visible); err != nil {
		return nil, err
	}
	return ok()
}

// deleteDockableWindow removes an add-in window (wire dockableWindows.delete).
func deleteDockableWindow(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.DeleteDockableWindowArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.DeleteDockableWindow(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

// listDockableWindows returns the add-in windows in creation order
// (wire dockableWindows.list).
func listDockableWindows(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListDockableWindowsResult{Windows: s.DockableWindows().List()})
}
