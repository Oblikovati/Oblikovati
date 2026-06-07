// SPDX-License-Identifier: GPL-2.0-only

// Package icon turns the embedded ribbon SVG glyphs into tintable RGBA bitmaps the
// head can upload as ImGui textures. It is the ONLY importer of the third-party SVG
// rasterizer (oksvg/rasterx), so the rest of the head depends on this thin seam
// rather than the library (CLAUDE.md: wrap third-party libs behind our own interface).
//
// Glyphs are rasterized as a MONOCHROME alpha mask: the coverage the SVG produces
// becomes the alpha channel and RGB is forced to white, so a theme can tint the icon
// at draw time (ImGui's ImageButton tint color) without re-rasterizing it.
//
// Every glyph is also size-NORMALIZED: its drawn content is cropped to its tight
// bounding box and rescaled to fill a fixed fraction of the output, centred. The hand-
// authored SVGs carry inconsistent internal margins, so without this a glyph that
// happens to sit in a small part of its 24x24 viewBox would render much smaller than one
// that fills it. Normalizing makes every ribbon icon the same visual size regardless of
// how its source art is laid out.
package icon

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
)

const (
	// contentFraction is how much of the output square the glyph's longest side fills
	// (the rest is an even margin). Tuned so the densest icons fill the button while a
	// hair of breathing room remains.
	contentFraction = 0.92
	// supersample renders at N× the target before the normalize downscale, so the
	// crop-and-rescale stays crisp.
	supersample = 4
	// alphaThreshold ignores near-transparent anti-aliasing fringe when measuring the
	// content bounding box, so a faint edge pixel doesn't inflate the glyph bounds.
	alphaThreshold = 16
)

// Rasterize renders svg into a px×px tintable, size-normalized RGBA glyph (RGB=white,
// A=coverage).
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
	hi := px * supersample
	parsed.SetTarget(0, 0, float64(hi), float64(hi))
	big := image.NewRGBA(image.Rect(0, 0, hi, hi))
	scanner := rasterx.NewScannerGV(hi, hi, big, big.Bounds())
	parsed.Draw(rasterx.NewDasher(hi, hi, scanner), 1.0)
	return whiteWithCoverageAlpha(normalizeContent(big, px)), nil
}

// normalizeContent crops src to its drawn content and rescales that into a px×px image so
// the glyph's longest side spans contentFraction of the output, centred. An empty (blank)
// source yields a blank output.
func normalizeContent(src *image.RGBA, px int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, px, px))
	content := alphaBounds(src)
	if content.Empty() {
		return out
	}
	longest := content.Dx()
	if content.Dy() > longest {
		longest = content.Dy()
	}
	scale := contentFraction * float64(px) / float64(longest)
	dw := clampDim(float64(content.Dx()) * scale)
	dh := clampDim(float64(content.Dy()) * scale)
	dst := image.Rect(0, 0, dw, dh).Add(image.Pt((px-dw)/2, (px-dh)/2)) // centred
	xdraw.CatmullRom.Scale(out, dst, src, content, xdraw.Src, nil)
	return out
}

// clampDim rounds a scaled dimension to at least one pixel.
func clampDim(v float64) int {
	d := int(math.Round(v))
	if d < 1 {
		return 1
	}
	return d
}

// alphaBounds returns the tight bounding box of pixels whose alpha exceeds the fringe
// threshold, or the empty rectangle when nothing was drawn.
func alphaBounds(img *image.RGBA) image.Rectangle {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		base := img.PixOffset(b.Min.X, y)
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.Pix[base+(x-b.Min.X)*4+3] <= alphaThreshold {
				continue
			}
			found = true
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// whiteWithCoverageAlpha rewrites every pixel to straight-alpha white (RGB=255),
// keeping the coverage in the alpha channel. ImGui blends with straight (non-
// premultiplied) src-alpha and multiplies by the per-vertex tint, so a white mask tints
// cleanly to any theme color: result = tint × (white, coverage). Go's image.RGBA is
// alpha-premultiplied, but for a white fill RGB already equals coverage, so only the
// (cropped/rescaled, possibly non-white) RGB needs forcing.
func whiteWithCoverageAlpha(img *image.RGBA) *image.RGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
		img.Pix[i+1] = 255
		img.Pix[i+2] = 255
	}
	return img
}
