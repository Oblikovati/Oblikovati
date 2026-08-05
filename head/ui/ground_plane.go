//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"

	"oblikovati.org/api/contract"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
	"oblikovati.org/renderer"

	gmath "oblikovati.org/math"
)

// groundAlbedo is the neutral matte grey of the shadow-catching ground plane.
var groundAlbedo = [4]float32{0.55, 0.55, 0.57, 1}

// groundPlaneSession is what drawing the ground needs: the active document's display settings
// and the visual style. A consumer-side interface rather than the whole *app.Session (audit I5,
// the arrowSession pattern).
type groundPlaneSession interface {
	ActiveDocument() *doc.Document
	DocumentDisplaySettings(id doc.ID) contract.DisplaySettings
	VisualStyle() renderer.VisualStyle
}

var _ groundPlaneSession = (*app.Session)(nil)

// wantGround reports whether to draw the ground plane: the document's display-settings keep it
// visible (M16-F07 #643) and the active style shades faces (a wireframe style has no surfaces to
// draw it on).
//
// It deliberately does NOT consult Ground Shadows. That toggle controls whether the ground
// RECEIVES the cast shadow — applyShadow's castDirect — not whether the ground exists. Gating
// both on it made View ▸ Ground Plane a no-op in either direction on a fresh part, where ground
// shadows are off (#2042), while its tooltip promised it shows and hides the ground.
func wantGround(s groundPlaneSession) bool {
	if !displayGroundVisible(s) {
		return false
	}
	return renderer.PassSetFor(s.VisualStyle()).Faces != renderer.ShadeNone
}

// displayGroundVisible reports the active document's display-settings ground-plane visibility
// (defaulting to visible when there is no active document).
func displayGroundVisible(s groundPlaneSession) bool {
	if s.ActiveDocument() == nil {
		return true
	}
	return s.DocumentDisplaySettings(0).GroundPlane().Visible()
}

// displayGroundColor is the active document's display-settings ground-plane color as an rgba
// float array, falling back to the neutral grey when there is no active document.
func displayGroundColor(s groundPlaneSession) [4]float32 {
	if s.ActiveDocument() == nil {
		return groundAlbedo
	}
	return s.DocumentDisplaySettings(0).GroundPlane().Color().Rgba().Array()
}

// groundPlaneItem builds the shadow-catching ground: a large horizontal quad (Y is up) at the
// model's base, centered under it and sized to a margin around its footprint. color is the
// display-settings ground color. It shades with the active face mode so it receives IBL and the
// sun shadow map like any surface (ADR-0026 §6). cullMode is NONE and the shader is two-sided,
// so winding is irrelevant.
func groundPlaneItem(min, max [3]float32, shading renderer.Shading, color [4]float32) renderer.DrawItem {
	cx := float64(min[0]+max[0]) * 0.5
	cz := float64(min[2]+max[2]) * 0.5
	y := float64(min[1]) // the floor sits at the lowest point of the model
	ext := 1.5 * math.Max(math.Max(float64(max[0]-min[0]), float64(max[2]-min[2])), 1)
	x0, x1 := cx-ext, cx+ext
	z0, z1 := cz-ext, cz+ext
	pos := []gmath.Point3{
		gmath.P3(x0, y, z0), gmath.P3(x1, y, z0), gmath.P3(x1, y, z1), gmath.P3(x0, y, z1),
	}
	up := gmath.V3(0, 1, 0)
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: pos,
		Normals:   []gmath.Vector3{up, up, up, up},
		Indices:   []int{0, 1, 2, 0, 2, 3},
		Color:     color,
		Metallic:  0,
		Roughness: 0.95,
		Opacity:   1,
		Shading:   shading,
	}
}
