//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/doc"
	"oblikovati/scene"
)

// viewCubeMenuID is the ImGui id of the ViewCube right-click projection menu.
const viewCubeMenuID = "##viewcube-projection"

// viewCubeProjectionMenu renders the ViewCube right-click menu when open (opened via
// OpenPopup at the cube right-click sites): a radio set of projection modes acting on the
// active view. Call once per frame inside the viewport window.
func viewCubeProjectionMenu(s *app.Session) {
	if !native.BeginPopup(viewCubeMenuID) {
		return
	}
	cur := s.ActiveViewProjection()
	if native.MenuItem(projItemLabel("Orthographic", cur == doc.ProjOrthographic)) {
		s.SetActiveViewProjection(doc.ProjOrthographic)
	}
	if native.MenuItem(projItemLabel("Perspective", cur == doc.ProjPerspective)) {
		s.SetActiveViewProjection(doc.ProjPerspective)
	}
	if native.MenuItem(projItemLabel("Perspective with Ortho Faces", cur == doc.ProjPerspectiveOrthoFaces)) {
		s.SetActiveViewProjection(doc.ProjPerspectiveOrthoFaces)
	}
	native.EndPopup()
}

// projItemLabel prefixes the active mode with a filled dot so the current projection reads
// as checked (the MenuItem binding has no built-in radio state).
func projItemLabel(name string, active bool) string {
	if active {
		return "● " + name
	}
	return "    " + name
}

const (
	viewCubeRadius   = 24  // cube half-size projected, px (center → face)
	viewCubeMargin   = 44  // cube-center inset from the tile's top-right corner, px
	viewCubeEdgeW    = 1.5 // cube edge line thickness, px
	viewCubeHomeR    = 9   // home-button half-size, px
	viewCubeHomeGap  = 16  // gap from cube bottom to the home button, px
	viewCubeSnapSecs = 0.35
)

// ViewCube colors. Faces are a light translucent panel; the hovered region's faces tint to
// the accent. (Theming via tokens is a Phase-C follow-up.)
var (
	viewCubeFaceColor  = [4]float32{0.82, 0.85, 0.90, 0.55}
	viewCubeHoverColor = [4]float32{0.36, 0.66, 0.96, 0.85}
	viewCubeEdgeColor  = [4]float32{0.20, 0.24, 0.30, 0.90}
	viewCubeTextColor  = [4]float32{0.12, 0.14, 0.18, 1}
	viewCubeHomeColor  = [4]float32{0.62, 0.66, 0.72, 0.95}
)

// drawViewCube paints the navigation cube centered at screen (cx,cy) for the camera, with
// the hovered region (if any) tinted and the home button highlighted when homeHovered.
// Drawn after the tile image so it sits on top; uses screen coordinates (ImGui draw list).
func drawViewCube(cam scene.Camera, cx, cy float32, hovered *Region, homeHovered bool) {
	for _, f := range visibleFaces(cam, viewCubeRadius) {
		col := viewCubeFaceColor
		if hovered != nil && faceInRegion(f.region, hovered) {
			col = viewCubeHoverColor
		}
		x0, y0 := cx+f.corner[0].sx, cy+f.corner[0].sy
		x1, y1 := cx+f.corner[1].sx, cy+f.corner[1].sy
		x2, y2 := cx+f.corner[2].sx, cy+f.corner[2].sy
		x3, y3 := cx+f.corner[3].sx, cy+f.corner[3].sy
		native.DrawTriangleFilled(x0, y0, x1, y1, x2, y2, col)
		native.DrawTriangleFilled(x0, y0, x2, y2, x3, y3, col)
		native.DrawLine(x0, y0, x1, y1, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x1, y1, x2, y2, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x2, y2, x3, y3, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x3, y3, x0, y0, viewCubeEdgeColor, viewCubeEdgeW)
		lx, ly := (x0+x2)/2, (y0+y2)/2 // face center
		native.DrawText(lx-float32(len(f.region.Label))*3, ly-7, f.region.Label, viewCubeTextColor)
	}
	hx, hy := homeCenter(cx, cy)
	drawHomeButton(hx, hy, homeHovered)
}

// faceInRegion reports whether face f is part of the hovered region (its axis sign matches),
// so a corner hover tints its three faces, an edge two, a face one.
func faceInRegion(f Region, h *Region) bool {
	switch {
	case f.X != 0:
		return h.X == f.X
	case f.Y != 0:
		return h.Y == f.Y
	default:
		return h.Z == f.Z
	}
}

// homeCenter is the home button's screen center, below the cube.
func homeCenter(cx, cy float32) (float32, float32) {
	return cx, cy + viewCubeRadius + viewCubeHomeGap
}

// drawHomeButton paints a small house glyph (roof triangle + body) at (hx,hy).
func drawHomeButton(hx, hy float32, hovered bool) {
	col := viewCubeHomeColor
	if hovered {
		col = viewCubeHoverColor
	}
	r := float32(viewCubeHomeR)
	// Body (square, lower two-thirds) as two triangles.
	bx0, by0 := hx-r*0.7, hy-r*0.1
	bx1, by1 := hx+r*0.7, hy+r*0.7
	native.DrawTriangleFilled(bx0, by0, bx1, by0, bx1, by1, col)
	native.DrawTriangleFilled(bx0, by0, bx1, by1, bx0, by1, col)
	// Roof (triangle) on top.
	native.DrawTriangleFilled(hx-r, hy-r*0.1, hx+r, hy-r*0.1, hx, hy-r, col)
}

// overHomeButton reports whether screen point (mx,my) is over the home button at cube
// center (cx,cy).
func overHomeButton(mx, my, cx, cy float32) bool {
	hx, hy := homeCenter(cx, cy)
	r := float32(viewCubeHomeR)
	return mx >= hx-r && mx <= hx+r && my >= hy-r && my <= hy+r
}
