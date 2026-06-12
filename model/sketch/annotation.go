// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// FillRegion is a hatched/filled closed region of the sketch, identified by a seed point
// inside the region (the click point) plus a fill-style name. It is annotative — it
// carries no constrainable points and is not itself a profile (it references one).
type FillRegion struct {
	entityBase
	Seed  math.Point2 // a point inside the region to fill
	Style string      // fill/hatch style name ("" ⇒ solid)
}

// FillRegions creates and tracks fill regions.
type FillRegions struct {
	s     *Sketch
	items []*FillRegion
}

// Add fills the region containing seed with the given style.
func (c *FillRegions) Add(seed math.Point2, style string) *FillRegion {
	f := &FillRegion{entityBase: newEntity(), Seed: seed, Style: style}
	c.s.add(f)
	c.items = append(c.items, f)
	return f
}

// Count returns the number of fill regions; Item returns the i-th.
func (c *FillRegions) Count() int             { return len(c.items) }
func (c *FillRegions) Item(i int) *FillRegion { return c.items[i] }

// Region returns the profile the fill region's seed point falls inside, or nil if the
// seed is not enclosed (the fill is then dangling — Inventor lets this be a warning).
func (f *FillRegion) Region(s *Sketch) *Profile {
	for _, p := range s.Profiles().All() {
		if p.Contains(f.Seed) {
			return p
		}
	}
	return nil
}

// TextHJustify is a text box's horizontal justification. It mirrors
// [oblikovati.org/api/types.TextHorizontalAlign] but stays an int enum here so existing
// call sites and the document codec are unchanged.
type TextHJustify uint8

const (
	TextLeft TextHJustify = iota
	TextCenter
	TextRight
)

// TextVJustify is a text box's vertical justification (mirrors
// [oblikovati.org/api/types.TextVerticalAlign]). Baseline is the sketch-text default.
type TextVJustify uint8

const (
	TextBaseline TextVJustify = iota
	TextLower
	TextMiddle
	TextUpper
)

// TextBox is sketch text: a string anchored at a point, with a character Height (cm),
// Rotation (radians CCW about the anchor), horizontal + vertical justification, and a font
// (Family name + FontSize cm; FontSize 0 tracks Height). It is NOT baked geometry — it
// keeps its content/font and DERIVES its glyph outlines on demand (Outlines/Region), so
// editing it re-derives the geometry and an emboss that references it recomputes. This is
// the by-reference text model (Inventor: TextBox.ConvertToGeometry, edited via the sketch).
type TextBox struct {
	entityBase
	Anchor       math.Point2
	Text         string
	Height       math.Scalar
	Rotation     math.Scalar
	Justify      TextHJustify
	VJustify     TextVJustify
	Family       string
	FontResource string // document font resource UUID (ADR-0031); when set, overrides Family
	FontSize     math.Scalar
}

// FontRef is the string the font resolver resolves the text's face by: the document font
// resource UUID when one was chosen (an embedded OS font or an app-provided face), else the
// family name (legacy/default). See compdef's text.FontResolver.
func (t *TextBox) FontRef() string {
	if t.FontResource != "" {
		return t.FontResource
	}
	return t.Family
}

// TextBoxes creates and tracks sketch text.
type TextBoxes struct {
	s     *Sketch
	items []*TextBox
}

// Add places text at anchor with the given content, height, rotation, and horizontal
// justification (left-justified vertical baseline, default font). Use AddStyled for the
// full field set.
func (c *TextBoxes) Add(anchor math.Point2, text string, height, rotation math.Scalar, justify TextHJustify) *TextBox {
	return c.AddStyled(anchor, text, height, rotation, justify, TextBaseline, "", 0)
}

// AddStyled places text with the full type-setting field set.
func (c *TextBoxes) AddStyled(anchor math.Point2, text string, height, rotation math.Scalar, hjust TextHJustify, vjust TextVJustify, family string, fontSize math.Scalar) *TextBox {
	t := &TextBox{
		entityBase: newEntity(), Anchor: anchor, Text: text, Height: height,
		Rotation: rotation, Justify: hjust, VJustify: vjust, Family: family, FontSize: fontSize,
	}
	c.s.add(t)
	c.items = append(c.items, t)
	// Every text box is tied down by its non-deletable anchor record
	// (M06-F11, #626) so constraint enumeration explains the anchoring.
	c.s.anchorTextBox(t)
	return t
}

func (c *TextBoxes) remove(t *TextBox) { c.items = removeItem(c.items, t) }

// Count returns the number of text boxes; Item returns the i-th.
func (c *TextBoxes) Count() int          { return len(c.items) }
func (c *TextBoxes) Item(i int) *TextBox { return c.items[i] }
