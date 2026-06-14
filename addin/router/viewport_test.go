// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestCaptureWindowFlagsRequest checks viewport.captureWindow flags a whole-window capture (the head
// services it after the swapchain composites) and returns the path it will write, so an MCP client
// can poll for the file.
func TestCaptureWindowFlagsRequest(t *testing.T) {
	r, s := seededSession(t)
	var res wire.CaptureWindowResult
	call(t, r, s, "viewport.captureWindow", `{"path":"/tmp/w.png"}`, &res)
	if res.Path != "/tmp/w.png" {
		t.Errorf("result path = %q, want /tmp/w.png", res.Path)
	}
	if path, ok := s.TakeWindowCapture(); !ok || path != "/tmp/w.png" {
		t.Errorf("captureWindow did not flag the request: (%q, %v)", path, ok)
	}
}

// TestCaptureWindowDefaultPath checks an empty Path falls back to the default whole-window file,
// distinct from the viewport-capture default so one never clobbers the other.
func TestCaptureWindowDefaultPath(t *testing.T) {
	r, s := seededSession(t)
	var res wire.CaptureWindowResult
	call(t, r, s, "viewport.captureWindow", `{}`, &res)
	if res.Path == "" || res.Path == defaultCapturePath {
		t.Errorf("default window path = %q, want a non-empty path distinct from the viewport default", res.Path)
	}
}
