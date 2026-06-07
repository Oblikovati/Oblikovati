//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/math"
	"oblikovati/model/doc"
	"oblikovati/scene"
)

// viewCubeMenuID is the ImGui id of the ViewCube right-click menu.
const viewCubeMenuID = "##viewcube-menu"

// viewCubeContextMenu renders the ViewCube right-click menu when open (opened via OpenPopup
// at the cube right-click sites): Home actions, projection modes, and the compass toggle —
// all acting on the active view. Call once per frame inside the viewport window.
func viewCubeContextMenu(s *app.Session) {
	if !native.BeginPopup(viewCubeMenuID) {
		return
	}
	if native.MenuItem("Go Home") {
		s.GoHome()
	}
	if native.MenuItem("Set Current View as Home (Fixed Distance)") {
		s.SetActiveViewHome(false)
	}
	if native.MenuItem("Set Current View as Home (Fit to View)") {
		s.SetActiveViewHome(true)
	}
	native.Separator()
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
	native.Separator()
	if native.MenuItem(projItemLabel("Show Compass", s.ShowCompass())) {
		s.SetShowCompass(!s.ShowCompass())
	}
	native.EndPopup()
}

// drawCompass paints a North ring in the cube's ground plane (z = −1), projected with the
// cube so it foreshortens at its base, with an "N" tick at world +Y. Conveys the model's
// heading as the view orbits.
func drawCompass(cam scene.Camera, cx, cy float32) {
	right, up, fwd := camBasis(cam)
	const rc, segs = 1.5, 48 // ring radius in cube units, just outside the base
	var px, py float32
	for i := 0; i <= segs; i++ {
		t := float64(i) / segs * 2 * stdmath.Pi
		c := project(math.V3(rc*stdmath.Cos(t), rc*stdmath.Sin(t), -1), right, up, fwd, viewCubeRadius)
		x, y := cx+c.sx, cy+c.sy
		if i > 0 {
			native.DrawLine(px, py, x, y, viewCubeCompassColor, 1.4)
		}
		px, py = x, y
	}
	n := project(math.V3(0, rc, -1), right, up, fwd, viewCubeRadius) // +Y = North
	native.DrawText(cx+n.sx-3, cy+n.sy-7, "N", viewCubeCompassColor)
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
	viewCubeRadius   = 36   // cube half-size projected, px (center → face)
	viewCubeMargin   = 64   // cube-center inset from the tile's top-right corner, px
	viewCubeEdgeW    = 1.6  // cube edge line thickness, px
	viewCubeLabelW   = 1.4  // face-label stroke thickness, px
	viewCubeHomeR    = 13   // home-button half-size, px
	viewCubeReach    = 62.4 // max projected half-extent of the rotating cube (√3·radius), px
	viewCubeHomeGap  = 8    // clear margin between the cube's reach and the home button, px
	viewCubeSnapSecs = 0.35
)

// ViewCube colors. Faces are a light translucent panel; the hovered region's faces tint to
// the accent. (Theming via tokens is a Phase-C follow-up.)
var (
	viewCubeFaceColor    = [4]float32{0.85, 0.86, 0.88, 1.0}  // opaque light gray
	viewCubeHoverColor   = [4]float32{0.36, 0.66, 0.96, 0.95} // accent on hover
	viewCubeEdgeColor    = [4]float32{0.50, 0.52, 0.56, 1.0}  // medium gray, not black
	viewCubeTextColor    = [4]float32{0.20, 0.22, 0.26, 1}
	viewCubeHomeColor    = [4]float32{0.62, 0.66, 0.72, 0.95}
	viewCubeCompassColor = [4]float32{0.46, 0.52, 0.60, 0.95}
)

// drawViewCube paints the navigation cube centered at screen (cx,cy) for the camera, with
// the hovered region (if any) tinted and the home button highlighted when homeHovered.
// Drawn after the tile image so it sits on top; uses screen coordinates (ImGui draw list).
func drawViewCube(cam scene.Camera, cx, cy float32, hovered *Region, homeHovered, compass bool) {
	if compass {
		drawCompass(cam, cx, cy) // under the cube faces
	}
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
		// Label painted IN the face plane (projected with the cube), not screen-aligned.
		for _, s := range faceLabelSegments(f, cam, viewCubeRadius) {
			native.DrawLine(cx+s[0], cy+s[1], cx+s[2], cy+s[3], viewCubeTextColor, viewCubeLabelW)
		}
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

// homeCenter is the home button's screen center: below and to the LEFT of the cube
// (bottom-left). The vertical offset clears the cube's FULL rotational reach (√3·radius,
// a corner pointing at the viewer) plus a margin + the button radius, so the cube never
// touches the button at any orientation.
func homeCenter(cx, cy float32) (float32, float32) {
	return cx - viewCubeRadius, cy + viewCubeReach + viewCubeHomeGap + viewCubeHomeR
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
