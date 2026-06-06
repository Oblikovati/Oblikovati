// SPDX-License-Identifier: GPL-2.0-only

// Package text renders TrueType/OpenType glyph outlines into closed 2D polygon contours, so
// the sketch layer can build real text profiles (for emboss/engrave) from a font file. It is
// a thin, project-owned facade over golang.org/x/image/font/sfnt — the only place the
// application depends on a font library (wrap third-party libs behind our own interface).
package text

import (
	"fmt"

	"oblikovati/math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// curveFlattenSteps is how many line segments each quadratic/cubic glyph curve is flattened
// into — enough that text profiles read smooth at typical emboss sizes.
const curveFlattenSteps = 8

// Font is a parsed TrueType/OpenType font: a thin, project-owned facade over sfnt.
type Font struct {
	f    *sfnt.Font
	upem float64
}

// Parse reads a font from its raw bytes (a .ttf/.otf file's contents).
func Parse(data []byte) (*Font, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("text: parse font: %w", err)
	}
	return &Font{f: f, upem: float64(f.UnitsPerEm())}, nil
}

// Outlines returns the closed polygon contours of the text laid out along the baseline
// (y up, first glyph starting at x=0), scaled so one em is `height` model units. Each contour
// is one glyph loop — a letter's outer boundary and its counters (the hole in A/O/B) come
// back as separate contours, which the sketch profile detector nests into a region with
// holes. Curves are flattened to polylines. Contours omit the closing duplicate point (a
// closed polyline rejoins its last point to its first).
func (ft *Font) Outlines(s string, height float64) ([][]math.Point2, error) {
	scale := height / ft.upem
	ppem := fixed.Int26_6(int(ft.upem) << 6) // ppem = unitsPerEm ⇒ 1 px == 1 design unit
	var buf sfnt.Buffer
	var out [][]math.Point2
	penX := 0.0
	for _, r := range s {
		idx, err := ft.f.GlyphIndex(&buf, r)
		if err != nil {
			return nil, fmt.Errorf("text: glyph index %q: %w", r, err)
		}
		segs, err := ft.f.LoadGlyph(&buf, idx, ppem, nil)
		if err != nil {
			return nil, fmt.Errorf("text: load glyph %q: %w", r, err)
		}
		for _, c := range flattenSegments(segs) {
			out = append(out, scaleContour(c, penX, scale))
		}
		if adv, err := ft.f.GlyphAdvance(&buf, idx, ppem, font.HintingNone); err == nil {
			penX += f26(adv)
		}
	}
	return out, nil
}

// scaleContour offsets a design-unit contour by penX and scales it into model units. sfnt
// uses Y-DOWN (raster) coordinates, so Y is negated to put text upright in the Y-up sketch
// (the baseline stays at y=0; cap height becomes positive).
func scaleContour(c []pt, penX, scale float64) []math.Point2 {
	out := make([]math.Point2, len(c))
	for i, p := range c {
		out[i] = math.P2(math.Scalar((p.X+penX)*scale), math.Scalar(-p.Y*scale))
	}
	return out
}

// pt is a design-unit point during flattening.
type pt struct{ X, Y float64 }

func f26(v fixed.Int26_6) float64        { return float64(v) / 64 }
func ptOf(p fixed.Point26_6) pt          { return pt{f26(p.X), f26(p.Y)} }
func lerp(a, b pt, t float64) pt         { return pt{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t} }
func quad(a, c, b pt, t float64) pt      { return lerp(lerp(a, c, t), lerp(c, b, t), t) }
func cube(a, c1, c2, b pt, t float64) pt { return lerp(quad(a, c1, c2, t), quad(c1, c2, b, t), t) }

// flattenSegments turns a glyph's sfnt segments into closed polygon contours (design units),
// flattening quadratic/cubic curves and dropping each contour's closing duplicate point.
func flattenSegments(segs sfnt.Segments) [][]pt {
	var contours [][]pt
	var cur []pt
	var prev pt
	flush := func() {
		if n := len(cur); n >= 2 && samePt(cur[0], cur[n-1]) {
			cur = cur[:n-1]
		}
		if len(cur) >= 3 {
			contours = append(contours, cur)
		}
		cur = nil
	}
	for _, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			flush()
			prev = ptOf(s.Args[0])
			cur = []pt{prev}
		case sfnt.SegmentOpLineTo:
			prev = ptOf(s.Args[0])
			cur = append(cur, prev)
		case sfnt.SegmentOpQuadTo:
			c, b := ptOf(s.Args[0]), ptOf(s.Args[1])
			for i := 1; i <= curveFlattenSteps; i++ {
				cur = append(cur, quad(prev, c, b, float64(i)/curveFlattenSteps))
			}
			prev = b
		case sfnt.SegmentOpCubeTo:
			c1, c2, b := ptOf(s.Args[0]), ptOf(s.Args[1]), ptOf(s.Args[2])
			for i := 1; i <= curveFlattenSteps; i++ {
				cur = append(cur, cube(prev, c1, c2, b, float64(i)/curveFlattenSteps))
			}
			prev = b
		}
	}
	flush()
	return contours
}

func samePt(a, b pt) bool {
	const eps = 1e-6
	return a.X-b.X < eps && b.X-a.X < eps && a.Y-b.Y < eps && b.Y-a.Y < eps
}
