// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

func camAt(eyeX float64) scene.Camera {
	c := scene.NewCamera(800, 600)
	c.Eye, c.Target, c.Up = math.P3(eyeX, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	return c
}

func TestPreviousViewRestoresRecordedCamera(t *testing.T) {
	s := NewSession()
	a := camAt(5)
	s.SetCamera(a)
	s.PushViewHistory() // record A
	s.SetCamera(camAt(20))

	s.PreviousView()
	if s.Camera().Eye != a.Eye {
		t.Errorf("PreviousView eye = %v, want the recorded %v", s.Camera().Eye, a.Eye)
	}
	if s.ViewHistoryDepth() != 0 {
		t.Errorf("history depth after restore = %d, want 0 (popped)", s.ViewHistoryDepth())
	}
	// A second PreviousView with empty history is a no-op (camera unchanged).
	before := s.Camera()
	s.PreviousView()
	if s.Camera() != before {
		t.Error("PreviousView with empty history should not move the camera")
	}
}

func TestViewHistoryCapsAtMax(t *testing.T) {
	s := NewSession()
	for i := range maxViewHistory + 10 {
		s.SetCamera(camAt(float64(i)))
		s.PushViewHistory()
	}
	if s.ViewHistoryDepth() != maxViewHistory {
		t.Errorf("history depth = %d, want it capped at %d", s.ViewHistoryDepth(), maxViewHistory)
	}
	// The most recent push survives the trim: Previous View returns the last camera recorded.
	s.SetCamera(camAt(999))
	s.PreviousView()
	if s.Camera().Eye != camAt(float64(maxViewHistory+9)).Eye {
		t.Errorf("after the cap, PreviousView eye = %v, want the most recent recorded", s.Camera().Eye)
	}
}

func TestFitViewRecordsHistory(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	start := s.Camera()
	s.FitView() // a discrete view change → records the pre-fit view
	if s.ViewHistoryDepth() != 1 {
		t.Fatalf("FitView should record one history entry, got %d", s.ViewHistoryDepth())
	}
	s.PreviousView()
	if s.Camera().Eye != start.Eye {
		t.Errorf("PreviousView after FitView eye = %v, want the pre-fit %v", s.Camera().Eye, start.Eye)
	}
}

func TestF5RestoresPreviousViewViaBinding(t *testing.T) {
	s := NewSession()
	a := camAt(3)
	s.SetCamera(a)
	s.PushViewHistory()
	s.SetCamera(camAt(40))

	if err := s.PressKey(KeyEvent{Key: "F5"}); err != nil {
		t.Fatalf("F5: %v", err)
	}
	if s.Camera().Eye != a.Eye {
		t.Errorf("F5 did not restore the previous view: eye=%v want=%v", s.Camera().Eye, a.Eye)
	}
}

func TestF6RunsHomeViaBinding(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	if err := s.PressKey(KeyEvent{Key: "F6"}); err != nil {
		t.Fatalf("F6: %v", err)
	}
	// Home swings the camera (animated); the binding resolved if a history entry was recorded.
	if s.ViewHistoryDepth() != 1 {
		t.Errorf("F6 (Home) should record the prior view: depth=%d, want 1", s.ViewHistoryDepth())
	}
}
