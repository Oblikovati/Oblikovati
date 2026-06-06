//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"log"

	"oblikovati/head/internal/envimage"
	"oblikovati/head/internal/native"
	"oblikovati/head/viewport"
	"oblikovati/renderer"
)

// envCache memoizes the uploaded IBL environment so the (expensive) decode + mip + GPU upload
// happens only when the chosen environment image changes, not every frame. The pixel key
// (preset + file) gates re-decoding; rotation/intensity ride along on each upload (they change
// only on user action, never per frame).
type envCache struct {
	pixelKey string
	paramKey string
	upload   envimage.Upload
	active   bool
}

// viewportEnv is the single viewport's environment cache (the head has one 3D viewport).
var viewportEnv envCache

// applyEnvironment resolves the session's environment and uploads it to the viewport when it
// has changed since the last frame, so metallic/smooth bodies reflect the HDR sky (ADR-0026
// §3,§4). An inactive environment disables IBL (the analytic ambient resumes).
func applyEnvironment(win *native.Window, env renderer.Environment) {
	pixelKey := fmt.Sprintf("%d|%s", env.Preset, env.FilePath)
	paramKey := fmt.Sprintf("%g|%g", env.Rotation, env.Intensity)
	if pixelKey == viewportEnv.pixelKey && paramKey == viewportEnv.paramKey {
		return
	}
	if pixelKey != viewportEnv.pixelKey {
		viewportEnv.resolve(env)
		viewportEnv.pixelKey = pixelKey
	}
	viewportEnv.paramKey = paramKey
	if !viewportEnv.active {
		win.SetViewportEnvironment(nil, nil, 0, 0)
		return
	}
	win.SetViewportEnvironment(viewportEnv.upload.Data, viewportEnv.upload.Dims,
		env.Rotation, env.Intensity)
}

// applySkybox enables the HDR sky background when the active environment has ShowImage set,
// passing the inverse view-projection for view-ray reconstruction; otherwise the themed clear
// color shows through (ADR-0026 §5).
func applySkybox(win *native.Window, env renderer.Environment, mvp [16]float32) {
	if !env.ShowImage || !env.IsActive() {
		win.SetViewportSkybox(nil, false)
		return
	}
	inv, ok := viewport.Invert4x4(mvp)
	if !ok {
		win.SetViewportSkybox(nil, false)
		return
	}
	win.SetViewportSkybox(inv[:], true)
}

// resolve decodes/generates the environment image and builds its mip chain, caching the upload;
// a load failure (e.g. a missing .hdr) disables IBL and logs, rather than crashing the frame.
func (e *envCache) resolve(env renderer.Environment) {
	img, ok, err := envimage.Resolve(env)
	if err != nil {
		log.Printf("viewport: environment %q: %v", env.FilePath, err)
	}
	if !ok || err != nil {
		e.upload = envimage.Upload{}
		e.active = false
		return
	}
	e.upload = envimage.Flatten(envimage.MipChain(img))
	e.active = true
}
