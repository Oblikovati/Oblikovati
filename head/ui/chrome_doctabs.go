//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/math"
)

// prevFramedDoc tracks which document the camera was last framed to, so switching
// the active document (New Part, a tab, an add-in) reframes the view to the new one.
var prevFramedDoc uint64

// followActiveDocument reframes the viewport when the active document changes, so a
// switch actually shows the new document. Without this the camera stays on the
// previous document's view — and since an empty new part has no bounds to fit, it
// would look like an empty viewport.
func followActiveDocument(s *app.Session) {
	active := s.ActiveDocument()
	var cur uint64
	if active != nil {
		cur = uint64(active.ID())
	}
	if cur == prevFramedDoc {
		return
	}
	prevFramedDoc = cur
	if active == nil || s.CameraAnimating() {
		return
	}
	// Frame the geometry if there is any; otherwise center on the model origin so the
	// part's coordinate planes are visible rather than a void (Home keeps the current
	// target on an empty model, so point it at the origin first).
	if len(activeBodies(s)) == 0 {
		cam := s.Camera()
		cam.Target = math.P3(0, 0, 0)
		s.SetCamera(cam)
	}
	s.HomeView()
}

// prevActiveDoc tracks the active document id across frames so a programmatic switch
// (e.g. an add-in activating a document) force-selects its tab once, without fighting
// the user's own tab clicks on subsequent frames.
var prevActiveDoc uint64

// drawDocumentTabs renders one tab per open document at the top of the viewport. The
// active document's tab is shown selected; clicking another tab activates that
// document, and when the active document changes elsewhere (an add-in, the menu) its
// tab is selected so the strip follows. The tabs read the workspace each frame, so
// opening/closing/activating documents is reflected automatically.
func drawDocumentTabs(s *app.Session) {
	docs := s.Workspace().Documents()
	if len(docs) == 0 {
		return
	}
	active := s.ActiveDocument()
	var cur uint64
	if active != nil {
		cur = uint64(active.ID())
	}
	// Force-select the active tab only on the frame the active document changed out
	// from under the UI (a New Part, an add-in's activate_document); otherwise leave
	// selection to the user's clicks.
	force := cur != prevActiveDoc
	prevActiveDoc = cur

	if native.BeginTabBar("##doc-tabs") {
		for _, d := range docs {
			selected := force && active != nil && d.ID() == active.ID()
			open := native.BeginTabItemSelected(d.DisplayName(), selected)
			// Only treat a tab as a user click on non-force frames. On a force frame we
			// are asserting the active tab via SetSelected, which ImGui applies on the
			// NEXT frame — so this frame BeginTabItem still reports the OLD visible tab.
			// Acting on that would call SetActiveDocument for the old document and flip
			// the active doc back, oscillating every frame.
			if open && !force && (active == nil || d.ID() != active.ID()) {
				_ = s.Workspace().SetActiveDocument(d)
			}
			if open {
				native.EndTabItem()
			}
		}
		native.EndTabBar()
	}
}
