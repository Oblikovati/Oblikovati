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
	if native.MenuItem("Set Current View as Front") {
		s.SetActiveViewAsFront()
	}
	if native.MenuItem("Reset Front") {
		s.ResetFront()
	}
	native.Separator()
	if native.MenuItem(projItemLabel("Lock to Current Selection", s.LockToSelection())) {
		s.SetLockToSelection(!s.LockToSelection())
	}
	if native.MenuItem(projItemLabel("Show Compass", s.ShowCompass())) {
		s.SetShowCompass(!s.ShowCompass())
	}
	if native.MenuItem("Options...") {
		showViewCubeOptions = true
	}
	native.EndPopup()
}

// showViewCubeOptions tracks whether the ViewCube Options window is open (toggled from the
// right-click menu's "Options…").
var showViewCubeOptions bool

// drawViewCubeOptions renders the ViewCube Options window (size/opacity/compass) when open.
// Settings persist as global user preferences. Call once per frame.
func drawViewCubeOptions(s *app.Session) {
	if !showViewCubeOptions {
		return
	}
	native.SetNextWindowSizeOnce(320, 200)
	if native.Begin("ViewCube Options") {
		compass := s.ShowCompass()
		if native.Checkbox("Show compass", &compass) {
			s.SetShowCompass(compass)
		}
		op := s.InactiveOpacity()
		if native.SliderFloat("Inactive opacity", &op, 0.1, 1.0) {
			s.SetInactiveOpacity(op)
		}
		size := s.CubeSize()
		if native.SliderFloat("Size", &size, 20, 80) { // matches app's CubeSize bounds
			s.SetCubeSize(int(size))
		}
		drawCornerCombo(s)
		if native.Button("Close") {
			showViewCubeOptions = false
		}
	}
	native.End()
}

// cornerNames labels the four anchor corners, indexed by Session.CubeCorner.
var cornerNames = []string{"Top-right", "Top-left", "Bottom-right", "Bottom-left"}

// drawCornerCombo renders the on-screen-position picker, persisting the chosen corner.
func drawCornerCombo(s *app.Session) {
	cur := s.CubeCorner()
	if !native.BeginCombo("Position", cornerNames[cur]) {
		return
	}
	for i, name := range cornerNames {
		if native.Selectable(name, i == cur) {
			s.SetCubeCorner(i)
		}
	}
	native.EndCombo()
}

// drawCompass paints a North ring in the cube's ground plane (z = −1), projected with the
// cube so it foreshortens at its base, with an "N" tick at world +Y. Conveys the model's
// heading as the view orbits.
func drawCompass(cam scene.Camera, o doc.CubeOrient, cx, cy, r float32) {
	right, up, fwd := cubeBasis(cam, o)
	const rc, segs = 1.5, 48 // ring radius in cube units, just outside the base
	var px, py float32
	for i := 0; i <= segs; i++ {
		t := float64(i) / segs * 2 * stdmath.Pi
		c := project(math.V3(rc*stdmath.Cos(t), rc*stdmath.Sin(t), -1), right, up, fwd, r)
		x, y := cx+c.sx, cy+c.sy
		if i > 0 {
			native.DrawLine(px, py, x, y, viewCubeCompassColor, 1.4)
		}
		px, py = x, y
	}
	n := project(math.V3(0, rc, -1), right, up, fwd, r) // +Y = North
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
	viewCubeEdgeW    = 1.6 // cube edge line thickness, px
	viewCubeLabelW   = 1.4 // face-label stroke thickness, px
	viewCubeSnapSecs = 0.35
	// Placement geometry as ratios of the (runtime) cube radius, so changing the size
	// scales the margins, the rotational reach, and the home button together.
	viewCubeMarginRatio  = 1.78 // cube-center inset from the chosen corner
	viewCubeReachRatio   = 1.74 // max projected half-extent of the rotating cube (≈√3)
	viewCubeHomeGapRatio = 0.22 // clear margin between the cube's reach and the home button
	viewCubeHomeRRatio   = 0.36 // home-button half-size
)

// ViewCube anchor corners (match Session.CubeCorner).
const (
	cornerTR = 0
	cornerTL = 1
	cornerBR = 2
	cornerBL = 3
)

