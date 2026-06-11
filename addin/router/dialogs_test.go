// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestShowFileDialogQueuesRequest(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "dialogs.showFileDialog",
		`{"id":"sim.report","title":"Save report","save":true,"filter":"HTML (*.html)|*.html"}`, nil)
	pending, ok := s.PendingFileDialog()
	if !ok || pending.ID != "sim.report" || !pending.Save || pending.Filter == "" {
		t.Fatalf("pending = (%+v, %v), want the queued save ask", pending, ok)
	}
	if _, err := r.Handle(s, "dialogs.showFileDialog", []byte(`{"title":"no id"}`)); err == nil {
		t.Error("a request without an id should fail")
	}
}

func TestWebDialogsOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "dialogs.showWebDialog",
		`{"dialog":{"id":"docs","title":"Help","url":"https://example.org","dock":2,"visible":true}}`, nil)

	var lst wire.ListWebViewsResult
	call(t, r, s, "dialogs.listWebViews", "{}", &lst)
	if len(lst.Views) != 1 || lst.Views[0].Dock != types.DockRight || !lst.Views[0].Visible {
		t.Fatalf("views = %+v, want one visible dock-right view", lst.Views)
	}

	call(t, r, s, "dialogs.closeWebDialog", `{"id":"docs"}`, nil)
	call(t, r, s, "dialogs.listWebViews", "{}", &lst)
	if len(lst.Views) != 0 {
		t.Fatalf("views after close = %+v, want none", lst.Views)
	}
	if _, err := r.Handle(s, "dialogs.closeWebDialog", []byte(`{"id":"docs"}`)); err == nil {
		t.Error("closing an unknown view should fail")
	}
}
