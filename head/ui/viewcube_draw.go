//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
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
	homeMenuItems(s)
	native.Separator()
	projectionMenuItems(s)
	native.Separator()
	frontMenuItems(s)
	native.Separator()
	viewCubeToggleItems(s)
	native.EndPopup()
}

// homeMenuItems renders Go Home + the two "Set Current View as Home" rows.
func homeMenuItems(s *app.Session) {
	if native.MenuItem("Go Home") {
		s.GoHome()
	}
	if native.MenuItem("Set Current View as Home (Fixed Distance)") {
		s.SetActiveViewHome(false)
	}
	if native.MenuItem("Set Current View as Home (Fit to View)") {
		s.SetActiveViewHome(true)
	}
}

// frontMenuItems renders the Set/Reset Front rows.
func frontMenuItems(s *app.Session) {
	if native.MenuItem("Set Current View as Front") {
		s.SetActiveViewAsFront()
	}
	if native.MenuItem("Reset Front") {
		s.ResetFront()
	}
}

// viewCubeToggleItems renders Lock to Selection, Show Compass, and Options….
func viewCubeToggleItems(s *app.Session) {
	if native.MenuItem(projItemLabel("Lock to Current Selection", s.LockToSelection())) {
		s.SetLockToSelection(!s.LockToSelection())
	}
	if native.MenuItem(projItemLabel("Show Compass", s.ShowCompass())) {
		s.SetShowCompass(!s.ShowCompass())
	}
	if native.MenuItem("Options...") {
		showViewCubeOptions = true
	}
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
	const rc, segs = 1.95, 48 // ring radius (cube units) — clears the cube's iso reach (≈1.74) with a gap
	var px, py float32
	for i := 0; i <= segs; i++ {
		t := float64(i) / segs * 2 * stdmath.Pi
		c := project(math.V3(rc*stdmath.Cos(t), rc*stdmath.Sin(t), -1), right, up, fwd, r)
		x, y := cx+c.sx, cy+c.sy
		if i > 0 {
			native.DrawLine(px, py, x, y, viewCubeCompassColor, 1.6)
		}
		px, py = x, y
	}
	// The four cardinals, just outside the ring (N=+Y/BACK, E=+X, S=−Y/FRONT, W=−X), matching the
	// reference's bold N/E/S/W rose.
	const lr = rc * 1.13 // cardinals just outside the ring
	for _, card := range []struct {
		v math.Vector3
		s string
	}{
		{math.V3(0, lr, -1), "N"}, {math.V3(lr, 0, -1), "E"},
		{math.V3(0, -lr, -1), "S"}, {math.V3(-lr, 0, -1), "W"},
	} {
		c := project(card.v, right, up, fwd, r)
		native.DrawText(cx+c.sx-3.5, cy+c.sy-7, card.s, viewCubeCompassColor)
	}
}

