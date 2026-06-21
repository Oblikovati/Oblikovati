//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/head/icon"
	"oblikovati.org/head/internal/native"
)

// Rasterization sizes per button style. Small icons sit beside their label in a panel
// column's stacked rows (and alone in compact grids); large icons head a panel over a
// caption. Glyphs are size-normalized at raster time (see icon.RasterizeRoles), so
// these are the on-screen icon sizes directly. Small dropped from 30 when small
// buttons gained side labels: three 18px rows + labels match a large captioned button,
// the classic two ribbon button sizes.
const (
	smallIconPx = 18
	largeIconPx = 40
)

// retireAfterFrames is how many frames a replaced texture outlives its replacement
// before it is destroyed — it may still be referenced by a swapchain frame in flight,
// so freeing it immediately would be a GPU use-after-free.
const retireAfterFrames = 3

// icons is the chrome's icon texture cache, lazily created from the window on the first
// DrawChrome (package-global to match the chrome's other per-window state).
var icons *iconCache

// iconCache turns embedded SVG glyphs into themed ImGui textures in two stages: each
// glyph is rasterized ONCE into theme-independent per-role coverage masks, and the
// masks are composed with the active theme's icon colors (iconColors, theme_apply.go)
// into the uploaded texture. A theme change only re-composes and re-uploads — lazily,
// per icon actually drawn — which is what makes the appearance editor's live preview
// affordable. Replaced textures are retired for a few frames before destruction.
type iconCache struct {
	win         *native.Window
	masks       map[iconKey]*icon.RoleMasks // nil entry = known-bad asset, never retried
	tex         map[iconKey]uint64
	texRevision uint64 // theme revision the cached textures were composed with
	frame       uint64
	retired     []retiredTexture
}

type iconKey struct {
	name string // bundled asset key, or "" when inline SVG is supplied
	svg  string // add-in-supplied inline SVG markup; takes precedence over name
	px   int
}

type retiredTexture struct {
	tex   uint64
	frame uint64 // frame the texture was retired on
}

func newIconCache(win *native.Window) *iconCache {
	return &iconCache{win: win, masks: map[iconKey]*icon.RoleMasks{}, tex: map[iconKey]uint64{}}
}

// beginFrame advances the cache's frame clock, frees safely-retired textures, and —
// when the theme changed — retires every composed texture so the next texture() call
// re-composes it with the new colors. Called once per frame from prepareChromeFrame.
func (c *iconCache) beginFrame(themeRevision uint64) {
	c.frame++
	c.destroyExpired()
	if themeRevision == c.texRevision {
		return
	}
	c.texRevision = themeRevision
	for _, t := range c.tex {
		c.retire(t)
	}
	clear(c.tex)
}

// texture returns the ImGui texture handle for an icon at the given pixel size,
// composing and uploading it on first use after a theme change. The glyph comes from the
// inline SVG when an add-in supplied one, otherwise from the bundled asset key. It returns
// (0,false) when there is no asset or rasterization/upload fails, so the caller falls back to
// text. A failed lookup is cached too, so a bad glyph is not retried per frame.
func (c *iconCache) texture(name, svg string, px int) (uint64, bool) {
	if (name == "" && svg == "") || px <= 0 {
		return 0, false
	}
	k := iconKey{name: name, svg: svg, px: px}
	if t, ok := c.tex[k]; ok {
		return t, t != 0
	}
	t := c.compose(k)
	c.tex[k] = t
	return t, t != 0
}

// compose builds the themed texture for one key: cached role masks (rasterizing on
// first need) colored with the current icon theme colors, returning 0 on any failure.
func (c *iconCache) compose(k iconKey) uint64 {
	masks, ok := c.masks[k]
	if !ok {
		masks = rasterizeRoles(k)
		c.masks[k] = masks
	}
	if masks == nil {
		return 0
	}
	img := masks.Compose(iconColors)
	return c.win.CreateTexture(img.Pix, k.px, k.px)
}

// rasterizeRoles renders one glyph's role masks, or nil when the asset is missing or
// invalid (the nil is cached so the failure costs once, not every frame). An add-in's inline
// SVG is used directly; otherwise the bundled asset is looked up by key.
func rasterizeRoles(k iconKey) *icon.RoleMasks {
	svg := []byte(k.svg)
	if k.svg == "" {
		bundled, ok := icon.SVG(k.name)
		if !ok {
			return nil
		}
		svg = bundled
	}
	masks, err := icon.RasterizeRoles(svg, k.px)
	if err != nil {
		return nil
	}
	return masks
}

// retire queues a texture for destruction once no frame in flight can reference it.
func (c *iconCache) retire(tex uint64) {
	if tex == 0 {
		return
	}
	c.retired = append(c.retired, retiredTexture{tex: tex, frame: c.frame})
}

// destroyExpired frees retired textures older than the in-flight window.
func (c *iconCache) destroyExpired() {
	kept := c.retired[:0]
	for _, r := range c.retired {
		if c.frame-r.frame > retireAfterFrames {
			c.win.DestroyTexture(r.tex)
			continue
		}
		kept = append(kept, r)
	}
	c.retired = kept
}
