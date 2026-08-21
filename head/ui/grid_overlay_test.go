//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
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

// eyeAbove looks straight down the +Z axis, so the XY-plane grid nudges toward +Z (#2087).
var eyeAbove = math.P3(0, 0, 100)

func TestGridOverlayNonPositiveSpacing(t *testing.T) {
	if got := gridOverlay(xyPlane(t), 0, 5, eyeAbove); got != nil {
		t.Errorf("spacing 0: got %d items, want nil", len(got))
	}
	if got := gridOverlay(xyPlane(t), -1, 5, eyeAbove); got != nil {
		t.Errorf("spacing -1: got %d items, want nil", len(got))
	}
}

// TestGridOverlayOriginAxisColors: the line through the origin along X is red (axisColorX)
// and the line along Y is green (axisColorY), each a single segment spanning the grid. Both sit at
// the #2087 nudge height (a hair toward the eye off the host plane), not exactly on z=0.
func TestGridOverlayOriginAxisColors(t *testing.T) {
	const spacing = 1.0
	half := float64(gridCells) * spacing
	z := gridCoplanarNudge * half // the grid is lifted toward +Z (eye above); |origin| == 0 < half
	items := gridOverlay(xyPlane(t), spacing, 5, eyeAbove)

	xItem, ok := itemWithColor(items, axisColorX)
	if !ok {
		t.Fatalf("no X-axis (red) grid item; colors=%v", colorsOf(items))
	}
	assertAxisSegment(t, "X", xItem, math.P3(-half, 0, math.Scalar(z)), math.P3(half, 0, math.Scalar(z)))

	yItem, ok := itemWithColor(items, axisColorY)
	if !ok {
		t.Fatalf("no Y-axis (green) grid item; colors=%v", colorsOf(items))
	}
	assertAxisSegment(t, "Y", yItem, math.P3(0, -half, math.Scalar(z)), math.P3(0, half, math.Scalar(z)))
}

// TestGridOverlayNudgesTowardEye is the #2087 regression: the coplanar grid must be lifted off its
// host plane TOWARD the eye — strictly off the plane (never left at z=0, where it z-fights the host
// face) and on the eye's side (never pushed away). Asserting a strict sign, not an exact value,
// keeps the guard honest: dropping the nudge (offset → 0) leaves the grid on the plane and fails
// here, which an assertion derived from the nudge constant would not catch. The lift stays a hair —
// far under one grid cell — so it wins the depth test without visibly floating above the face.
func TestGridOverlayNudgesTowardEye(t *testing.T) {
	const spacing = 1.0
	aboveZ := float64(gridZ(t, gridOverlay(xyPlane(t), spacing, 5, math.P3(0, 0, 100))))
	if aboveZ <= 0 || aboveZ >= spacing {
		t.Errorf("eye above the XY plane: grid Z = %g, want a small +Z lift in (0, %g)", aboveZ, spacing)
	}
	belowZ := float64(gridZ(t, gridOverlay(xyPlane(t), spacing, 5, math.P3(0, 0, -100))))
	if belowZ >= 0 || belowZ <= -spacing {
		t.Errorf("eye below the XY plane: grid Z = %g, want a small −Z lift in (-%g, 0)", belowZ, spacing)
	}
	if stdmath.Abs(aboveZ+belowZ) > 1e-12 {
		t.Errorf("lift is not symmetric about the plane: above=%g below=%g", aboveZ, belowZ)
	}
}

// gridZ returns the Z of the X-axis grid item's first endpoint — the height the whole grid was
// lifted to (every vertex shares the same normal offset on the XY plane).
func gridZ(t *testing.T, items []renderer.DrawItem) math.Scalar {
	t.Helper()
	it, ok := itemWithColor(items, axisColorX)
	if !ok {
		t.Fatalf("no X-axis grid item; colors=%v", colorsOf(items))
	}
	return it.Positions[0].Z
}

// TestGridOverlayKeepsMinorAndMajor: the axis split does not drop the minor/major groups.
func TestGridOverlayKeepsMinorAndMajor(t *testing.T) {
	items := gridOverlay(xyPlane(t), 1.0, 5, eyeAbove)
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
