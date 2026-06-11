// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

func TestWindowsFramesReportMirroredState(t *testing.T) {
	r, s := seededSession(t)
	s.SetWindowFrameStatus(app.WindowFrameStatus{
		Caption: "test.obk — Oblikovati", State: types.WindowMaximized, Width: 2560, Height: 1480,
	})
	var res wire.ListViewFramesResult
	call(t, r, s, "windows.listFrames", "{}", &res)
	if len(res.Frames) != 1 || res.Frames[0].State != types.WindowMaximized || res.Frames[0].Width != 2560 {
		t.Fatalf("frames = %+v, want the mirrored maximized frame", res.Frames)
	}
}

func TestWindowsTabsActivateAndClose(t *testing.T) {
	r, s := seededSession(t)
	var tabs wire.ListViewTabsResult
	call(t, r, s, "windows.listTabs", "{}", &tabs)
	if len(tabs.Tabs) != 1 || !tabs.Tabs[0].Active {
		t.Fatalf("tabs = %+v, want the seeded active document", tabs.Tabs)
	}
	id := tabs.Tabs[0].Document

	call(t, r, s, "windows.activateTab", `{"document":`+itoaInt(int(id))+`}`, nil)
	if uint64(s.Workspace().ActiveDocument().ID()) != id {
		t.Fatal("activateTab did not activate")
	}

	call(t, r, s, "windows.closeTab", `{"document":`+itoaInt(int(id))+`,"force":true}`, nil)
	call(t, r, s, "windows.listTabs", "{}", &tabs)
	if len(tabs.Tabs) != 0 {
		t.Fatalf("tabs after close = %+v, want none", tabs.Tabs)
	}
	if _, err := r.Handle(s, "windows.activateTab", []byte(`{"document":999}`)); err == nil {
		t.Error("activating an unknown tab should fail")
	}
}
