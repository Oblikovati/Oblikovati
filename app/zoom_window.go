// SPDX-License-Identifier: GPL-2.0-only

package app

// Zoom Window (Inventor's Zoom Area, N18→N16 in the viewport audit, #913): arm the tool, drag a
// rectangle over the viewport, and the view zooms to fit it. The rubber band reuses BoxSelection's
// geometry but is independent of box-select — no RegionPicker, no window/crossing hit semantics —
// because it changes the camera, not the selection. The camera math is scene.Camera.ZoomToRect.

// ArmZoomWindow arms the Zoom Window tool: the next left-drag over the viewport sweeps the zoom
// rectangle. It clears any stale rubber band.
func (s *Session) ArmZoomWindow() {
	s.zoomWindowArmed = true
	s.zoomWindow = BoxSelection{}
}

// ZoomWindowArmed reports whether the Zoom Window tool is waiting for (or running) a drag — the
// enable/active state for its command and the head's input routing.
func (s *Session) ZoomWindowArmed() bool { return s.zoomWindowArmed }

// DisarmZoomWindow cancels the tool and any in-progress rubber band (e.g. Esc).
func (s *Session) DisarmZoomWindow() {
	s.zoomWindowArmed = false
	s.zoomWindow = BoxSelection{}
}

// BeginZoomWindow anchors the zoom rectangle at the press point (no-op unless armed).
func (s *Session) BeginZoomWindow(x, y float64) {
	if !s.zoomWindowArmed {
		return
	}
	s.zoomWindow = BoxSelection{X0: x, Y0: y, X1: x, Y1: y, Active: true}
}

// UpdateZoomWindow moves the rectangle's free corner to the current cursor position.
func (s *Session) UpdateZoomWindow(x, y float64) {
	if s.zoomWindow.Active {
		s.zoomWindow.X1, s.zoomWindow.Y1 = x, y
	}
}

// ZoomWindowDragging reports whether a zoom rectangle is being swept (the head draws it then).
func (s *Session) ZoomWindowDragging() bool { return s.zoomWindow.Active }

// ZoomWindowRect returns the in-progress rectangle in viewport-local pixels for the head to draw.
func (s *Session) ZoomWindowRect() (x0, y0, x1, y1 float64) {
	b := s.zoomWindow
	return b.X0, b.Y0, b.X1, b.Y1
}

// CommitZoomWindow zooms the active view to the swept rectangle and disarms the tool. A degenerate
// rectangle (a click, or no drag) just disarms — ZoomToRect ignores it. The view change is recorded
// so Previous View returns.
func (s *Session) CommitZoomWindow() {
	b := s.zoomWindow
	s.DisarmZoomWindow()
	if !b.Active {
		return
	}
	cam := s.Camera().ZoomToRect(b.X0, b.Y0, b.X1, b.Y1)
	if cam == s.Camera() {
		return // degenerate box: ZoomToRect was a no-op, don't churn view history
	}
	s.PushViewHistory()
	s.SetCamera(cam)
}
