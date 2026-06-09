// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"path/filepath"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// defaultCapturePath is where viewport.capture writes when no Path is given — distinct from the
// Tools ▸ Save Viewport PNG path so a programmatic capture never clobbers a manual one.
var defaultCapturePath = filepath.Join("/tmp", "oblikovati-capture.png")

// captureViewport requests a PNG of the active document's viewport framebuffer
// (wire.MethodViewportCapture). It only FLAGS the request — the head writes the file after the next
// frame renders, so the image is exactly what is on screen — and returns the path it will write plus
// the current viewport pixel size, so the caller can poll the path (or its mtime) for completion.
func captureViewport(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CaptureViewportArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	path := in.Path
	if path == "" {
		path = defaultCapturePath
	}
	s.RequestViewportCapture(path)
	cam := s.Camera()
	return json.Marshal(wire.CaptureViewportResult{Path: path, Width: cam.Width, Height: cam.Height})
}

// setNormalDebug turns the viewport's normal-debug render on/off (wire.MethodViewportSetNormalDebug);
// the head applies it on the next frame.
func setNormalDebug(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetNormalDebugArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	s.SetNormalDebug(in.On)
	return json.Marshal(wire.NormalDebugResult(in))
}

// setMeshColors turns the mesh-debug-colors render on/off (wire.MethodViewportSetMeshColors); the
// head's draw-list cache rebuilds with per-face/per-triangle colors on the next frame.
func setMeshColors(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetMeshColorsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	s.SetMeshColors(in.On, in.PerTriangle)
	return json.Marshal(wire.MeshColorsResult(in))
}
