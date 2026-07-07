// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Retained GPU point-cloud upload (#645 perf). Scan points are drawn by a native GL-points pipeline
// from a buffer that lives in VRAM across frames, not the per-frame line-cross markers that used to
// ride the overlay atlas (and defeated the whole-model geometry-upload cache). uploadPointClouds
// pushes the active part's visible points to that buffer, but only rebuilds the interleaved vertex
// data + re-uploads when the content key changes — so orbiting a loaded scan does zero CPU work here
// and zero PCIe transfer (the native side redraws the resident buffer every RenderViewport).
//
// The gate is package-level to match the single render-loop assumption the rest of this package
// already relies on (frameAtlasCache, visibleScratch): there is one native viewport, and its tiled
// slots share the one retained point buffer, so one key tracks what is resident.
var (
	pointUploadWindow *native.Window
	pointUploadKey    uint64
	pointUploadValid  bool
)

// pointCloudUploadSource is the session surface uploadPointClouds reads (audit I5, the arrowSession
// pattern): the content key that gates the re-upload, the native point size, and the interleaved
// GPU vertices. Taking this ≤6-method seam instead of the whole *app.Session documents that the
// upload only reads point-cloud render state and makes it testable against a small fake source.
type pointCloudUploadSource interface {
	PointCloudDisplayKey() uint64
	PointCloudPointSize() float32
	PointCloudGPUVertices() ([]float32, int)
}

var _ pointCloudUploadSource = (*app.Session)(nil)

// uploadPointClouds refreshes the native point buffer for s's active part. It is a no-op transfer
// while the displayed set and its colors are unchanged, but still updates native point size.
func uploadPointClouds(win *native.Window, s pointCloudUploadSource) {
	if pointUploadWindow != win {
		pointUploadWindow = win
		pointUploadKey, pointUploadValid = 0, false
	}
	key := s.PointCloudDisplayKey()
	sizePx := s.PointCloudPointSize()
	if pointUploadValid && key == pointUploadKey {
		win.UploadPoints(nil, 0, key, sizePx)
		return
	}
	verts, count := s.PointCloudGPUVertices()
	win.UploadPoints(verts, count, key, sizePx)
	pointUploadKey, pointUploadValid = key, true
}
