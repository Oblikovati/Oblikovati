// SPDX-License-Identifier: GPL-2.0-only

// Package icon turns the embedded ribbon SVG glyphs into tintable RGBA bitmaps the
// head can upload as ImGui textures. It is the ONLY importer of the third-party SVG
// rasterizer (oksvg/rasterx), so the rest of the head depends on this thin seam
// rather than the library (CLAUDE.md: wrap third-party libs behind our own interface).
//
// Glyphs are rasterized as a MONOCHROME alpha mask: the coverage the SVG produces
// becomes the alpha channel and RGB is forced to white, so a theme can tint the icon
// at draw time (ImGui's ImageButton tint color) without re-rasterizing it.
package icon

import (
	"bytes"
	"fmt"
	"image"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Rasterize renders svg into a px×px tintable RGBA glyph (RGB=white, A=coverage).
//
//	img, err := icon.Rasterize(svgBytes, 32) // 32px large-button glyph
func Rasterize(svg []byte, px int) (*image.RGBA, error) {
	if px <= 0 {
		return nil, fmt.Errorf("icon: rasterize px must be > 0, got %d", px)
	}
	parsed, err := oksvg.ReadIconStream(bytes.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("icon: parse SVG (%d bytes): %w", len(svg), err)
	}
	parsed.SetTarget(0, 0, float64(px), float64(px))
	img := image.NewRGBA(image.Rect(0, 0, px, px))
	scanner := rasterx.NewScannerGV(px, px, img, img.Bounds())
	parsed.Draw(rasterx.NewDasher(px, px, scanner), 1.0)
	return whiteWithCoverageAlpha(img), nil
}

// whiteWithCoverageAlpha rewrites every pixel to straight-alpha white (RGB=255),
// keeping the coverage the rasterizer wrote into the alpha channel. ImGui blends with
// straight (non-premultiplied) src-alpha and multiplies by the per-vertex tint, so a
// white mask tints cleanly to any theme color: result = tint × (white, coverage).
// Go's image.RGBA is alpha-premultiplied, but for an opaque fill the alpha already
// equals coverage regardless of the SVG's fill color, so only RGB needs forcing.
func whiteWithCoverageAlpha(img *image.RGBA) *image.RGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
		img.Pix[i+1] = 255
		img.Pix[i+2] = 255
	}
	return img
}
