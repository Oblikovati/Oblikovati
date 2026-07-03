// SPDX-License-Identifier: GPL-2.0-only

package app

// Fit-the-camera-after-import intent (S11, #1645): an import that adds visible geometry can land
// entirely outside the current view frustum, so the viewport looks unchanged and the user concludes
// the importer did nothing (the georeferenced-DWG blank render, PR#1150; the PDF importer's recorded
// "Zoom-All" lesson). The model layer stays pure: instead of driving the camera directly it raises a
// one-shot request here — the same Request/Take convention the mesh and point-cloud imports use
// (mesh_import.go, point_cloud_import.go). The head polls TakeFitViewRequest once per frame and calls
// FitView (which fits the UNION of all visible geometry, so a small part inserted into a large
// assembly is framed with it, not yanked away). A headless/CLI import never polls, so it is unaffected.

// RequestFitView flags that an import added visible geometry and the camera should fit it once. It is
// idempotent within a frame (the flag coalesces multiple imports into one fit).
func (s *Session) RequestFitView() { s.fitViewRequested = true }

// TakeFitViewRequest returns and clears the pending fit-view request, so the head fits exactly once.
func (s *Session) TakeFitViewRequest() bool {
	req := s.fitViewRequested
	s.fitViewRequested = false
	return req
}
