// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

// SketchImage is a raster image placed on the sketch plane (the underlay used for
// tracing): a reference to the image data, an anchor point, a size, a rotation, and an
// opacity. It is annotative — it carries no constrainable points and is not part of any
// profile.
type SketchImage struct {
	entityBase
	Ref      string // package-store reference / path to the image bytes
	Anchor   math.Point2
	Width    math.Scalar
	Height   math.Scalar
	Rotation math.Scalar // radians, CCW about the anchor
	Opacity  float64     // 0 (transparent) … 1 (opaque)
}

// SketchImages creates and tracks sketch images.
type SketchImages struct {
	s     *Sketch
	items []*SketchImage
}

// Add places an image on the sketch plane.
func (c *SketchImages) Add(ref string, anchor math.Point2, width, height, rotation math.Scalar, opacity float64) *SketchImage {
	img := &SketchImage{
		entityBase: newEntity(),
		Ref:        ref,
		Anchor:     anchor,
		Width:      width,
		Height:     height,
		Rotation:   rotation,
		Opacity:    opacity,
	}
	c.s.add(img)
	c.items = append(c.items, img)
	return img
}

// Count returns the number of images; Item returns the i-th.
func (c *SketchImages) Count() int              { return len(c.items) }
func (c *SketchImages) Item(i int) *SketchImage { return c.items[i] }