// cubePlacement is the resolved on-screen geometry of a ViewCube within a tile: the cube
// center + radius and the home-button center + radius. The home button is placed toward the
// panel interior from the cube (per corner) so it never clips the panel edge or overlaps the
// rotating cube.
type cubePlacement struct {
	cx, cy float32
	r      float32
	homeX  float32
	homeY  float32
	homeR  float32
}

// placeViewCube resolves the cube + home-button geometry for a tile (screen rect
// bx,by,pw,ph) at the given radius and anchor corner.
func placeViewCube(bx, by float32, pw, ph int, r float32, corner int) cubePlacement {
	m := r * viewCubeMarginRatio
	left := corner == cornerTL || corner == cornerBL
	top := corner == cornerTR || corner == cornerTL
	cx, cy := bx+float32(pw)-m, by+m
	if left {
		cx = bx + m
	}
	if !top {
		cy = by + float32(ph) - m
	}
	sx, sy := float32(-1), float32(1) // toward the panel interior (away from the corner)
	if left {
		sx = 1
	}
	if !top {
		sy = -1
	}
	homeR := r * viewCubeHomeRRatio
	homeOff := r*viewCubeReachRatio + r*viewCubeHomeGapRatio + homeR
	return cubePlacement{cx: cx, cy: cy, r: r, homeX: cx + sx*r, homeY: cy + sy*homeOff, homeR: homeR}
}

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
func drawViewCube(cam scene.Camera, o doc.CubeOrient, p cubePlacement, hovered *Region, homeHovered, compass bool, opacity float32) {
	if compass {
		drawCompass(cam, o, p.cx, p.cy, p.r) // under the cube faces
	}
	for _, f := range visibleFaces(cam, o, p.r) {
		col := viewCubeFaceColor
		if hovered != nil && faceInRegion(f.region, hovered) {
			col = viewCubeHoverColor
		} else {
			col[3] = opacity // inactive faces honor the user's opacity preference
		}
		x0, y0 := p.cx+f.corner[0].sx, p.cy+f.corner[0].sy
		x1, y1 := p.cx+f.corner[1].sx, p.cy+f.corner[1].sy
		x2, y2 := p.cx+f.corner[2].sx, p.cy+f.corner[2].sy
		x3, y3 := p.cx+f.corner[3].sx, p.cy+f.corner[3].sy
		native.DrawTriangleFilled(x0, y0, x1, y1, x2, y2, col)
		native.DrawTriangleFilled(x0, y0, x2, y2, x3, y3, col)
		native.DrawLine(x0, y0, x1, y1, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x1, y1, x2, y2, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x2, y2, x3, y3, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x3, y3, x0, y0, viewCubeEdgeColor, viewCubeEdgeW)
		// Label painted IN the face plane (projected with the cube), not screen-aligned.
		for _, s := range faceLabelSegments(f, cam, o, p.r) {
			native.DrawLine(p.cx+s[0], p.cy+s[1], p.cx+s[2], p.cy+s[3], viewCubeTextColor, viewCubeLabelW)
		}
	}
	drawHomeButton(p.homeX, p.homeY, p.homeR, homeHovered)
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

// drawHomeButton paints a small house glyph (roof triangle + body) at (hx,hy), sized by r.
func drawHomeButton(hx, hy, r float32, hovered bool) {
	col := viewCubeHomeColor
	if hovered {
		col = viewCubeHoverColor
	}
	// Body (square, lower two-thirds) as two triangles.
	bx0, by0 := hx-r*0.7, hy-r*0.1
	bx1, by1 := hx+r*0.7, hy+r*0.7
	native.DrawTriangleFilled(bx0, by0, bx1, by0, bx1, by1, col)
	native.DrawTriangleFilled(bx0, by0, bx1, by1, bx0, by1, col)
	// Roof (triangle) on top.
	native.DrawTriangleFilled(hx-r, hy-r*0.1, hx+r, hy-r*0.1, hx, hy-r, col)
}

// overHomeButton reports whether screen point (mx,my) is over the home button.
func overHomeButton(mx, my float32, p cubePlacement) bool {
	return mx >= p.homeX-p.homeR && mx <= p.homeX+p.homeR && my >= p.homeY-p.homeR && my <= p.homeY+p.homeR
}
