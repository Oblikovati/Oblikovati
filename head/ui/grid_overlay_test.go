//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// xyPlane is the model XY plane, so ToModel(P2(u,v)) == (u, v, 0) and grid lines map
// straight onto the world axes for assertion.
func xyPlane(t *testing.T) sketch.Plane {
	t.Helper()
	p, err := sketch.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}

func itemWithColor(items []renderer.DrawItem, color [4]float32) (renderer.DrawItem, bool) {
	for _, it := range items {
		if it.Color == color {
			return it, true
		}
	}
	return renderer.DrawItem{}, false
}

func TestGridOverlayNonPositiveSpacing(t *testing.T) {
	if got := gridOverlay(xyPlane(t), 0, 5); got != nil {
		t.Errorf("spacing 0: got %d items, want nil", len(got))
	}
	if got := gridOverlay(xyPlane(t), -1, 5); got != nil {
		t.Errorf("spacing -1: got %d items, want nil", len(got))
	}
}

// TestGridOverlayOriginAxisColors: the line through the origin along X is red (axisColorX)
// and the line along Y is green (axisColorY), each a single segment spanning the grid.
func TestGridOverlayOriginAxisColors(t *testing.T) {
	const spacing = 1.0
	half := float64(gridCells) * spacing
	items := gridOverlay(xyPlane(t), spacing, 5)

	xItem, ok := itemWithColor(items, axisColorX)
	if !ok {
		t.Fatalf("no X-axis (red) grid item; colors=%v", colorsOf(items))
	}
	assertAxisSegment(t, "X", xItem, math.P3(-half, 0, 0), math.P3(half, 0, 0))

	yItem, ok := itemWithColor(items, axisColorY)
	if !ok {
		t.Fatalf("no Y-axis (green) grid item; colors=%v", colorsOf(items))
	}
	assertAxisSegment(t, "Y", yItem, math.P3(0, -half, 0), math.P3(0, half, 0))
}

// TestGridOverlayKeepsMinorAndMajor: the axis split does not drop the minor/major groups.
func TestGridOverlayKeepsMinorAndMajor(t *testing.T) {
	items := gridOverlay(xyPlane(t), 1.0, 5)
	if _, ok := itemWithColor(items, chromeTheme.gridMinorColor); !ok {
		t.Errorf("no minor grid item; colors=%v", colorsOf(items))
	}
	if _, ok := itemWithColor(items, chromeTheme.gridMajorColor); !ok {
		t.Errorf("no major grid item; colors=%v", colorsOf(items))
	}
}

func assertAxisSegment(t *testing.T, name string, it renderer.DrawItem, p, q math.Point3) {
	t.Helper()
	if it.Primitive != renderer.Lines {
		t.Errorf("%s axis: primitive=%v, want Lines", name, it.Primitive)
	}
	if len(it.Positions) != 2 || len(it.Indices) != 2 {
		t.Fatalf("%s axis: %d positions / %d indices, want a single segment", name, len(it.Positions), len(it.Indices))
	}
	// The XY plane maps integer grid coords exactly, so direct equality is safe here.
	if it.Positions[0] != p || it.Positions[1] != q {
		t.Errorf("%s axis endpoints = %v..%v, want %v..%v", name, it.Positions[0], it.Positions[1], p, q)
	}
}

func colorsOf(items []renderer.DrawItem) [][4]float32 {
	out := make([][4]float32, len(items))
	for i, it := range items {
		out[i] = it.Color
	}
	return out
}
