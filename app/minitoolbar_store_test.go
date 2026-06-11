// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

func probeToolbar(command string) wire.MiniToolbarSpec {
	return wire.MiniToolbarSpec{
		ID: "sim.probe", Command: command, Visible: true, HeadsUpText: "Probe",
		ShowOK: true, ShowCancel: true,
		Controls: []wire.MiniToolbarControlSpec{
			{Kind: types.MiniToolbarValueEditor, ID: "depth", Label: "Depth", Value: "10 mm"},
			{Kind: types.MiniToolbarSlider, ID: "blend", Number: 0.5, Min: 0, Max: 1},
		},
	}
}

func TestMiniToolbarSetUpdateRemove(t *testing.T) {
	s := NewSession()
	if err := s.SetMiniToolbar(probeToolbar("")); err != nil {
		t.Fatalf("SetMiniToolbar: %v", err)
	}
	if err := s.UpdateMiniToolbarControls("sim.probe", []wire.MiniToolbarControlSpec{
		{ID: "depth", Value: "12 mm"},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	tb, ok := s.MiniToolbars().Get("sim.probe")
	if !ok || tb.Controls[0].Value != "12 mm" {
		t.Fatalf("toolbar = %+v, want depth merged to 12 mm", tb)
	}
	if err := s.UpdateMiniToolbarControls("sim.probe", []wire.MiniToolbarControlSpec{{ID: "ghost"}}); err == nil {
		t.Error("updating an unknown control should fail")
	}
	if err := s.RemoveMiniToolbar("sim.probe"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(s.MiniToolbars().List()) != 0 {
		t.Error("toolbar survived Remove")
	}
}

func TestMiniToolbarChangeAndCommitEvents(t *testing.T) {
	s := NewSession()
	var changes []MiniToolbarControlChanged
	var commits []MiniToolbarCommitted
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e MiniToolbarControlChanged) event.Outcome {
		changes = append(changes, e)
		return event.Continue()
	})
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e MiniToolbarCommitted) event.Outcome {
		commits = append(commits, e)
		return event.Continue()
	})
	if err := s.SetMiniToolbar(probeToolbar("")); err != nil {
		t.Fatalf("SetMiniToolbar: %v", err)
	}

	if err := s.ChangeMiniToolbarControl("sim.probe", wire.MiniToolbarControlSpec{
		ID: "blend", Number: 0.75,
	}); err != nil {
		t.Fatalf("Change: %v", err)
	}
	if len(changes) != 1 || changes[0].Control != "blend" || changes[0].Number != 0.75 {
		t.Fatalf("changes = %+v, want blend 0.75", changes)
	}
	if tb, _ := s.MiniToolbars().Get("sim.probe"); tb.Controls[1].Number != 0.75 {
		t.Error("the change did not land in the stored spec")
	}

	if err := s.CommitMiniToolbar("sim.probe", MiniToolbarApply); err != nil {
		t.Fatalf("Commit apply: %v", err)
	}
	if len(s.MiniToolbars().List()) != 1 {
		t.Error("apply must keep the toolbar")
	}
	if err := s.CommitMiniToolbar("sim.probe", MiniToolbarOK); err != nil {
		t.Fatalf("Commit ok: %v", err)
	}
	if len(s.MiniToolbars().List()) != 0 {
		t.Error("ok must dismiss the toolbar")
	}
	if len(commits) != 2 || commits[0].Gesture != "apply" || commits[1].Gesture != "ok" {
		t.Fatalf("commits = %+v, want apply then ok", commits)
	}
}

// FakeMiniToolbarTool is a named minimal tool for the lifecycle test.
type FakeMiniToolbarTool struct{}

func (FakeMiniToolbarTool) Name() string              { return "fake" }
func (FakeMiniToolbarTool) Start(*Session)            {}
func (FakeMiniToolbarTool) Pick(*Session, Selectable) {}
func (FakeMiniToolbarTool) CanCommit() bool           { return true }
func (FakeMiniToolbarTool) Commit(*Session) error     { return nil }
func (FakeMiniToolbarTool) Cancel(*Session)           {}

func TestCommandBoundMiniToolbarDiesWithTool(t *testing.T) {
	s := NewSession()
	s.StartTool(FakeMiniToolbarTool{})
	if err := s.SetMiniToolbar(probeToolbar("Sim.Probe")); err != nil {
		t.Fatalf("SetMiniToolbar: %v", err)
	}
	free := probeToolbar("")
	free.ID = "sim.free"
	if err := s.SetMiniToolbar(free); err != nil {
		t.Fatalf("SetMiniToolbar(free): %v", err)
	}

	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	left := s.MiniToolbars().List()
	if len(left) != 1 || left[0].ID != "sim.free" {
		t.Fatalf("toolbars after commit = %+v, want only the unbound one", left)
	}
}
