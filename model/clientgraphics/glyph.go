// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// defaultPointPixels is the on-screen half-size of a point glyph when the primitive gives
// no PointSize — a marker a few pixels across, like the sketch snap glyphs.
const defaultPointPixels = 6.0

// glyphCircleSegments is the polygon resolution of a "circle" point glyph.
const glyphCircleSegments = 16

// billboard holds the camera-facing right/up axes used to draw screen-aligned glyphs so a
// point marker keeps a constant on-screen size and orientation regardless of view.
type billboard struct {
	right math.Vector3
	up    math.Vector3
}

// newBillboard derives the camera's right/up basis (forward = eye→target).
func newBillboard(cam scene.Camera) billboard {
	forward := cam.Forward()
	right := normalize(forward.Cross(cam.Up))
	return billboard{right: right, up: normalize(right.Cross(forward))}
}

// pointGlyphs expands a points primitive into glyph line segments (pairs of endpoints) at
// the primitive's screen-constant size, using the camera billboard so every marker faces
// the viewer. PointSize, when set, is the glyph's on-screen half-size in pixels.
func pointGlyphs(p Primitive, bb billboard, worldPerPixel float64) [][2]math.Point3 {
	pixels := p.PointSize
	if pixels <= 0 {
		pixels = defaultPointPixels
	}
	r := pixels * worldPerPixel
	style := p.PointStyle
	if style == "" {
		style = types.GraphicsPointPlus
	}
	var segs [][2]math.Point3
	for _, c := range p.Coords {
		segs = append(segs, glyphSegments(style, c, bb, r)...)
	}
	return segs
}

// glyphSegments returns the line segments of one glyph centered at c.
func glyphSegments(style types.GraphicsPointStyle, c math.Point3, bb billboard, r float64) [][2]math.Point3 {
	switch style {
	case types.GraphicsPointCross:
		return diagonalGlyph(c, bb, r)
	case types.GraphicsPointSquare, types.GraphicsPointDot:
		return polygonGlyph(c, bb, r, 4, 0.25)
	case types.GraphicsPointCircle:
		return polygonGlyph(c, bb, r, glyphCircleSegments, 0)
	default: // plus
		return plusGlyph(c, bb, r)
	}
}

// plusGlyph builds an axis-aligned "+" in the billboard plane.
func plusGlyph(c math.Point3, bb billboard, r float64) [][2]math.Point3 {
	return [][2]math.Point3{
		{translate(c, bb.right, -r), translate(c, bb.right, r)},
		{translate(c, bb.up, -r), translate(c, bb.up, r)},
	}
}

// diagonalGlyph builds an "X" in the billboard plane.
func diagonalGlyph(c math.Point3, bb billboard, r float64) [][2]math.Point3 {
	d1 := bb.right.Add(bb.up)
	d2 := bb.right.Sub(bb.up)
	return [][2]math.Point3{
		{c.TranslateBy(d1.Scale(-r)), c.TranslateBy(d1.Scale(r))},
		{c.TranslateBy(d2.Scale(-r)), c.TranslateBy(d2.Scale(r))},
	}
}

// polygonGlyph builds a closed n-gon of radius r in the billboard plane, rotated by
// phaseTurns turns (e.g. 0.25 turns the 4-gon into an axis-aligned square).
func polygonGlyph(c math.Point3, bb billboard, r float64, n int, phaseTurns float64) [][2]math.Point3 {
	pts := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		t := 2 * stdmath.Pi * (float64(i)/float64(n) + phaseTurns)
		off := bb.right.Scale(r * stdmath.Cos(t)).Add(bb.up.Scale(r * stdmath.Sin(t)))
		pts[i] = c.TranslateBy(off)
	}
	segs := make([][2]math.Point3, n)
	for i := 0; i < n; i++ {
		segs[i] = [2]math.Point3{pts[i], pts[(i+1)%n]}
	}
	return segs
}

// translate moves c by dir scaled by d.
func translate(c math.Point3, dir math.Vector3, d float64) math.Point3 {
	return c.TranslateBy(dir.Scale(d))
}

// normalize returns v as a unit vector, or the zero vector if v has no length.
func normalize(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}
