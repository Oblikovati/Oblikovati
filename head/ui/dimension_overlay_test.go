//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// A selected dimension used to draw exactly like an unselected one, so even once clicking a
// dimension selected it the user saw no change and the feature still read as broken (#2017).

// TestSelectedDimensionDrawsInTheSelectionColour: the selected dimension's segments land in the
// selection-coloured item, and the unselected ones keep their driving/driven colours.
func TestSelectedDimensionDrawsInTheSelectionColour(t *testing.T) {
	views := []app.DimensionView{
		{Segments: seg(0, 0, 1, 0)},
		{Segments: seg(0, 1, 1, 1), Driven: true},
		{Segments: seg(0, 2, 1, 2), Selected: true},
	}
	items := dimensionLines(sketch.Plane{}, views)
	byColor := map[[4]float32]int{}
	for _, it := range items {
		byColor[it.Color] += len(it.Positions)
	}
	if byColor[chromeTheme.selectedPlaneColor] == 0 {
		t.Fatalf("no segments drawn in the selection colour; got %v", byColor)
	}
	if byColor[chromeTheme.dimensionSketchColor] == 0 || byColor[chromeTheme.dimensionDrivenColor] == 0 {
		t.Fatalf("selection bucket swallowed the driving/driven ones; got %v", byColor)
	}
}

// TestSelectionColourWinsOverDriven: a driven dimension that is selected draws as selected, not
// as driven — selection feedback must not be hidden by the dimension's own state.
func TestSelectionColourWinsOverDriven(t *testing.T) {
	driving, driven, selected := &segAccum{}, &segAccum{}, &segAccum{}
	got := dimensionAccum(app.DimensionView{Driven: true, Selected: true}, driving, driven, selected)
	if got != selected {
		t.Fatalf("a selected driven dimension went to the wrong colour bucket")
	}
}

// seg is one sketch-plane segment as a DimensionView carries them.
func seg(x1, y1, x2, y2 float64) [][2]math.Point2 {
	return [][2]math.Point2{{math.P2(math.Scalar(x1), math.Scalar(y1)), math.P2(math.Scalar(x2), math.Scalar(y2))}}
}
