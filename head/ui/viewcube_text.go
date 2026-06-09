// SPDX-License-Identifier: GPL-2.0-only

// Pure-Go (no cgo): ViewCube face labels are painted as vector strokes laid out IN each
// face's 3D plane and projected with the cube, so the text foreshortens and rotates with
// the face (like paint on the cube) instead of floating screen-aligned. A tiny uppercase
// stroke font covers the six face words (TOP/BOTTOM/FRONT/BACK/LEFT/RIGHT).
package ui

import (
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// Label layout within a face, in unit-cube face units (the face spans [-1,1]); chosen so a
// 6-letter word (BOTTOM) fits comfortably.
const (
	labelGlyphH   = 0.34 // glyph cell height
	labelAdvance  = 0.50 // per-glyph advance (cell width + gap)
	labelGlyphW   = 0.38 // drawn glyph width within its advance
	labelMaxWidth = 1.55 // shrink the whole word to fit this
)

// faceTextFrame returns a labeled face's center and in-plane text axes (u = reading
// direction, v = up) on the unit cube, so a glyph at face-local (du,dv) sits at
// center + u·du + v·dv — on the face, never floating.
func faceTextFrame(r Region) (center, u, v math.Vector3) {
	switch {
	case r.Z == 1: // TOP
		return math.V3(0, 0, 1), math.V3(1, 0, 0), math.V3(0, 1, 0)
	case r.Z == -1: // BOTTOM
		return math.V3(0, 0, -1), math.V3(1, 0, 0), math.V3(0, -1, 0)
	case r.Y == -1: // FRONT
		return math.V3(0, -1, 0), math.V3(1, 0, 0), math.V3(0, 0, 1)
	case r.Y == 1: // BACK
		return math.V3(0, 1, 0), math.V3(-1, 0, 0), math.V3(0, 0, 1)
	case r.X == 1: // RIGHT — viewed from +X, screen-right is +Y
		return math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)
	default: // LEFT (X == -1) — viewed from −X, screen-right is −Y
		return math.V3(-1, 0, 0), math.V3(0, -1, 0), math.V3(0, 0, 1)
	}
}

// faceLabelSegments projects a labeled face's word into the face plane and returns the
// screen-space line segments (offsets from the cube center, px) that draw it. Empty for an
// unlabeled face.
func faceLabelSegments(f cubeFace, cam scene.Camera, o doc.CubeOrient, radius float32) [][4]float32 {
	label := f.region.Label
	if label == "" {
		return nil
	}
	scale := 1.0
	if total := float64(len(label)) * labelAdvance; total > labelMaxWidth {
		scale = labelMaxWidth / total
	}
	adv := labelAdvance * scale
	right, up, fwd := cubeBasis(cam, o)
	center, uAxis, vAxis := faceTextFrame(f.region)
	lay := labelLayout{center, uAxis, vAxis, right, up, fwd, radius, labelGlyphW * scale, labelGlyphH * scale}
	startU := -float64(len(label)) * adv / 2 // left edge of the word, centered on the face
	var segs [][4]float32
	for i, r := range label {
		baseU := startU + float64(i)*adv + (adv-lay.gw)/2 // glyph cell left + side bearing
		segs = append(segs, lay.glyphSegs(baseU, r)...)
	}
	return segs
}

// labelLayout holds the resolved frame + scale for projecting a face label's glyph strokes.
type labelLayout struct {
	center, uAxis, vAxis math.Vector3
	right, up, fwd       math.Vector3
	radius               float32
	gw, gh               float64
}

