// SPDX-License-Identifier: GPL-2.0-only

package app

// Mouse-navigation preferences surface (Inventor-parity work, 2026-08-17).
// Oblikovati ships Inventor's default bindings; these accessors let the head
// adapt the pointer gestures from persisted options without code changes.
// MMBMode/ShiftMMBMode swap the middle button between pan and orbit;
// WheelInvert flips the scroll direction; ZoomToCursor=false zooms to the view
// centre (cam.Dolly) instead of the cursor.

// MMBMode reports the middle-button drag mode ("pan" | "orbit").
func (s *Session) MMBMode() string {
	return s.appOptions.Input.MMBMode
}

// SetMMBMode persists the middle-button drag mode.
func (s *Session) SetMMBMode(v string) error {
	s.appOptions.Input.MMBMode = v
	return s.saveOptions()
}

// ShiftMMBMode reports the Shift+middle-button drag mode ("orbit" | "pan").
func (s *Session) ShiftMMBMode() string {
	return s.appOptions.Input.ShiftMMBMode
}

// SetShiftMMBMode persists the Shift+middle-button drag mode.
func (s *Session) SetShiftMMBMode(v string) error {
	s.appOptions.Input.ShiftMMBMode = v
	return s.saveOptions()
}

// CtrlMMBMode reports the Ctrl+middle-button drag mode ("pan" | "orbit" | "zoom").
// Inventor exposes the Control override alongside the plain and Shift mappings.
func (s *Session) CtrlMMBMode() string {
	return s.appOptions.Input.CtrlMMBMode
}

// SetCtrlMMBMode persists the Ctrl+middle-button drag mode.
func (s *Session) SetCtrlMMBMode(v string) error {
	s.appOptions.Input.CtrlMMBMode = v
	return s.saveOptions()
}

// WheelInvert reports whether the wheel zoom direction is inverted
// (true = scroll-up zooms out; Inventor default is scroll-up zooms in).
func (s *Session) WheelInvert() bool {
	return s.appOptions.Input.WheelInvert
}

// SetWheelInvert persists the wheel zoom direction.
func (s *Session) SetWheelInvert(v bool) error {
	s.appOptions.Input.WheelInvert = v
	return s.saveOptions()
}

// ZoomToCursor reports whether the wheel zooms toward the cursor
// (true, Inventor default) or toward the view centre (false).
func (s *Session) ZoomToCursor() bool {
	return s.appOptions.Input.ZoomToCursor
}

// SetZoomToCursor persists the wheel zoom anchor.
func (s *Session) SetZoomToCursor(v bool) error {
	s.appOptions.Input.ZoomToCursor = v
	return s.saveOptions()
}
