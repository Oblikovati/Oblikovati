// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati/api/types"
	"oblikovati/math"
	"oblikovati/model/text"
)

// Outlines derives the text box's glyph contours (closed polylines in sketch-plane
// coordinates: aligned about the anchor per the box's justification, rotated by Rotation,
// then translated to Anchor). It resolves the font via fonts, so the geometry is computed
// on demand and never baked into the sketch. Returns nil for empty text.
//
// Example: tb.Outlines(text.DefaultResolver()) for the glyph loops of an embossable string.
func (t *TextBox) Outlines(fonts text.FontResolver) ([][]math.Point2, error) {
	if t.Text == "" {
		return nil, nil
	}
	req := text.Request{
		Content: t.Text, Family: t.Family,
		Height: float64(t.Height), FontSize: float64(t.FontSize),
		HAlign: hAlignOf(t.Justify), VAlign: vAlignOf(t.VJustify),
	}
	contours, err := text.AlignedOutlines(req, fonts)
	if err != nil {
		return nil, err
	}
	return placeContours(contours, float64(t.Rotation), t.Anchor), nil
}

// TextProfiles derives the closed profiles of the text box's glyphs (outer letter
// boundaries with their counters as holes), ready to extrude/emboss. It reuses the same
// even–odd nesting the sketch profile detector uses, so a counter (the hole in A/O/B) nests
// as a profile hole. This is how an emboss reads a referenced text entity's geometry
// WITHOUT any baked sketch lines.
func (t *TextBox) TextProfiles(fonts text.FontResolver) ([]*Profile, error) {
	contours, err := t.Outlines(fonts)
	if err != nil {
		return nil, err
	}
	loops := make([]Loop, 0, len(contours))
	for _, c := range contours {
		if len(c) >= 3 {
			loops = append(loops, Loop{polygon: c, closed: true})
		}
	}
	return groupRegions(loops), nil
}

// placeContours rotates each contour by angle (radians, CCW) about the origin and
// translates it to anchor — the rigid placement applied after alignment.
func placeContours(contours [][]math.Point2, angle float64, anchor math.Point2) [][]math.Point2 {
	cos, sin := stdmath.Cos(angle), stdmath.Sin(angle)
	out := make([][]math.Point2, len(contours))
	for i, c := range contours {
		moved := make([]math.Point2, len(c))
		for j, p := range c {
			x, y := float64(p.X), float64(p.Y)
			moved[j] = math.P2(anchor.X+math.Scalar(x*cos-y*sin), anchor.Y+math.Scalar(x*sin+y*cos))
		}
		out[i] = moved
	}
	return out
}

// hAlignOf / vAlignOf map the model's int enums to the api/types string alignment used by
// the text layout package.
func hAlignOf(j TextHJustify) types.TextHorizontalAlign {
	switch j {
	case TextCenter:
		return types.TextAlignCenter
	case TextRight:
		return types.TextAlignRight
	default:
		return types.TextAlignLeft
	}
}

func vAlignOf(j TextVJustify) types.TextVerticalAlign {
	switch j {
	case TextLower:
		return types.TextAlignLower
	case TextMiddle:
		return types.TextAlignMiddle
	case TextUpper:
		return types.TextAlignUpper
	default:
		return types.TextAlignBaseline
	}
}
