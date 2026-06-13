//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Save-time thumbnail capture (M03-F09, #610): with the save policy set to
// activeWindowOnSave, a successful save arms a capture of the viewport into a
// git-ignored sidecar image next to the document (<document>.png, e.g.
// bracket.opd.png — thumbnails are never document content, ADR-0020/ADR-0034).
// The capture is serviced AFTER the
// viewport has rendered the frame, like the screenshot service, so the image
// reflects what is on screen.

// pendingThumbnailPath is the armed sidecar destination, "" when idle.
var pendingThumbnailPath string

// queueSaveThumbnail arms a capture for the document just saved at savedPath,
// when the save policy asks for one.
func queueSaveThumbnail(s *app.Session, savedPath string) {
	if savedPath == "" {
		return
	}
	if s.SavePolicy().Thumbnail() == types.ThumbnailActiveWindowOnSave {
		pendingThumbnailPath = savedPath + ".png"
	}
}

// serviceSaveThumbnail writes the armed sidecar after the viewport rendered.
// Success is silent (a sidecar, not a user action); failure lands in the
// status bar.
func serviceSaveThumbnail(win *native.Window, s *app.Session) {
	if pendingThumbnailPath == "" {
		return
	}
	path := pendingThumbnailPath
	pendingThumbnailPath = ""
	if err := win.SaveViewportPNG(path); err != nil {
		fileNotice(s, "Thumbnail capture failed: %v", err)
	}
}
