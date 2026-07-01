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
	r.readOnly(wire.MethodBrowserSetPane, setBrowserPane)
	r.readOnly(wire.MethodBrowserDeletePane, deleteBrowserPane)
	r.readOnly(wire.MethodBrowserListPanes, listBrowserPanes)
	r.readOnly(wire.MethodDockableWindowsSet, setDockableWindow)
	r.readOnly(wire.MethodDockableWindowsSetVisible, setDockableWindowVisible)
	r.readOnly(wire.MethodDockableWindowsSetValue, setDockableWindowValue)
	r.readOnly(wire.MethodDockableWindowsSetReferences, setDockableWindowReferences)
	r.readOnly(wire.MethodDockableWindowsDelete, deleteDockableWindow)
	r.readOnly(wire.MethodDockableWindowsList, listDockableWindows)
	r.readOnly(wire.MethodTaskPanelShow, showTaskPanel)
	r.readOnly(wire.MethodTaskPanelClose, closeTaskPanel)
	r.readOnly(wire.MethodUIListEnvironments, listEnvironments)
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

// setDockableWindowValue drives one editable control of a dockable window to a value, exactly as a
// user edit would: the host updates the stored control and notifies the owning add-in
// (wire dockableWindows.setValue). Lets automation/MCP edit add-in panels (e.g. switch the CAM
// simulator's View dropdown).
func setDockableWindowValue(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetDockableWindowValueArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.PanelValueChanged(req.WindowId, req.ControlId, req.Value)
	return ok()
}

// setDockableWindowReferences replaces a referenceList control's rows and notifies the owning
// add-in (wire dockableWindows.setReferences). Refs is the full new set.
func setDockableWindowReferences(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetDockableWindowReferencesArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.SetDockableWindowReferences(req.WindowId, req.ControlId, req.Refs)
	return ok()
}

// showTaskPanel stores a modal task panel for the head to display (wire taskPanel.show).
func showTaskPanel(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowTaskPanelArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ShowTaskPanel(req.Panel); err != nil {
		return nil, err
	}
	return ok()
}

// closeTaskPanel removes a modal task panel programmatically (wire taskPanel.close).
func closeTaskPanel(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.CloseTaskPanelArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.CloseTaskPanel(req.ID); err != nil {
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
