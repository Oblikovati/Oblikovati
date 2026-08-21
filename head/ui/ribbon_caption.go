//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "strings"

// Large ribbon buttons stack their caption under the glyph, so an unwrapped multi-word label
// sets the tile's width all by itself: at the chrome's default font "Replace Face" measures
// roughly three times the 40px glyph frame, and a Modify panel of thirteen such tiles ran off
// the right of a maximized 3840px window. Wrapping the caption to a second line at a word
// boundary keeps each tile near its glyph width, which is the cheapest way to make a tab fit
// before the panel-collapse system (ribbon_collapse.go) has to hide anything behind a flyout.
//
// This lives apart from chrome_ribbon.go and takes its text measurer as a parameter so the
// wrap decision is a pure function, testable without a window or a font atlas.

// largeCaptionMaxLines is how many lines a large button's caption may wrap onto. Two: enough
// for every registered command name, and the ceiling the band reserves height for
// (largeButtonHeight). A label still too wide at two lines is left long rather than truncated —
// a clipped command name is worse than a wide tile, and the panel it widens can still collapse.
const largeCaptionMaxLines = 2

// wrapCaption splits a large button's caption so no line is wider than maxW, up to
// largeCaptionMaxLines lines, measuring with the given text measurer. The split point is the
// word boundary that minimises the widest resulting line, so "New 2D Sketch" breaks as
// "New 2D" / "Sketch" rather than "New" / "2D Sketch". A label that already fits, a
// single-word label, and an empty label all come back as one line — there is nothing to break.
func wrapCaption(label string, maxW float32, width func(string) float32) []string {
	if width(label) <= maxW {
		return []string{label}
	}
	words := strings.Fields(label)
	if len(words) < 2 {
		return []string{label} // one long word: no boundary to break at
	}
	best, bestWidest := 1, float32(-1)
	for split := 1; split < len(words); split++ {
		head := strings.Join(words[:split], " ")
		tail := strings.Join(words[split:], " ")
		widest := maxF32(width(head), width(tail))
		if bestWidest < 0 || widest < bestWidest {
			best, bestWidest = split, widest
		}
	}
	return []string{strings.Join(words[:best], " "), strings.Join(words[best:], " ")}
}
