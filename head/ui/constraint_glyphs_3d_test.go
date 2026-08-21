//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// fake3DGlyphs is a constraint3DGlyphSource returning a fixed marker list, so the overlay can be
// exercised without a live session.
type fake3DGlyphs []app.ConstraintGlyphView3D

func (f fake3DGlyphs) SketchConstraintGlyphs3D() []app.ConstraintGlyphView3D { return f }

func lookingDownCamera() scene.Camera {
	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	return cam
}

// TestConstraint3DGlyphOverlayDrawsAMarkerPerGlyph: each 3D marker becomes camera-facing line
// strokes; two markers with strokes produce a non-empty line item. An empty source draws nothing.
func TestConstraint3DGlyphOverlayDrawsAMarkerPerGlyph(t *testing.T) {
	cam := lookingDownCamera()
	if got := constraint3DGlyphOverlay(fake3DGlyphs(nil), cam, 0.2); got != nil {
		t.Errorf("no glyphs: got %d items, want nil", len(got))
	}
	src := fake3DGlyphs{
		{Kind: sketch.ParallelKind, At: math.P3(1, 0, 0)},
		{Kind: sketch.CoincidentKind, At: math.P3(0, 1, 0)},
	}
	items := constraint3DGlyphOverlay(src, cam, 0.2)
	if len(items) == 0 {
		t.Fatal("two 3D markers drew no items")
	}
	segs := 0
	for _, it := range items {
		segs += len(it.Positions)
	}
	if segs == 0 {
		t.Error("marker items carry no stroke vertices")
	}
	// A camera looking down −Z billboards into the XY plane, so a marker at (1,0,0) has strokes
	// spread around z=0 with x near 1 — i.e. it renders where the constraint lives, not at the origin.
	var sawNearAnchor bool
	for _, it := range items {
		for _, p := range it.Positions {
			if stdmath.Abs(float64(p.X)-1) <= 0.5 && stdmath.Abs(float64(p.Z)) <= 0.5 {
				sawNearAnchor = true
			}
		}
	}
	if !sawNearAnchor {
		t.Error("no stroke vertex near the (1,0,0) anchor — the billboard is not placed at the marker")
	}
}

// TestCameraBillboardAxesArePerpendicular: the billboard basis is orthonormal for a real camera and
// declines a degenerate one (view direction parallel to up).
func TestCameraBillboardAxesArePerpendicular(t *testing.T) {
	right, up, ok := cameraBillboardAxes(lookingDownCamera())
	if !ok {
		t.Fatal("a camera looking down −Z has a valid facing plane")
	}
	if d := float64(right.AsVector().Dot(up.AsVector())); stdmath.Abs(d) > 1e-9 {
		t.Errorf("right·up = %g, want 0 (orthonormal billboard axes)", d)
	}

	degenerate := scene.NewCamera(400, 400)
	degenerate.Eye, degenerate.Target, degenerate.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 0, 1)
	if _, _, ok := cameraBillboardAxes(degenerate); ok {
		t.Error("a view direction parallel to up has no facing plane; want ok=false")
	}
}
