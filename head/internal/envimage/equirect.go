// SPDX-License-Identifier: GPL-2.0-only

// Package envimage resolves a renderer.Environment into an equirectangular HDR image the
// viewport uploads for image-based lighting (ADR-0026 §3,§4). It is pure Go with no cgo or
// GPU, so the procedural presets and the .hdr decoder are unit-tested headlessly; the native
// layer only uploads the float pixels and samples them in the Realistic shader.
package envimage

import (
	"fmt"

	"oblikovati/renderer"
)

// Equirect is an equirectangular HDR image in linear RGB, row-major, interleaved RGBA float32
// (alpha = 1), ready to upload as an R16G16B16A16_SFLOAT texture. Row 0 is the zenith (+Z up),
// the last row the nadir; columns sweep azimuth 0..2π. Width is 2×Height (the lat-long ratio).
// HDR values may exceed 1 (e.g. a sun disk).
type Equirect struct {
	W, H   int
	Pixels []float32
}

// At returns the RGB at pixel (x,y); for tests and convolution.
func (e Equirect) At(x, y int) (r, g, b float32) {
	i := (y*e.W + x) * 4
	return e.Pixels[i], e.Pixels[i+1], e.Pixels[i+2]
}

// set writes the RGB at pixel (x,y) (alpha forced to 1).
func (e Equirect) set(x, y int, r, g, b float32) {
	i := (y*e.W + x) * 4
	e.Pixels[i], e.Pixels[i+1], e.Pixels[i+2], e.Pixels[i+3] = r, g, b, 1
}

// newEquirect allocates a w×h equirect with all pixels opaque black.
func newEquirect(w, h int) Equirect {
	return Equirect{W: w, H: h, Pixels: make([]float32, w*h*4)}
}

// Resolve turns a renderer.Environment into its HDR image: a decoded file when FilePath is set,
// otherwise the built-in preset. It returns ok=false for an inactive environment (EnvNone with
// no file), in which case the caller skips IBL and keeps the analytic ambient.
//
//	img, ok, err := envimage.Resolve(session.Environment())
//	if ok { win.SetViewportEnvironment(img, ...) }
func Resolve(env renderer.Environment) (Equirect, bool, error) {
	if env.FilePath != "" {
		img, err := DecodeHDR(env.FilePath)
		if err != nil {
			return Equirect{}, false, fmt.Errorf("envimage: load %q: %w", env.FilePath, err)
		}
		return img, true, nil
	}
	if env.Preset == renderer.EnvNone {
		return Equirect{}, false, nil
	}
	return presetEquirect(env.Preset), true, nil
}
