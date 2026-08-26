// SPDX-License-Identifier: GPL-2.0-only

// Package icon turns the embedded ribbon SVG glyphs into themable RGBA bitmaps the
// head can upload as ImGui textures. It is the ONLY importer of the third-party SVG
// rasterizer (oksvg/rasterx), so the rest of the head depends on this thin seam
// rather than the library (CLAUDE.md: wrap third-party libs behind our own interface).
//
// Glyphs are rasterized into one COVERAGE MASK PER COLOR ROLE (ADR-0033): an asset
// assigns elements to the primary/secondary/tertiary/background roles by painting them
// with sentinel colors, each role renders alone into an alpha mask, and Compose layers
// the masks with the active theme's icon colors into the final image — so the whole
// set recolors with the theme without touching the SVGs.
//
// Every glyph is also size-NORMALIZED: the full glyph's drawn content is cropped to
// its tight bounding box and rescaled to fill a fixed fraction of the output, centred.
// The bounds come from rendering ALL roles together and the identical crop/scale is
// applied to every role's pass, so the color layers stay registered.
package icon

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"

	"oblikovati.org/theme/blenderxml"
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

// RoleMasks is one glyph rasterized as a px×px coverage mask per color role — the
// theme-independent form the icon cache keeps, recoloring it via [RoleMasks.Compose]
// whenever the theme changes.
type RoleMasks struct {
	px    int
	cover [RoleCount][]uint8 // row-major px×px coverage, indexed by Role
}

// Px returns the mask edge length in pixels.
func (m *RoleMasks) Px() int { return m.px }

// RasterizeRoles renders svg into px×px size-normalized coverage masks, one per role.
//
//	masks, err := icon.RasterizeRoles(svgBytes, 32) // 32px large-button glyph
func RasterizeRoles(svg []byte, px int) (*RoleMasks, error) {
	if px <= 0 {
		return nil, fmt.Errorf("icon: rasterize px must be > 0, got %d", px)
	}
	doc, err := parseSVG(svg)
	if err != nil {
		return nil, err
	}
	content, err := fullContentBounds(doc, px)
	if err != nil {
		return nil, err
	}
	masks := &RoleMasks{px: px}
	for r := range RoleCount {
		big, err := renderDoc(filterForRole(doc, r), px*supersample)
		if err != nil {
			return nil, fmt.Errorf("icon: role %s: %w", r, err)
		}
		masks.cover[r] = normalizedAlpha(big, content, px)
	}
	return masks, nil
}

// parseSVG reads an SVG into a generic element tree with namespaces flattened — the
// xmlns URI stays as a literal root attribute, but element names drop their namespace
// so re-marshaling never emits duplicate xmlns declarations.
func parseSVG(svg []byte) (*blenderxml.Node, error) {
	doc, err := blenderxml.Parse(svg)
	if err != nil {
		return nil, fmt.Errorf("icon: parse SVG (%d bytes): %w", len(svg), err)
	}
	stripNamespace(doc)
	return doc, nil
}

// stripNamespace clears the namespace from every element and attribute name in place.
func stripNamespace(n *blenderxml.Node) {
	n.XMLName.Space = ""
	for i := range n.Attrs {
		n.Attrs[i].Name.Space = ""
	}
	for _, c := range n.Children {
		stripNamespace(c)
	}
}

// fullContentBounds renders the document with every role drawn and measures the tight
// content box all role passes share, so layers stay registered after normalization.
func fullContentBounds(doc *blenderxml.Node, px int) (image.Rectangle, error) {
	big, err := renderDoc(doc, px*supersample)
	if err != nil {
		return image.Rectangle{}, fmt.Errorf("icon: bounds pass: %w", err)
	}
	return alphaBounds(big), nil
}

// renderDoc serializes the element tree and rasterizes it into a hi×hi RGBA image.
func renderDoc(doc *blenderxml.Node, hi int) (*image.RGBA, error) {
	data, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	parsed, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	parsed.SetTarget(0, 0, float64(hi), float64(hi))
	img := image.NewRGBA(image.Rect(0, 0, hi, hi))
	scanner := rasterx.NewScannerGV(hi, hi, img, img.Bounds())
	parsed.Draw(rasterx.NewDasher(hi, hi, scanner), 1.0)
	return img, nil
}

// normalizedAlpha crops content out of the supersampled render, rescales it into a
// px×px box (longest side = contentFraction, centred) and returns the alpha channel.
// An empty content box (nothing drawn anywhere in the glyph) yields a blank mask.
func normalizedAlpha(src *image.RGBA, content image.Rectangle, px int) []uint8 {
	out := image.NewRGBA(image.Rect(0, 0, px, px))
	if !content.Empty() {
		xdraw.CatmullRom.Scale(out, normalizedDst(content, px), src, content, xdraw.Src, nil)
	}
	alpha := make([]uint8, px*px)
	for i := range alpha {
		alpha[i] = out.Pix[i*4+3]
	}
	return alpha
}

// normalizedDst is the centred destination rectangle the shared content box scales
// into — computed from the box alone, so every role pass lands identically.
func normalizedDst(content image.Rectangle, px int) image.Rectangle {
	longest := content.Dx()
	if content.Dy() > longest {
		longest = content.Dy()
	}
	scale := contentFraction * float64(px) / float64(longest)
	dw := clampDim(float64(content.Dx()) * scale)
	dh := clampDim(float64(content.Dy()) * scale)
	return image.Rect(0, 0, dw, dh).Add(image.Pt((px-dw)/2, (px-dh)/2))
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
