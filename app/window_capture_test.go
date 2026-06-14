// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestWindowCaptureRequestRoundTrip checks RequestWindowCapture flags a path that TakeWindowCapture
// then consumes exactly once — the one-shot seam the head's frame loop drains to write the
// whole-window PNG after the swapchain composites.
func TestWindowCaptureRequestRoundTrip(t *testing.T) {
	s := NewSession()
	if _, ok := s.TakeWindowCapture(); ok {
		t.Fatal("a fresh session should have no pending window capture")
	}
	s.RequestWindowCapture("/tmp/win.png")
	path, ok := s.TakeWindowCapture()
	if !ok || path != "/tmp/win.png" {
		t.Fatalf("TakeWindowCapture = (%q, %v), want (/tmp/win.png, true)", path, ok)
	}
	if _, ok := s.TakeWindowCapture(); ok {
		t.Error("the pending capture should be consumed after one Take")
	}
}

// TestWindowAndViewportCapturesAreIndependent checks the whole-window and viewport-only capture
// requests use separate slots, so requesting one never consumes or clobbers the other.
func TestWindowAndViewportCapturesAreIndependent(t *testing.T) {
	s := NewSession()
	s.RequestViewportCapture("/tmp/vp.png")
	s.RequestWindowCapture("/tmp/win.png")
	if path, ok := s.TakeWindowCapture(); !ok || path != "/tmp/win.png" {
		t.Errorf("TakeWindowCapture = (%q, %v), want the window path", path, ok)
	}
	if path, ok := s.TakeViewportCapture(); !ok || path != "/tmp/vp.png" {
		t.Errorf("TakeViewportCapture = (%q, %v), want the viewport path still pending", path, ok)
	}
}
