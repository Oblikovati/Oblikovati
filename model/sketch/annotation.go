// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

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

// TextHJustify is a text box's horizontal justification.
type TextHJustify uint8

const (
	TextLeft TextHJustify = iota
	TextCenter
	TextRight
)

// TextBox is sketch text: a string anchored at a point, with a height (cm), rotation
// (radians CCW about the anchor), and horizontal justification. Annotative.
type TextBox struct {
	entityBase
	Anchor   math.Point2
	Text     string
	Height   math.Scalar
	Rotation math.Scalar
	Justify  TextHJustify
}

// TextBoxes creates and tracks sketch text.
type TextBoxes struct {
	s     *Sketch
	items []*TextBox
}

// Add places text at anchor with the given content, height, rotation, and justification.
func (c *TextBoxes) Add(anchor math.Point2, text string, height, rotation math.Scalar, justify TextHJustify) *TextBox {
	t := &TextBox{entityBase: newEntity(), Anchor: anchor, Text: text, Height: height, Rotation: rotation, Justify: justify}
	c.s.add(t)
	c.items = append(c.items, t)
	return t
}

// Count returns the number of text boxes; Item returns the i-th.
func (c *TextBoxes) Count() int          { return len(c.items) }
func (c *TextBoxes) Item(i int) *TextBox { return c.items[i] }
