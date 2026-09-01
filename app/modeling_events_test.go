// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// collectFeatureEvents subscribes to FeatureLifecycleChanged on the session bus and returns a
// pointer to the growing slice of events seen.
func collectFeatureEvents(s *Session) *[]FeatureLifecycleChanged {
	var got []FeatureLifecycleChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e FeatureLifecycleChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	return &got
}

// fakeAddTool is a producer tool: its AddedFeature reports the feature it "created", so the
// tool-commit seam fires featureAdded.
type fakeAddTool struct {
	dialogTool
	added *feature.PartFeature
}

func (fakeAddTool) Name() string                          { return "Fake Add" }
func (fakeAddTool) CanCommit() bool                       { return true }
func (fakeAddTool) Commit(*Session) error                 { return nil }
func (t *fakeAddTool) AddedFeature() *feature.PartFeature { return t.added }

// fakePlainTool is a non-producer tool (no AddedFeature) — committing it must emit nothing.
type fakePlainTool struct{ dialogTool }

func (fakePlainTool) Name() string          { return "Fake Plain" }
func (fakePlainTool) CanCommit() bool       { return true }
func (fakePlainTool) Commit(*Session) error { return nil }

// TestToolCommitEmitsFeatureAdded: committing a producer tool through Session.OK fires a single
// FeatureAdded carrying the created feature's identity (#1085, the UI-driven creation path).
func TestToolCommitEmitsFeatureAdded(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	f := feature.NewHullFeatures(def.Features()).Add()
	got := collectFeatureEvents(s)

	s.StartTool(&fakeAddTool{added: f})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Op != FeatureAdded || (*got)[0].Feature != uint64(f.ID()) {
		t.Fatalf("events = %+v, want one FeatureAdded for feature %d", *got, f.ID())
	}
}

// TestNonProducerToolCommitEmitsNothing: a tool without AddedFeature emits no feature-lifecycle
// event when committed, and neither does a producer that built nothing (nil feature) (#1085).
func TestNonProducerToolCommitEmitsNothing(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	got := collectFeatureEvents(s)

	s.StartTool(fakePlainTool{})
	if err := s.OK(); err != nil {
		t.Fatalf("OK plain: %v", err)
	}
	s.StartTool(&fakeAddTool{added: nil})
	if err := s.OK(); err != nil {
		t.Fatalf("OK nil-producer: %v", err)
	}
	if len(*got) != 0 {
		t.Errorf("emitted %d feature events for non-producing commits, want 0: %+v", len(*got), *got)
	}
}

// TestDeleteFeatureEmitsFeatureDeleted: Session.DeleteFeature (UI browser + router) fires a single
// FeatureDeleted carrying the removed feature's identity (#1085).
func TestDeleteFeatureEmitsFeatureDeleted(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	f := feature.NewHullFeatures(def.Features()).Add()
	got := collectFeatureEvents(s)

	if err := s.DeleteFeature(f); err != nil {
		t.Fatalf("DeleteFeature: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Op != FeatureDeleted || (*got)[0].Feature != uint64(f.ID()) {
		t.Fatalf("events = %+v, want one FeatureDeleted for feature %d", *got, f.ID())
	}
}

// TestCommitFeatureEditEmitsFeatureEdited: Session.CommitFeatureEdit (router/freeform edit seam)
// fires a single FeatureEdited carrying the edited feature's identity (#1085).
func TestCommitFeatureEditEmitsFeatureEdited(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	f := feature.NewHullFeatures(def.Features()).Add()
	got := collectFeatureEvents(s)

	if err := s.CommitFeatureEdit(f); err != nil {
		t.Fatalf("CommitFeatureEdit: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Op != FeatureEdited || (*got)[0].Feature != uint64(f.ID()) {
		t.Fatalf("events = %+v, want one FeatureEdited for feature %d", *got, f.ID())
	}
}

// TestEnterExitSketchEmitsSketchEdit: entering and leaving a 2D sketch each emit a
// SketchEditChanged carrying the sketch identity, with Entered distinguishing them (#148).
func TestEnterExitSketchEmitsSketchEdit(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())

	var got []SketchEditChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e SketchEditChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	s.EnterSketch(sk)
	s.ExitSketch()

	if len(got) != 2 {
		t.Fatalf("emitted %d sketch-edit events, want 2 (enter, exit)", len(got))
	}
	if !got[0].Entered || got[1].Entered {
		t.Errorf("entered flags = %v/%v, want true then false", got[0].Entered, got[1].Entered)
	}
	if got[0].Sketch != sk.Seq() || got[0].Sketch == 0 {
		t.Errorf("sketch id = %d, want the sketch's seq %d", got[0].Sketch, sk.Seq())
	}
}
