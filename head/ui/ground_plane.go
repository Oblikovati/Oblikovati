//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"

	"github.com/Oblikovati/oblikovati/app"
	gmath "github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/renderer"
)

// groundAlbedo is the neutral matte grey of the shadow-catching ground plane.
var groundAlbedo = [4]float32{0.55, 0.55, 0.57, 1}

// wantGround reports whether to draw the ground plane: the Ground Shadows toggle is on and the
// active style shades faces (a wireframe style has no surfaces to receive a shadow).
func wantGround(s *app.Session) bool {
	if !s.ShadowSettings().GroundShadows {
		return false
	}
	return renderer.PassSetFor(s.VisualStyle()).Faces != renderer.ShadeNone
}

// groundPlaneItem builds the shadow-catching ground: a large horizontal quad (Y is up) at the
// model's base, centered under it and sized to a margin around its footprint. It shades with
// the active face mode so it receives IBL and the sun shadow map like any surface (ADR-0026
// §6). cullMode is NONE and the shader is two-sided, so winding is irrelevant.
func groundPlaneItem(min, max [3]float32, shading renderer.Shading) renderer.DrawItem {
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
		Color:     groundAlbedo,
		Metallic:  0,
		Roughness: 0.95,
		Opacity:   1,
		Shading:   shading,
	}
}
