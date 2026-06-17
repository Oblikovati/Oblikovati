//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
)

var closeDocumentModal documentCloseGuard

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
	// Camera state is per-view now: a view that has already been framed (saved, loaded, or
	// navigated) keeps its camera, so switching documents restores each one's view instead
	// of resetting it. Only a brand-new, never-framed view is Home-fit on first show.
	if v := s.ActiveView(); v == nil || v.Framed {
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
	// Only visible documents get a tab: a component loaded in the background for placement is
	// referenced in memory but shows no tab until the user opens it via Edit (#764).
	docs := s.Workspace().VisibleDocuments()
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
			drawDocumentTab(s, d, active, force)
		}
		native.EndTabBar()
	}
}

func drawDocumentTab(s *app.Session, d *doc.Document, active *doc.Document, force bool) {
	selected := force && active != nil && d.ID() == active.ID()
	open, keep := native.BeginTabItemClosable(documentTabLabel(d), selected)
	if open && keep && !force && (active == nil || d.ID() != active.ID()) {
		_ = s.Workspace().SetActiveDocument(d)
	}
	if open {
		native.EndTabItem()
	}
	if !keep && closeDocumentModal.request(d) {
		closeDocumentNow(s, d, false)
	}
}

// documentTabLabel is the ImGui label for a document's tab. ImGui identifies a tab by its
// label string, so two documents whose display names collide (e.g. a part "box.opd" and a
// drawing "box.odd" both show "box") would hash to the SAME tab id — making ImGui report both
// as selected and the active document ping-pong between them every frame. The "###<uuid>"
// suffix gives each tab a stable, unique id from the document's persisted identity GUID (not
// the visible text, so a rename keeps the tab's identity) while ImGui still displays only the
// name before "###". A reference stub minted without a GUID falls back to the in-memory id.
func documentTabLabel(d *doc.Document) string {
	id := d.File().InternalName()
	if id == "" {
		id = fmt.Sprintf("%d", d.ID())
	}
	return fmt.Sprintf("%s###%s", d.DisplayName(), id)
}
