// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/persistence/dialogmemory"
)

// FakeDialogMemoryStore is a named in-memory dialogmemory.Store.
type FakeDialogMemoryStore struct {
	stored dialogmemory.Memory
	saves  int
}

func (f *FakeDialogMemoryStore) Load() (dialogmemory.Memory, error) { return f.stored, nil }
func (f *FakeDialogMemoryStore) Save(m dialogmemory.Memory) error {
	f.stored = m
	f.saves++
	return nil
}

func TestMessageCenterSectionsAndFlags(t *testing.T) {
	t.Parallel()
	m := NewMessageCenter()
	outer := m.BeginSection("Meshing")
	m.AddMessage("starting", types.SeverityInfo)
	inner := m.BeginSection("Face 12")
	m.AddMessage("degenerate face", types.SeverityWarning)
	if err := m.EndSection(outer); err == nil {
		t.Fatal("closing the outer section with the inner open should fail")
	}
	if err := m.EndSection(inner); err != nil {
		t.Fatalf("EndSection(inner): %v", err)
	}
	if err := m.EndSection(outer); err != nil {
		t.Fatalf("EndSection(outer): %v", err)
	}
	m.AddMessage("boolean failed", types.SeverityError)

	if !m.HasErrors() || !m.HasWarnings() || m.LastMessage() != "boolean failed" {
		t.Errorf("flags = errors %v warnings %v last %q, want both true + boolean failed",
			m.HasErrors(), m.HasWarnings(), m.LastMessage())
	}
	root := m.View()
	if len(root.Sections) != 1 || root.Sections[0].Title != "Meshing" ||
		len(root.Sections[0].Sections) != 1 || root.Sections[0].Sections[0].Messages[0].Text != "degenerate face" {
		t.Fatalf("tree = %+v, want Meshing ▸ Face 12 ▸ degenerate face", root)
	}

	m.Clear()
	if m.HasErrors() || len(m.View().Sections) != 0 {
		t.Error("Clear left state behind")
	}
}

func TestProgressLedgerLifecycleAndCancel(t *testing.T) {
	t.Parallel()
	s := NewSession()
	var cancelled []ProgressCancelled
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e ProgressCancelled) event.Outcome {
		cancelled = append(cancelled, e)
		return event.Continue()
	})

	outer, err := s.Progress().Begin(10, "outer")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	inner, err := s.Progress().Begin(4, "inner")
	if err != nil {
		t.Fatalf("Begin inner: %v", err)
	}
	if bar, ok := s.Progress().Innermost(); !ok || bar.ID != inner {
		t.Fatalf("Innermost = (%+v, %v), want the inner bar", bar, ok)
	}

	if err := s.CancelProgress(inner); err != nil {
		t.Fatalf("CancelProgress: %v", err)
	}
	got, err := s.Progress().Update(inner, 99, "")
	if err != nil || !got {
		t.Fatalf("Update after cancel = (%v, %v), want cancelled true", got, err)
	}
	if bar, _ := s.Progress().Innermost(); bar.Step != bar.Steps {
		t.Errorf("step = %d, want clamped to %d", bar.Step, bar.Steps)
	}
	if len(cancelled) != 1 || cancelled[0].ID != inner {
		t.Fatalf("events = %+v, want one cancel of the inner bar", cancelled)
	}

	if err := s.Progress().End(inner); err != nil {
		t.Fatalf("End inner: %v", err)
	}
	if bar, ok := s.Progress().Innermost(); !ok || bar.ID != outer {
		t.Errorf("after ending inner, innermost = (%+v, %v), want the outer bar", bar, ok)
	}
	if _, err := s.Progress().Begin(0, "bad"); err == nil {
		t.Error("Begin(0 steps) should fail")
	}
}

