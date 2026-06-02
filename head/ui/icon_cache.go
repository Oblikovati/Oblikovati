//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/head/icon"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// iconTint is the color every ribbon glyph is drawn with. The icons are rasterized as
// white alpha masks, so this single tint recolors all of them — the hook a future
// theme manager sets. Default: a light gray that reads on the dark chrome.
var iconTint = [4]float32{0.85, 0.87, 0.90, 1}

// Rasterization sizes per button style. Small icons fill the dense sketch tool grids;
// large icons head a panel (Inventor's two button sizes).
const (
	smallIconPx = 18
	largeIconPx = 32
)

// icons is the chrome's icon texture cache, lazily created from the window on the first
// DrawChrome (package-global to match the chrome's other per-window state).
var icons *iconCache

// iconCache rasterizes embedded SVG glyphs and uploads them as ImGui textures, caching
// by (key, px) so each glyph is rasterized and uploaded exactly once for the window's
// lifetime. The native side frees the GPU textures on window teardown.
type iconCache struct {
	win *native.Window
	tex map[iconKey]uint64
}

type iconKey struct {
	name string
	px   int
}

func newIconCache(win *native.Window) *iconCache {
	return &iconCache{win: win, tex: map[iconKey]uint64{}}
}

// texture returns the ImGui texture handle for an icon at the given pixel size,
// rasterizing and uploading it on first use. It returns (0,false) when the key has no
// bundled asset or rasterization/upload fails, so the caller falls back to text. A
// failed lookup is cached (as 0) too, so a missing glyph is not retried every frame.
func (c *iconCache) texture(name string, px int) (uint64, bool) {
	if name == "" || px <= 0 {
		return 0, false
	}
	k := iconKey{name, px}
	if t, ok := c.tex[k]; ok {
		return t, t != 0
	}
	t := c.upload(name, px)
	c.tex[k] = t
	return t, t != 0
}

// upload rasterizes the named glyph to px×px and hands the RGBA to the GPU, returning 0
// if the asset is missing or either step fails.
func (c *iconCache) upload(name string, px int) uint64 {
	svg, ok := icon.SVG(name)
	if !ok {
		return 0
	}
	img, err := icon.Rasterize(svg, px)
	if err != nil {
		return 0
	}
	return c.win.CreateTexture(img.Pix, px, px)
}
