// SPDX-License-Identifier: GPL-2.0-only

package app

// Viewport capture is the headless "screenshot" seam: the viewport.capture wire method (and the MCP
// bridge that exposes it) flags a path here, and the head's render loop writes the framebuffer to it
// AFTER the frame renders — so the PNG is exactly what is on screen. One-shot, mirroring the Tools ▸
// Save Viewport PNG menu action; lets an add-in / Claude grab the rendered image without a human.

// RequestViewportCapture flags that the viewport framebuffer should be written to a PNG at path.
func (s *Session) RequestViewportCapture(path string) { s.capturePath = path }

// TakeViewportCapture returns the pending capture path and clears it (ok=false when none pending).
func (s *Session) TakeViewportCapture() (path string, ok bool) {
	if s.capturePath == "" {
		return "", false
	}
	path, s.capturePath = s.capturePath, ""
	return path, true
}

// RequestWindowCapture flags that the WHOLE application window (chrome + dialogs + viewport, the full
// swapchain image) should be written to a PNG at path — the headless way to SEE the UI state, not
// just the 3D render. Like [RequestViewportCapture] but the head reads back the swapchain after the
// frame composites.
func (s *Session) RequestWindowCapture(path string) { s.captureWindowPath = path }

// TakeWindowCapture returns the pending whole-window capture path and clears it (ok=false when none
// pending).
func (s *Session) TakeWindowCapture() (path string, ok bool) {
	if s.captureWindowPath == "" {
		return "", false
	}
	path, s.captureWindowPath = s.captureWindowPath, ""
	return path, true
}

// SetNormalDebug enables/disables the viewport's normal-debug render (front-facing GREEN, back-facing
// RED) — the headless way to inspect winding/orientation. The head applies it on the next frame.
func (s *Session) SetNormalDebug(on bool) { s.normalDebug = on }

// NormalDebug reports whether normal-debug render is enabled.
func (s *Session) NormalDebug() bool { return s.normalDebug }

// SetMeshColors enables/disables the viewport's mesh-debug-colors render — each B-rep face (or each
// TRIANGLE when perTriangle) painted a distinct, index-derived color, so a rendered region maps back
// to a face/triangle index in the mesh data.
func (s *Session) SetMeshColors(on, perTriangle bool) {
	s.meshColors, s.meshColorsPerTri = on, perTriangle
}

// MeshColors reports whether mesh-debug-colors render is enabled, and whether it colors per triangle.
func (s *Session) MeshColors() (on, perTriangle bool) { return s.meshColors, s.meshColorsPerTri }
