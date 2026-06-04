// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import "testing"

func mapper() *ColorMapper {
	// 0 → blue, 1 → red.
	return &ColorMapper{Values: []float64{0, 1}, Colors: [][4]float32{{0, 0, 1, 1}, {1, 0, 0, 1}}}
}

func TestColorMapperInterpolatesMidpoint(t *testing.T) {
	got := mapper().At(0.5)
	want := [4]float32{0.5, 0, 0.5, 1}
	if got != want {
		t.Errorf("At(0.5) = %v, want %v", got, want)
	}
}

func TestColorMapperClampsBelowAndAbove(t *testing.T) {
	m := mapper()
	if got := m.At(-3); got != (m.Colors[0]) {
		t.Errorf("At(-3) = %v, want clamp to %v", got, m.Colors[0])
	}
	if got := m.At(9); got != (m.Colors[1]) {
		t.Errorf("At(9) = %v, want clamp to %v", got, m.Colors[1])
	}
}

func TestColorMapperPicksRightSegment(t *testing.T) {
	// 0 → black, 0.5 → green, 1 → white. A quarter-way is half into the first segment.
	m := &ColorMapper{Values: []float64{0, 0.5, 1}, Colors: [][4]float32{{0, 0, 0, 1}, {0, 1, 0, 1}, {1, 1, 1, 1}}}
	got := m.At(0.25)
	want := [4]float32{0, 0.5, 0, 1}
	if got != want {
		t.Errorf("At(0.25) = %v, want %v", got, want)
	}
}
