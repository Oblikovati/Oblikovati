// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerWindowHandlers wires the view-frame/tab methods (M05-F10, #617).
func (r *Router) registerWindowHandlers() {
	r.readOnly(wire.MethodWindowsListFrames, listViewFrames)
	r.readOnly(wire.MethodWindowsListTabs, listViewTabs)
	r.readOnly(wire.MethodWindowsActivateTab, typed(activateViewTab))
	r.readOnly(wire.MethodWindowsCloseTab, typed(closeViewTab))
}

// listViewFrames reports the host's top-level frames — exactly one on the
// single-frame head (wire windows.listFrames).
func listViewFrames(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	f := s.WindowFrameStatus()
	frame := wire.ViewFrameInfo{Caption: f.Caption, State: f.State, Width: f.Width, Height: f.Height}
	return json.Marshal(wire.ListViewFramesResult{Frames: []wire.ViewFrameInfo{frame}})
}

// listViewTabs reports the document tab strip (wire windows.listTabs).
func listViewTabs(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	ws := s.Workspace()
	active := ws.ActiveDocument()
	docs := ws.Documents()
	tabs := make([]wire.ViewTabInfo, len(docs))
	for i, d := range docs {
		tabs[i] = wire.ViewTabInfo{
			Document: uint64(d.ID()), Title: d.DisplayName(),
			Active: d == active, Dirty: d.Dirty(),
		}
	}
	return json.Marshal(wire.ListViewTabsResult{Tabs: tabs})
}

// activateViewTab brings a document tab to the front (wire windows.activateTab).
func activateViewTab(s *app.Session, in wire.ActivateViewTabArgs) (wire.OKResult, error) {
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == in.Document {
			if err := s.Workspace().SetActiveDocument(d); err != nil {
				return wire.OKResult{}, err
			}
			return wire.OKResult{OK: true}, nil
		}
	}
	return wire.OKResult{}, fmt.Errorf("no open document with id %d", in.Document)
}

// closeViewTab closes a document tab with documents.close's save-first/force
// semantics (wire windows.closeTab).
func closeViewTab(s *app.Session, in wire.CloseViewTabArgs) (wire.OKResult, error) {
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == in.Document {
			if err := s.Workspace().Close(d, in.Force); err != nil {
				return wire.OKResult{}, err
			}
			return wire.OKResult{OK: true}, nil
		}
	}
	return wire.OKResult{}, fmt.Errorf("no open document with id %d", in.Document)
}
