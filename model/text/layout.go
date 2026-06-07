// SPDX-License-Identifier: GPL-2.0-only

package text

import (
	"oblikovati/api/types"
	"oblikovati/math"
)

// Request is one piece of sketch text to lay out: its content, em Height (model units),
// optional Family (resolved via a FontResolver), and how it is aligned about its anchor.
// FontSize, when > 0, scales the glyph em instead of Height (matching Inventor's separate
// TextStyle.FontSize); 0 tracks Height.
type Request struct {
	Content  string
	Family   string
	Height   float64
	FontSize float64
	HAlign   types.TextHorizontalAlign
	VAlign   types.TextVerticalAlign
}

// emHeight is the height the glyph em is scaled to: FontSize when set, else Height.
func (r Request) emHeight() float64 {
	if r.FontSize > 0 {
		return r.FontSize
	}
	return r.Height
}

// AlignedOutlines lays the request's text out and returns its glyph contours (closed
// polylines, baseline at y=0, left edge at x=0 before alignment), shifted so the text is
// aligned about the ORIGIN per HAlign/VAlign. A caller then translates the contours by the
// entity's anchor. Each contour is one glyph loop (a counter is its own nested contour, so
// the sketch profile detector nests it as a hole). This is the single text->geometry path
// the sketch text entity and the emboss recompute share, so geometry is never baked.
func AlignedOutlines(r Request, fonts FontResolver) ([][]math.Point2, error) {
	ft, err := fonts.Resolve(r.Family)
	if err != nil {
		return nil, err
	}
	em := r.emHeight()
	contours, err := ft.Outlines(r.Content, em)
	if err != nil {
		return nil, err
	}
	dx, err := horizontalShift(ft, r, em)
	if err != nil {
		return nil, err
	}
	dy := verticalShift(r, contours)
	return shiftContours(contours, dx, dy), nil
}

// horizontalShift returns the x offset that aligns the text about the origin per HAlign.
func horizontalShift(ft *Font, r Request, em float64) (float64, error) {
	if r.HAlign == types.TextAlignLeft || r.HAlign == "" {
		return 0, nil
	}
	w, err := ft.Advance(r.Content, em)
	if err != nil {
		return 0, err
	}
	if r.HAlign == types.TextAlignRight {
		return -w, nil
	}
	return -w / 2, nil // center
}

// verticalShift returns the y offset that aligns the text about the origin per VAlign,
// measured from the text's actual rendered bounds (so middle visually centres the glyphs,
// not the font's ascent/descent box which carries line-gap padding). Baseline (the
// sketch-text default) leaves the baseline on the anchor.
func verticalShift(r Request, contours [][]math.Point2) float64 {
	if r.VAlign == types.TextAlignBaseline || r.VAlign == "" {
		return 0
	}
	minY, maxY := contourYBounds(contours)
	switch r.VAlign {
	case types.TextAlignUpper: // top of text on the anchor
		return -maxY
	case types.TextAlignLower: // bottom of text on the anchor
		return -minY
	default: // middle: text bounds centred on the anchor
		return -(minY + maxY) / 2
	}
}

// contourYBounds returns the min/max y over all contour points (0,0 when empty).
func contourYBounds(contours [][]math.Point2) (minY, maxY float64) {
	first := true
	for _, c := range contours {
		for _, p := range c {
			y := float64(p.Y)
			if first {
				minY, maxY, first = y, y, false
				continue
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return
}

// shiftContours returns the contours translated by (dx, dy).
func shiftContours(contours [][]math.Point2, dx, dy float64) [][]math.Point2 {
	out := make([][]math.Point2, len(contours))
	for i, c := range contours {
		moved := make([]math.Point2, len(c))
		for j, p := range c {
			moved[j] = math.P2(p.X+math.Scalar(dx), p.Y+math.Scalar(dy))
		}
		out[i] = moved
	}
	return out
}
