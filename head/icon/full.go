// SPDX-License-Identifier: GPL-2.0-only

package icon

import (
	"fmt"
	"image"

	xdraw "golang.org/x/image/draw"
)

// RenderFull rasterizes an SVG into a px×px RGBA image in its OWN colors, on a
// transparent background — unlike [RasterizeRoles], which renders theme-recolorable
// coverage masks for ribbon glyphs. It is the seam the application icon uses so the
// app/window/installer glyph comes from the same source SVG and the same vetted
// rasterizer (this package is the only importer of oksvg).
//
//	rgba, err := icon.RenderFull(appSVG, 256) // a 256px window/app icon
func RenderFull(svg []byte, px int) (*image.RGBA, error) {
	if px <= 0 {
		return nil, fmt.Errorf("icon: render px must be > 0, got %d", px)
	}
	doc, err := parseSVG(svg)
	if err != nil {
		return nil, err
	}
	// Supersample then high-quality downscale so thin strokes stay crisp at small icon
	// sizes (the app mark is a fine wireframe that would alias if drawn straight at 16px).
	big, err := renderDoc(doc, px*supersample)
	if err != nil {
		return nil, fmt.Errorf("icon: render full at %dpx: %w", px, err)
	}
	if supersample == 1 {
		return big, nil
	}
	out := image.NewRGBA(image.Rect(0, 0, px, px))
	xdraw.CatmullRom.Scale(out, out.Bounds(), big, big.Bounds(), xdraw.Src, nil)
	return out, nil
}
