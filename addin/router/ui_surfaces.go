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
	r.readOnly(wire.MethodBrowserSetPane, typed(setBrowserPane))
	r.readOnly(wire.MethodBrowserDeletePane, typed(deleteBrowserPane))
	r.readOnly(wire.MethodBrowserListPanes, listBrowserPanes)
	r.readOnly(wire.MethodDockableWindowsSet, typed(setDockableWindow))
	r.readOnly(wire.MethodDockableWindowsSetVisible, typed(setDockableWindowVisible))
	r.readOnly(wire.MethodDockableWindowsSetValue, typed(setDockableWindowValue))
	r.readOnly(wire.MethodDockableWindowsSetReferences, typed(setDockableWindowReferences))
	r.readOnly(wire.MethodDockableWindowsDelete, typed(deleteDockableWindow))
	r.readOnly(wire.MethodDockableWindowsList, listDockableWindows)
	r.readOnly(wire.MethodTaskPanelShow, typed(showTaskPanel))
	r.readOnly(wire.MethodTaskPanelClose, typed(closeTaskPanel))
	r.readOnly(wire.MethodUIListEnvironments, listEnvironments)
}

// setBrowserPane creates or replaces an add-in browser pane (wire browser.setPane).
func setBrowserPane(s *app.Session, in wire.SetBrowserPaneArgs) (wire.OKResult, error) {
	if err := s.BrowserPanes().Set(in.Pane); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// deleteBrowserPane removes an add-in browser pane (wire browser.deletePane).
func deleteBrowserPane(s *app.Session, in wire.DeleteBrowserPaneArgs) (wire.OKResult, error) {
	if err := s.BrowserPanes().Delete(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listBrowserPanes returns the add-in panes in creation order (wire browser.listPanes).
func listBrowserPanes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListBrowserPanesResult{Panes: s.BrowserPanes().List()})
}

// setDockableWindow creates or replaces an add-in dockable window
// (wire dockableWindows.set).
func setDockableWindow(s *app.Session, in wire.SetDockableWindowArgs) (wire.OKResult, error) {
	if err := s.SetDockableWindow(in.Window); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setDockableWindowVisible shows/hides an add-in window (wire dockableWindows.setVisible).
func setDockableWindowVisible(s *app.Session, in wire.SetDockableWindowVisibleArgs) (wire.OKResult, error) {
	if err := s.SetDockableWindowVisible(in.ID, in.Visible); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setDockableWindowValue drives one editable control of a dockable window to a value, exactly as a
// user edit would: the host updates the stored control and notifies the owning add-in
// (wire dockableWindows.setValue). Lets automation/MCP edit add-in panels (e.g. switch the CAM
// simulator's View dropdown).
func setDockableWindowValue(s *app.Session, in wire.SetDockableWindowValueArgs) (wire.OKResult, error) {
	s.PanelValueChanged(in.WindowId, in.ControlId, in.Value)
	return wire.OKResult{OK: true}, nil
}

// setDockableWindowReferences replaces a referenceList control's rows and notifies the owning
// add-in (wire dockableWindows.setReferences). Refs is the full new set.
func setDockableWindowReferences(s *app.Session, in wire.SetDockableWindowReferencesArgs) (wire.OKResult, error) {
	s.SetDockableWindowReferences(in.WindowId, in.ControlId, in.Refs)
	return wire.OKResult{OK: true}, nil
}

// showTaskPanel stores a modal task panel for the head to display (wire taskPanel.show).
func showTaskPanel(s *app.Session, in wire.ShowTaskPanelArgs) (wire.OKResult, error) {
	if err := s.ShowTaskPanel(in.Panel); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// closeTaskPanel removes a modal task panel programmatically (wire taskPanel.close).
func closeTaskPanel(s *app.Session, in wire.CloseTaskPanelArgs) (wire.OKResult, error) {
	if err := s.CloseTaskPanel(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// deleteDockableWindow removes an add-in window (wire dockableWindows.delete).
func deleteDockableWindow(s *app.Session, in wire.DeleteDockableWindowArgs) (wire.OKResult, error) {
	if err := s.DeleteDockableWindow(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listDockableWindows returns the add-in windows in creation order
// (wire dockableWindows.list).
func listDockableWindows(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListDockableWindowsResult{Windows: s.DockableWindows().List()})
}
