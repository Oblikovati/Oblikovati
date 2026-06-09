// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/math"

// View navigation commands operate on the session camera. FitView frames the active
// part in the viewport keeping the current orientation (Inventor's Zoom All); HomeView
// switches to the default isometric view and frames it. Both are no-ops on an empty
// model. A ribbon command's Run calls these, and a test can call them directly.

// FitView reframes the camera so the whole active part fits the viewport. It writes
// through the active view (SetCamera) so the framing is remembered per view.
func (s *Session) FitView() { s.SetCamera(s.Camera().Fit(s.modelBounds())) }

// HomeView switches to the default isometric view, framed to fit the active part, writing
// through the active view so the framing is remembered.
func (s *Session) HomeView() { s.SetCamera(s.Camera().Home(s.modelBounds())) }

// modelBounds is the union of the active part's body bounding boxes (empty if none).
func (s *Session) modelBounds() math.Box {
	box := math.EmptyBox()
	for _, b := range s.sceneBodies() {
		box = box.Union(b.RangeBox())
	}
	return box
}
