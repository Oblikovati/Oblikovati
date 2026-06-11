// SPDX-License-Identifier: GPL-2.0-only

package envimage

import (
	"embed"
	"sync"
)

// The embedded photographic skymap behind [renderer.EnvSky]: "Qwantani (Pure Sky)" by
// Greg Zaal / Jarod Guest, published CC0 by Poly Haven (see assets/CREDITS.md) — a clear
// midday sky that serves as the default environment of a fresh session. Embedding the 1k
// Radiance file keeps the default skymap dependency- and filesystem-free, like the
// procedural presets (ADR-0026 §4).
//
//go:embed assets/qwantani_puresky_1k.hdr
var skyAssetFS embed.FS

// skyEquirect decodes the embedded skymap once and serves the cached image after (the
// decode is ~1 MB of RGBE; per-frame callers hit the upload cache anyway).
var skyEquirect = sync.OnceValues(func() (Equirect, error) {
	data, err := skyAssetFS.ReadFile("assets/qwantani_puresky_1k.hdr")
	if err != nil {
		return Equirect{}, err
	}
	return DecodeHDRBytes(data)
})

// skyOrStudio resolves [renderer.EnvSky]: the embedded skymap, falling back to the
// procedural studio gradient if the embedded asset ever fails to decode (a build/asset
// error — the viewport stays lit rather than going dark).
func skyOrStudio() Equirect {
	img, err := skyEquirect()
	if err != nil {
		return generate(studioShade)
	}
	return img
}