// projectionMenuItems renders the three projection-mode rows (radio-marked) for the active
// view's current mode.
func projectionMenuItems(s *app.Session) {
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
	viewCubeMarginRatio  = 2.4  // cube-center inset from the corner — fits the compass ring + cardinals
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

// viewCubeHit is the per-frame ViewCube hit state for a viewport.
type viewCubeHit struct {
	region   *Region
	homeHit  bool
	arrow    cubeArrowHit // a roll / adjacent-face arrow under the cursor (#914)
	overCube bool         // cursor is over the cube/home/arrows — caller suppresses orbit/pick
}

// viewCubeHover hit-tests the ViewCube at placement p for camera cam (read-only). The arrow
// widgets are tested first: a cursor on an arrow is an arrow hit, not a cube face/edge/corner.
func viewCubeHover(s *app.Session, p cubePlacement, cam scene.Camera) viewCubeHit {
	var h viewCubeHit
	if s.ShowViewCube() && native.IsItemHovered() {
		mx, my := native.MousePos()
		h.arrow = hitViewCubeArrow(mx, my, p)
		if h.arrow.kind == arrowNone {
			h.region = HitTest(mx-p.cx, my-p.cy, p.r, cam, s.CubeOrientation())
		}
		h.homeHit = overHomeButton(mx, my, p)
	}
	h.overCube = h.region != nil || h.homeHit || h.arrow.kind != arrowNone
	return h
}

// viewCubeClick acts on a click over the cube: left snaps the view (or Go Home), right opens
// the menu. onActivate (may be nil) makes the owning view active first; pw/ph size the snap
// camera. A no-op when the cursor is not over the cube.
func viewCubeClick(s *app.Session, h viewCubeHit, pw, ph int, onActivate func()) {
	if !h.overCube {
		return
	}
	left := native.IsItemClicked(native.MouseLeft)
	right := h.region != nil && native.IsItemClicked(native.MouseRight)
	if (left || right) && onActivate != nil {
		onActivate()
	}
	if right {
		native.OpenPopup(viewCubeMenuID)
		return
	}
	if !left {
		return
	}
	switch {
	case h.arrow.kind != arrowNone:
		applyViewCubeArrow(s, h.arrow, pw, ph) // roll / step to the adjacent face (#914)
	case h.homeHit:
		s.GoHome()
	case h.region != nil:
		snapToCubeRegion(s, h.region, pw, ph)
	}
}

// snapToCubeRegion animates the view to look at the pivot from a clicked face/edge/corner region.
func snapToCubeRegion(s *app.Session, region *Region, pw, ph int) {
	start := s.Camera()
	start.Width, start.Height = pw, ph
	s.SetCamera(start) // sync the tween's start to this view
	s.AnimateCameraTo(region.SnapCamera(start, s.ViewCubePivot(), s.CubeOrientation()), viewCubeSnapSecs)
}

// ViewCube colors. Faces are a light translucent panel; the hovered region's faces tint to
// the accent. (Theming via tokens is a Phase-C follow-up.)
// ViewCube palette, sampled from the reference: light blue-gray faces, a muted slate-blue hover,
// subtle gray grid lines, medium-gray labels (not black), and blue-gray compass cardinals.
var (
	viewCubeFaceColor    = [4]float32{0.85, 0.87, 0.91, 1.0} // light blue-gray face (≈217,222,232)
	viewCubeHoverColor   = [4]float32{0.42, 0.50, 0.64, 1.0} // muted slate blue (≈102,120,153)
	viewCubeEdgeColor    = [4]float32{0.64, 0.67, 0.71, 1.0} // subtle grid line
	viewCubeTextColor    = [4]float32{0.40, 0.43, 0.47, 1.0} // medium gray label (≈103,109,117)
	viewCubeHomeColor    = [4]float32{0.62, 0.66, 0.72, 0.95}
	viewCubeCompassColor = [4]float32{0.44, 0.48, 0.54, 0.95} // blue-gray cardinals
)

// faceShade brightens the top face and dims the bottom, like the reference's lit cube, for the 3D
// look (the cube's local +Z is TOP). Applied to the base face color, not the hover highlight.
func faceShade(r Region) float32 {
	switch r.Z {
	case 1: // TOP
		return 1.0
	case -1: // BOTTOM
		return 0.82
	default: // sides
		return 0.91
	}
}

// drawViewCube paints the navigation cube centered at screen (cx,cy) for the camera, with
// the hovered region (if any) tinted and the home button highlighted when homeHovered.
// Drawn after the tile image so it sits on top; uses screen coordinates (ImGui draw list).
func drawViewCube(cam scene.Camera, o doc.CubeOrient, p cubePlacement, hovered *Region, homeHovered, compass bool, opacity float32, arrow cubeArrowHit) {
	if compass {
		drawCompass(cam, o, p.cx, p.cy, p.r) // under the cube faces
	}
	drawViewCubeArrows(p, arrow) // adjacent-face + roll arrows around the cube (#914)
	right, up, fwd := cubeBasis(cam, o)
	for _, f := range visibleFaces(cam, o, p.r) {
		drawFaceCells(f, right, up, fwd, p, hovered, opacity)
		// Label painted IN the face plane (projected with the cube), not screen-aligned.
		for _, s := range faceLabelSegments(f, cam, o, p.r) {
			native.DrawLine(p.cx+s[0], p.cy+s[1], p.cx+s[2], p.cy+s[3], viewCubeTextColor, viewCubeLabelW)
		}
	}
	drawHomeButton(p.homeX, p.homeY, p.homeR, homeHovered)
}

// drawFaceCells paints a face's 3×3 Rubik grid: each cell is a filled quad with a grid border,
// and only the cell matching the hovered region is highlighted (so a face hover lights the centre
// cell, an edge hover one edge cell, a corner hover one corner cell).
func drawFaceCells(f cubeFace, right, up, fwd math.Vector3, p cubePlacement, hovered *Region, opacity float32) {
	shade := faceShade(f.region)
	for _, cell := range faceCells(faceDefFor(f.region)) {
		col := viewCubeFaceColor
		if hovered != nil && sameRegion(cell.region, *hovered) {
			col = viewCubeHoverColor
		} else {
			col[0], col[1], col[2] = col[0]*shade, col[1]*shade, col[2]*shade // lit cube look
			col[3] = opacity                                                  // inactive cells honor the user's opacity preference
		}
		var c [4]cubeCorner
		for k := range cell.quad {
			c[k] = project(cell.quad[k], right, up, fwd, p.r)
		}
		x0, y0 := p.cx+c[0].sx, p.cy+c[0].sy
		x1, y1 := p.cx+c[1].sx, p.cy+c[1].sy
		x2, y2 := p.cx+c[2].sx, p.cy+c[2].sy
		x3, y3 := p.cx+c[3].sx, p.cy+c[3].sy
		native.DrawQuadFilled(x0, y0, x1, y1, x2, y2, x3, y3, col)
		native.DrawLine(x0, y0, x1, y1, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x1, y1, x2, y2, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x2, y2, x3, y3, viewCubeEdgeColor, viewCubeEdgeW)
		native.DrawLine(x3, y3, x0, y0, viewCubeEdgeColor, viewCubeEdgeW)
	}
}

// faceDefFor returns the cube-face definition for a face region (its 3×3 grid is generated from it).
func faceDefFor(r Region) faceDef {
	for _, d := range cubeFaceDefs {
		n := d.normal()
		if int(n.X) == r.X && int(n.Y) == r.Y && int(n.Z) == r.Z {
			return d
		}
	}
	return cubeFaceDefs[0]
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
