// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strconv"
	"testing"

	"oblikovati.org/api/wire"
)

// itoaInt renders a plain int for the JSON one-liners (itoa in this package's
// tests is the uint64 entity-id helper).
func itoaInt(n int) string { return strconv.Itoa(n) }

func TestStatusTextOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "status.setText", `{"text":"Meshing…"}`, nil)
	var res wire.StatusTextResult
	call(t, r, s, "status.getText", "{}", &res)
	if res.Text != "Meshing…" {
		t.Fatalf("text = %q, want Meshing…", res.Text)
	}
}

func TestProgressOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var bar wire.BeginProgressResult
	call(t, r, s, "progress.begin", `{"steps":10,"message":"Meshing"}`, &bar)
	if bar.ID == 0 {
		t.Fatal("begin returned id 0")
	}

	var upd wire.UpdateProgressResult
	call(t, r, s, "progress.update", `{"id":`+itoaInt(bar.ID)+`,"step":4}`, &upd)
	if upd.Cancelled {
		t.Fatal("not cancelled yet")
	}
	if err := s.CancelProgress(bar.ID); err != nil {
		t.Fatalf("CancelProgress: %v", err)
	}
	call(t, r, s, "progress.update", `{"id":`+itoaInt(bar.ID)+`,"step":5}`, &upd)
	if !upd.Cancelled {
		t.Fatal("update should report the cancel")
	}
	call(t, r, s, "progress.end", `{"id":`+itoaInt(bar.ID)+`}`, nil)
	if _, err := r.Handle(s, "progress.update", []byte(`{"id":`+itoaInt(bar.ID)+`,"step":6}`)); err == nil {
		t.Error("updating an ended bar should fail")
	}
}

func TestBalloonTipsOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "balloonTip.register", `{"id":"sim.done","title":"Done","text":"Finished"}`, nil)
	var shown wire.ShowBalloonTipResult
	call(t, r, s, "balloonTip.show", `{"id":"sim.done"}`, &shown)
	if !shown.Shown {
		t.Fatal("first show should display")
	}
	s.DismissBalloonTip("sim.done", true)
	call(t, r, s, "balloonTip.show", `{"id":"sim.done"}`, &shown)
	if shown.Shown {
		t.Fatal("a suppressed tip should report shown=false")
	}
	if _, err := r.Handle(s, "balloonTip.show", []byte(`{"id":"ghost"}`)); err == nil {
		t.Error("showing an unregistered tip should fail")
	}
}

func TestPromptsOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var res wire.ShowPromptResult
	call(t, r, s, "prompts.show",
		`{"id":"p","message":"Replace?","buttons":["Replace","Keep"],"restriction":1}`, &res)
	if res.Resolved {
		t.Fatal("first show should be pending")
	}
	if err := s.AnswerPrompt("p", "Keep", true); err != nil {
		t.Fatalf("AnswerPrompt: %v", err)
	}
	call(t, r, s, "prompts.show",
		`{"id":"p","message":"Replace?","buttons":["Replace","Keep"],"restriction":1}`, &res)
	if !res.Resolved || res.Answer != "Keep" {
		t.Fatalf("second show = %+v, want the remembered Keep", res)
	}
}

func TestErrorsOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var sec wire.BeginMessageSectionResult
	call(t, r, s, "errors.beginSection", `{"title":"Meshing"}`, &sec)
	call(t, r, s, "errors.addMessage", `{"text":"degenerate face","severity":1}`, nil)
	call(t, r, s, "errors.endSection", `{"section":`+itoaInt(sec.Section)+`}`, nil)
	call(t, r, s, "errors.addMessage", `{"text":"boolean failed","severity":2}`, nil)

	var lst wire.ListErrorsResult
	call(t, r, s, "errors.list", "{}", &lst)
	if !lst.HasErrors || !lst.HasWarnings || lst.LastMessage != "boolean failed" {
		t.Fatalf("list = %+v, want both flags + last message", lst)
	}
	if len(lst.Root.Sections) != 1 || lst.Root.Sections[0].Messages[0].Text != "degenerate face" {
		t.Fatalf("tree = %+v, want the Meshing section", lst.Root)
	}

	call(t, r, s, "errors.show", "{}", nil)
	if !s.MessageCenterOpen() {
		t.Error("errors.show should open the panel")
	}
	call(t, r, s, "errors.clear", "{}", nil)
	var cleared wire.ListErrorsResult // fresh target: unmarshal keeps absent fields
	call(t, r, s, "errors.list", "{}", &cleared)
	if cleared.HasErrors || len(cleared.Root.Sections) != 0 {
		t.Error("clear left state behind")
	}
}
