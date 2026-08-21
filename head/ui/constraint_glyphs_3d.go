//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Show Constraints in a 3D sketch (#1998): a marker per geometric constraint, drawn at the
// model-space anchor the app derives (SketchConstraintGlyphs3D). A 3D sketch has no host plane, so
// each marker is a camera-facing BILLBOARD — the same glyph vocabulary as the 2D overlay
// (constraint_glyphs.go), drawn in a plane whose axes are the camera's right and up so the pictogram
// always faces the viewer and holds its screen size at any orbit. Display-only for now: 3D markers
// are not yet pickable, so they draw in one colour (a 3D constraint is deleted from the browser).

// constraint3DGlyphSource is the slice of the session this overlay reads — the 3D markers and
// nothing else (the consumer-interface pattern, so the overlay cannot touch the model).
type constraint3DGlyphSource interface {
	SketchConstraintGlyphs3D() []app.ConstraintGlyphView3D
}

var _ constraint3DGlyphSource = (*app.Session)(nil)

// constraint3DGlyphOverlay builds the camera-facing marker items for the active 3D sketch's
// constraints. hWorld is the marker half-size in world units (screen-constant, as for the 2D
// overlay). Empty when constraints are hidden, no 3D sketch is open, or the camera is degenerate.
func constraint3DGlyphOverlay(s constraint3DGlyphSource, cam scene.Camera, hWorld float64) []renderer.DrawItem {
	glyphs := s.SketchConstraintGlyphs3D()
	if len(glyphs) == 0 {
		return nil
	}
	right, up, ok := cameraBillboardAxes(cam)
	if !ok {
		return nil
	}
	acc := &segAccum{}
	for _, g := range glyphs {
		billboard, err := sketch.NewPlane(g.At, right, up)
		if err != nil {
			continue // right⊥up by construction, so this cannot happen for a real camera
		}
		constraintGlyphSegments(acc, billboard, math.P2(0, 0), hWorld, g.Kind)
	}
	return appendGrid(nil, acc, chromeTheme.sketchColor)
}

// cameraBillboardAxes returns an orthonormal (right, up) pair spanning the plane that faces the
// camera: right = forward × up, then up = right × forward so the two are perpendicular by
// construction. ok is false only for a degenerate camera (view direction parallel to its up
// vector), where no facing plane exists.
func cameraBillboardAxes(cam scene.Camera) (right, up math.UnitVector3, ok bool) {
	forward := cam.Forward()
	r, err := math.UnitVector3FromVector(forward.Cross(cam.Up))
	if err != nil {
		return math.UnitVector3{}, math.UnitVector3{}, false
	}
	u, err := math.UnitVector3FromVector(r.AsVector().Cross(forward))
	if err != nil {
		return math.UnitVector3{}, math.UnitVector3{}, false
	}
	return r, u, true
}