// glyphSegs projects one glyph's strokes (at cell-left baseU) into screen-space segments.
func (l labelLayout) glyphSegs(baseU float64, r rune) [][4]float32 {
	var segs [][4]float32
	for _, s := range glyphStrokes(r) {
		p0 := facePoint(l.center, l.uAxis, l.vAxis, baseU+s[0]*l.gw, (s[1]-0.5)*l.gh)
		p1 := facePoint(l.center, l.uAxis, l.vAxis, baseU+s[2]*l.gw, (s[3]-0.5)*l.gh)
		c0 := project(p0, l.right, l.up, l.fwd, l.radius)
		c1 := project(p1, l.right, l.up, l.fwd, l.radius)
		segs = append(segs, [4]float32{c0.sx, c0.sy, c1.sx, c1.sy})
	}
	return segs
}

// facePoint maps a face-local (du along u, dv along v) to a point on the unit cube's face.
func facePoint(center, u, v math.Vector3, du, dv float64) math.Vector3 {
	return center.Add(u.Scale(du)).Add(v.Scale(dv))
}

// glyphStrokes returns a glyph's strokes as {x0,y0,x1,y1} in a unit cell ([0,1]², y up).
// Only the uppercase letters used by the six face words are defined; others draw nothing.
func glyphStrokes(r rune) [][4]float64 {
	return strokeFont[r]
}

// strokeFont is a minimal uppercase vector font for T O P B M F R N A C K L E I G H.
var strokeFont = map[rune][][4]float64{
	'T': {{0.1, 0.9, 0.9, 0.9}, {0.5, 0.9, 0.5, 0.1}},
	'O': {{0.2, 0.1, 0.8, 0.1}, {0.8, 0.1, 0.8, 0.9}, {0.8, 0.9, 0.2, 0.9}, {0.2, 0.9, 0.2, 0.1}},
	'P': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.75, 0.9}, {0.75, 0.9, 0.75, 0.5}, {0.75, 0.5, 0.2, 0.5}},
	'B': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.7, 0.9}, {0.7, 0.9, 0.7, 0.5}, {0.7, 0.5, 0.2, 0.5},
		{0.2, 0.5, 0.78, 0.5}, {0.78, 0.5, 0.78, 0.1}, {0.78, 0.1, 0.2, 0.1}},
	'M': {{0.1, 0.1, 0.1, 0.9}, {0.1, 0.9, 0.5, 0.5}, {0.5, 0.5, 0.9, 0.9}, {0.9, 0.9, 0.9, 0.1}},
	'F': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.85, 0.9}, {0.2, 0.5, 0.7, 0.5}},
	'R': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.7, 0.9}, {0.7, 0.9, 0.7, 0.5}, {0.7, 0.5, 0.2, 0.5}, {0.4, 0.5, 0.82, 0.1}},
	'N': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.8, 0.1}, {0.8, 0.1, 0.8, 0.9}},
	'A': {{0.1, 0.1, 0.5, 0.9}, {0.5, 0.9, 0.9, 0.1}, {0.27, 0.42, 0.73, 0.42}},
	'C': {{0.82, 0.9, 0.2, 0.9}, {0.2, 0.9, 0.2, 0.1}, {0.2, 0.1, 0.82, 0.1}},
	'K': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.5, 0.82, 0.9}, {0.2, 0.5, 0.82, 0.1}},
	'L': {{0.25, 0.9, 0.25, 0.1}, {0.25, 0.1, 0.82, 0.1}},
	'E': {{0.2, 0.1, 0.2, 0.9}, {0.2, 0.9, 0.82, 0.9}, {0.2, 0.5, 0.68, 0.5}, {0.2, 0.1, 0.82, 0.1}},
	'I': {{0.5, 0.1, 0.5, 0.9}, {0.34, 0.9, 0.66, 0.9}, {0.34, 0.1, 0.66, 0.1}},
	'G': {{0.82, 0.9, 0.2, 0.9}, {0.2, 0.9, 0.2, 0.1}, {0.2, 0.1, 0.82, 0.1}, {0.82, 0.1, 0.82, 0.46}, {0.82, 0.46, 0.55, 0.46}},
	'H': {{0.2, 0.1, 0.2, 0.9}, {0.8, 0.1, 0.8, 0.9}, {0.2, 0.5, 0.8, 0.5}},
}