func TestBalloonTipsSuppressionPersists(t *testing.T) {
	t.Parallel()
	store := &FakeDialogMemoryStore{}
	s := NewSession()
	if err := s.UseDialogMemoryStore(store); err != nil {
		t.Fatalf("UseDialogMemoryStore: %v", err)
	}
	if err := s.BalloonTips().Register(BalloonTipSpec{ID: "sim.done", Title: "Done", Text: "Finished"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	shown, err := s.ShowBalloonTip("sim.done")
	if err != nil || !shown {
		t.Fatalf("Show = (%v, %v), want shown", shown, err)
	}
	s.DismissBalloonTip("sim.done", true) // "don't show again"
	if shown, _ := s.ShowBalloonTip("sim.done"); shown {
		t.Error("a suppressed tip should not show")
	}
	if len(store.stored.SuppressedTips) != 1 || store.stored.SuppressedTips[0] != "sim.done" {
		t.Errorf("stored = %+v, want the suppression persisted", store.stored)
	}

	// A new session over the same store starts suppressed.
	s2 := NewSession()
	if err := s2.UseDialogMemoryStore(store); err != nil {
		t.Fatalf("UseDialogMemoryStore(2): %v", err)
	}
	if err := s2.BalloonTips().Register(BalloonTipSpec{ID: "sim.done", Title: "Done", Text: "Finished"}); err != nil {
		t.Fatalf("Register(2): %v", err)
	}
	if shown, _ := s2.ShowBalloonTip("sim.done"); shown {
		t.Error("suppression did not survive the session boundary")
	}
}

func TestBalloonTipClickEmitsAndDismisses(t *testing.T) {
	t.Parallel()
	s := NewSession()
	var clicks []BalloonTipClicked
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e BalloonTipClicked) event.Outcome {
		clicks = append(clicks, e)
		return event.Continue()
	})
	_ = s.BalloonTips().Register(BalloonTipSpec{ID: "x", Title: "T", Text: "B"})
	_, _ = s.ShowBalloonTip("x")
	s.ClickBalloonTip("x")
	if len(clicks) != 1 || clicks[0].ID != "x" {
		t.Fatalf("clicks = %+v, want one on x", clicks)
	}
	if len(s.BalloonTips().Active()) != 0 {
		t.Error("a clicked balloon should dismiss")
	}
}

func TestPromptCenterRememberedAnswers(t *testing.T) {
	t.Parallel()
	store := &FakeDialogMemoryStore{}
	s := NewSession()
	if err := s.UseDialogMemoryStore(store); err != nil {
		t.Fatalf("UseDialogMemoryStore: %v", err)
	}
	var answered []PromptAnswered
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e PromptAnswered) event.Outcome {
		answered = append(answered, e)
		return event.Continue()
	})

	spec := PromptSpec{
		ID: "sim.replace", Message: "Replace results?", Buttons: []string{"Replace", "Keep"},
		Restriction: types.PromptAllowRemember,
	}
	resolved, _, err := s.ShowPrompt(spec)
	if err != nil || resolved {
		t.Fatalf("ShowPrompt = (%v, %v), want pending", resolved, err)
	}
	if err := s.AnswerPrompt("sim.replace", "Banana", true); err == nil {
		t.Fatal("an answer that is not a button should fail")
	}
	if err := s.AnswerPrompt("sim.replace", "Replace", true); err != nil {
		t.Fatalf("AnswerPrompt: %v", err)
	}
	if len(answered) != 1 || answered[0].Answer != "Replace" || !answered[0].Remembered {
		t.Fatalf("events = %+v, want remembered Replace", answered)
	}

	// The remembered answer now resolves instantly — and survives sessions.
	resolved, answer, err := s.ShowPrompt(spec)
	if err != nil || !resolved || answer != "Replace" {
		t.Fatalf("ShowPrompt(again) = (%v, %q, %v), want instant Replace", resolved, answer, err)
	}
	s2 := NewSession()
	if err := s2.UseDialogMemoryStore(store); err != nil {
		t.Fatalf("UseDialogMemoryStore(2): %v", err)
	}
	if resolved, answer, _ := s2.ShowPrompt(spec); !resolved || answer != "Replace" {
		t.Error("remembered answer did not survive the session boundary")
	}
}

func TestPromptAlwaysAskNeverRemembers(t *testing.T) {
	t.Parallel()
	s := NewSession()
	spec := PromptSpec{ID: "p", Message: "m", Buttons: []string{"OK"}}
	if resolved, _, _ := s.ShowPrompt(spec); resolved {
		t.Fatal("nothing remembered yet")
	}
	if err := s.AnswerPrompt("p", "OK", true); err != nil { // remember asked but not allowed
		t.Fatalf("AnswerPrompt: %v", err)
	}
	if resolved, _, _ := s.ShowPrompt(spec); resolved {
		t.Error("an always-ask prompt must not resolve from memory")
	}
}
