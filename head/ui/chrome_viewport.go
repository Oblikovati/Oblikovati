//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/viewport"
	"oblikovati.org/model/clientgraphics"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// drawViewportPanel renders the active part's geometry through the Vulkan viewport and
// shows the result as an image filling the panel. An invisible drag button under the
// image captures mouse navigation (orbit/pan/zoom) and updates the session camera, so
// the scene — rebuilt from the model each frame (renderer.BuildDrawList) and projected
// with the navigated camera — always reflects current model and view state (ADR-0004).
func drawViewportPanel(win *native.Window, s *app.Session) {
	win.SetViewportNormalDebug(normalDebugOn || s.NormalDebug()) // Tools ▸ Normal Debug, or viewport.setNormalDebug
	if native.Begin("Viewport") {
		drawDocumentTabs(s)
		// View configuration (layout, new/close view) lives on the View ribbon tab's
		// Windows panel — see app.windowsViewCommands.
		// Per-document views can tile (split layouts); a single view takes the classic
		// full-panel path. Each visible tile renders its own view's camera into its own
		// offscreen slot, so tiles show distinct cameras simultaneously.
		if rects, active := planTiles(s); len(rects) > 1 {
			drawTiledViewports(win, s, rects, active)
		} else {
			drawSingleViewport(win, s)
		}
		viewCubeContextMenu(s)       // the ViewCube right-click menu (opened from a tile/single path)
		drawViewCubeOptions(s)       // the ViewCube Options window (toggled from that menu)
		serviceScreenshot(win, s)    // Tools ▸ Save Viewport PNG, after the viewport rendered
		serviceSaveThumbnail(win, s) // the save policy's sidecar capture (M03-F09)
	}
	native.End()
}

// drawSingleViewport renders the active view filling the whole panel (slot 0) — the
// classic single-view path: navigation, picking, overlays and client graphics.
func drawSingleViewport(win *native.Window, s *app.Session) {
	w, h := native.ContentRegionAvail()
	pw, ph := clampDim(w), clampDim(h)

	// Reserve the region with an input-capturing button, then read navigation from it.
	cx, cy := native.GetCursorPos()
	native.InvisibleButton("##viewport-nav", float32(pw), float32(ph))
	if native.IsItemClicked(native.MouseRight) {
		openMarkingMenu() // the radial marking menu (M05-F12)
	}
	bx, by := native.ItemRectMin()

	cam := s.Camera()
	cam.Width, cam.Height = pw, ph
	// ViewCube: when the cursor is over it, it owns the click (no orbit/pick) and a click
	// snaps / homes / opens the menu. Single view is already the active view (no activate).
	p := placeViewCube(bx, by, pw, ph, s.CubeSize(), s.CubeCorner())
	hit := viewCubeHover(s, p, cam)
	viewCubeClick(s, hit, pw, ph, nil)

	cam, hovered := updateViewportCamera(s, pw, ph, hit.overCube)
	list, bodyCount, sketchPlane, dims := viewportDrawList(s, cam, hovered)
	list, gfxLabels := clientGraphicsOverlay(s, cam, list)
	list = gizmoOverlays(s, cam, list)
	renderViewportImage(win, s, 0, cam, list, bodyCount, pw, ph, cx, cy)
	drawViewportOverlays(s, cam, sketchPlane, dims, gfxLabels, cx, cy, ph)
	if s.ShowViewCube() {
		drawViewCube(cam, s.CubeOrientation(), p, hit.region, hit.homeHit, s.ShowCompass(), s.InactiveOpacity())
	}
}

// planTiles returns the tile rectangles for the active document's view layout and the
// index of the active (focused) tile. It returns nil when there is no active document or
// the layout resolves to a single view (the caller then takes the single-view path).
func planTiles(s *app.Session) (rects []TileRect, active int) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, 0
	}
	vs := d.Views()
	w, h := native.ContentRegionAvail()
	sx, sy := vs.Split()
	rects = tileRects(vs.Layout(), w, h, vs.Count(), sx, sy)
	if len(rects) <= 1 {
		return nil, 0
	}
	active = vs.ActiveIndex()
	if active >= len(rects) {
		active = 0
	}
	return rects, active
}

// drawTiledViewports lays out one tile per visible view from the panel content origin,
// each rendering its own view camera into its own slot. The focused tile is interactive
// (navigation, picking, overlays); the others render their cameras and accept a click or
// drag to become the active view.
func drawTiledViewports(win *native.Window, s *app.Session, rects []TileRect, active int) {
	w, h := native.ContentRegionAvail()
	ox, oy := native.GetCursorPos()
	for i, r := range rects {
		drawViewTile(win, s, i, r, ox, oy, i == active)
	}
	drawSplitters(s, ox, oy, w, h, len(rects))
}

// drawSplitters places draggable handles in the gutters between tiles so the user can
// resize the views. Dragging updates the document's split fractions (Views.SetSplit),
// which re-tile next frame. A vertical handle adjusts the left|right divider; a horizontal
// handle adjusts the top/bottom divider (full width for the quad/two-V layout, only the
// right column for the three-up layout).
func drawSplitters(s *app.Session, ox, oy, w, h float32, tiles int) {
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	vs := d.Views()
	layout := vs.Layout()
	sx, sy := vs.Split()
	g := float32(tileGutter)
	vx, vy := w*sx, h*sy

	hasVert := tiles >= 3 || (tiles == 2 && layout != types.LayoutTwoV)
	hasHorz := tiles >= 3 || (tiles == 2 && layout == types.LayoutTwoV)

	if hasVert {
		if dx, _, dragging := dragSplit("##split-v", ox+vx-g/2, oy, g, h); dragging {
			vs.SetSplit(sx+dx/w, sy)
		}
	}
	if hasHorz {
		hx, hw := float32(0), w
		if tiles == 3 { // only the right column is split top/bottom
			hx, hw = vx+g/2, w-(vx+g/2)
		}
		if _, dy, dragging := dragSplit("##split-h", ox+hx, oy+vy-g/2, hw, g); dragging {
			vs.SetSplit(sx, sy+dy/h)
		}
	}
}

// dragSplit draws an invisible splitter handle over a gutter rect (at panel-cursor px,py,
// size pw×ph) and reports the mouse delta while it is being dragged.
func dragSplit(id string, px, py, pw, ph float32) (dx, dy float32, dragging bool) {
	native.SetCursorPos(px, py)
	native.InvisibleButton(id, pw, ph)
	if native.IsItemActive() {
		mdx, mdy := native.MouseDelta()
		return mdx, mdy, true
	}
	return 0, 0, false
}

// drawViewTile renders view i into tile slot i. tx,ty is the tile's top-left in the
// panel's cursor space (so the rendered image is drawn back exactly over the tile's
// input button).
func drawViewTile(win *native.Window, s *app.Session, i int, r TileRect, ox, oy float32, isActive bool) {
	tx, ty := ox+r.X, oy+r.Y
	pw, ph := clampDim(r.W), clampDim(r.H)
	native.SetCursorPos(tx, ty)
	native.InvisibleButton("##tile-"+strconv.Itoa(i), float32(pw), float32(ph))
	bx, by := native.ItemRectMin() // tile's screen rect, for the active-tile border

	cam, ok := sizedViewCamera(s, i, pw, ph)
	if !ok {
		return
	}

	// ViewCube owns the cursor when hovered (no orbit/pick); a click snaps this tile's view.
	p := placeViewCube(bx, by, pw, ph, s.CubeSize(), s.CubeCorner())
	hit := viewCubeHover(s, p, cam)
	isActive = tileInput(s, i, hit, &cam, pw, ph, isActive)
	if c, ok := sizedViewCamera(s, i, pw, ph); ok { // re-read after nav/animation/snap
		cam = c
	}

	if isActive {
		if !hit.overCube {
			handleViewportClick(s)
		}
		renderActiveTile(win, s, i, cam, pw, ph, tx, ty, bx, by)
	} else {
		bl := baseDrawList(s, cam)
		renderViewportImage(win, s, i, cam, bl, len(bl.Items), pw, ph, tx, ty)
	}
	if s.ShowViewCube() {
		drawViewCube(cam, s.CubeOrientation(), p, hit.region, hit.homeHit, s.ShowCompass(), s.InactiveOpacity())
	}
}

// sizedViewCamera returns tile i's camera sized to the tile (pw×ph), or ok=false if there
// is no such view.
func sizedViewCamera(s *app.Session, i, pw, ph int) (scene.Camera, bool) {
	c, ok := s.ViewCameraAt(i)
	if ok {
		c.Width, c.Height = pw, ph
	}
	return c, ok
}

// tileInput processes this frame's input for tile i — advancing a running tween, handling a
// ViewCube click, or navigating — and returns whether the tile is the active view afterward.
func tileInput(s *app.Session, i int, hit viewCubeHit, cam *scene.Camera, pw, ph int, isActive bool) bool {
	switch {
	case isActive && s.CameraAnimating():
		s.TickCameraAnimation(float64(native.DeltaTime())) // advance a running snap/sketch tween
	case hit.overCube:
		viewCubeClick(s, hit, pw, ph, func() { _ = s.ActivateView(i); isActive = true })
	default:
		isActive = tileNavigate(s, i, cam, isActive)
	}
	return isActive
}

// tileNavigate applies hover/drag/wheel to tile i's camera; on a non-active tile a real
// manipulation (drag/wheel) makes it the active view. Returns the (possibly updated) active.
func tileNavigate(s *app.Session, i int, cam *scene.Camera, isActive bool) bool {
	nav := readNavInput(isPlacingTool(s) && isActive)
	if nav.Hovered && navInteracted(nav) && !isActive {
		_ = s.ActivateView(i)
		isActive = true
	}
	if nav.Hovered || nav.Active {
		*cam = ApplyNavigation(*cam, nav)
		s.SetViewCameraAt(i, *cam)
	}
	return isActive
}

// renderActiveTile renders the focused tile's scene with overlays and the active-tile border.
func renderActiveTile(win *native.Window, s *app.Session, i int, cam scene.Camera, pw, ph int, tx, ty, bx, by float32) {
	hovered := hoveredPlane(s)
	list, bodyCount, sketchPlane, dims := viewportDrawList(s, cam, hovered)
	list, gfxLabels := clientGraphicsOverlay(s, cam, list)
	renderViewportImage(win, s, i, cam, list, bodyCount, pw, ph, tx, ty)
	drawViewportOverlays(s, cam, sketchPlane, dims, gfxLabels, tx, ty, ph)
	drawActiveTileBorder(bx, by, float32(pw), float32(ph))
}

// drawActiveTileBorder strokes a dark border just inside the active tile's screen rect so
// the user can tell which view is focused (drawn after the image, so it sits on top).
func drawActiveTileBorder(x, y, w, h float32) {
	const thickness = 2
	c := activeViewportBorderColor // themed (types.TokenViewportActiveBorder)
	x0, y0, x1, y1 := x+1, y+1, x+w-1, y+h-1
	native.DrawLine(x0, y0, x1, y0, c, thickness)
	native.DrawLine(x1, y0, x1, y1, c, thickness)
	native.DrawLine(x1, y1, x0, y1, c, thickness)
	native.DrawLine(x0, y1, x0, y0, c, thickness)
}

// navInteracted reports whether this frame's input is an active manipulation (a drag or a
// wheel), as opposed to mere hovering — the signal to claim a tile as the active view.
func navInteracted(n NavInput) bool { return n.Active || n.Wheel != 0 }

// viewportDrawList builds the frame's draw list and returns it with bodyCount — the number of
// leading body items (everything after is overlays), so the instanced path can rebuild the bodies
// from local per-component meshes and treat the overlay tail as a single identity instance.
func viewportDrawList(s *app.Session, cam scene.Camera, hovered *feature.WorkPlane) (renderer.DrawList, int, sketch.Plane, []app.DimensionView) {
	list := baseDrawList(s, cam)
	bodyCount := len(list.Items)
	if s.InSketch() {
		list, sketchPlane, dims := sketchOverlays(s, cam, list)
		return list, bodyCount, sketchPlane, dims
	}
	return modelOverlays(s, cam, hovered, list), bodyCount, sketch.Plane{}, nil
}

// baseDrawList is the model geometry (styled) with the current selection highlighted, with
// no environment overlays — what a passive (non-focused) tile shows for its view's camera.
func baseDrawList(s *app.Session, cam scene.Camera) renderer.DrawList {
	list := cachedBodyDrawList(s, cam) // a per-frame copy of the cached tessellation (see viewport_cache.go)
	return highlightSelection(list, s.Selection().First(), activeBodies(s))
}

func drawViewportOverlays(s *app.Session, cam scene.Camera, sketchPlane sketch.Plane, dims []app.DimensionView, labels []clientgraphics.Label, cx, cy float32, ph int) {
	ox, oy := native.ItemRectMin()
	drawAxisGizmo(cam, ox, oy, ph)
	drawClientGraphicsLabels(cx, cy, cam, labels)
	drawMiniToolbars(s, cam, ox, oy) // in-canvas mini-toolbars (M05-F07)
	if s.InSketch() && len(dims) > 0 {
		if d := drawDimensionLabels(cx, cy, cam, sketchPlane, dims); d != nil {
			s.BeginEditDimension(d) // double-clicked a dimension's value
		}
	}
}

// updateViewportCamera sizes the camera to the panel and either advances the active camera
// tween (ignoring user input, e.g. while entering/exiting a sketch) or applies this frame's
// navigation and resolves click/hover so the picker hit-tests against the current view. It
// returns the camera to render with and the work plane under the cursor (nil while animating).
func updateViewportCamera(s *app.Session, pw, ph int, overCube bool) (scene.Camera, *feature.WorkPlane) {
	cam := s.Camera()
	cam.Width, cam.Height = pw, ph
	if s.DriveAnimating() {
		s.TickDriveAnimation(float64(native.DeltaTime())) // advance a running joint-drive playback (M12-F03)
	}
	if s.CameraAnimating() {
		s.SetCamera(cam)
		s.TickCameraAnimation(float64(native.DeltaTime()))
		cam = s.Camera()
		cam.Width, cam.Height = pw, ph
		return cam, nil
	}
	if overCube {
		return cam, nil // the ViewCube owns the cursor this frame: no orbit, no pick
	}
	gizmoActive := s.TriadDragging() || s.ManipulatorDragging()
	cam = ApplyNavigation(cam, readNavInput(isPlacingTool(s) || gizmoActive))
	s.SetCamera(cam)
	// The triad/manipulators own the pointer when hovered or mid-drag (M05-F13);
	// picking and tool clicks stand down for the frame.
	if !routeGizmoInput(s, cam) {
		handleViewportClick(s)
	}
	return cam, hoveredPlane(s)
}

// frameMeshAndInstances builds the geometry the viewport draws: the instanced path (ADR-0038)
// rebuilds the bodies from per-component LOCAL meshes (deduped, one per unique component) plus the
// overlay/ground tail as one identity instance, returning the merged mesh + per-instance matrices +
// draw records. It falls back to a single legacy flatten of the whole list (nil mats/recs) when
// instancing does not apply — mesh-color debug mode (its own builder) or no keyable geometry.
func frameMeshAndInstances(s *app.Session, cam scene.Camera, list renderer.DrawList, bodyCount int, ground []renderer.DrawItem) (viewport.Mesh, []float32, []int32) {
	if bodyCount < 0 || bodyCount > len(list.Items) {
		bodyCount = len(list.Items)
	}
	if on, _ := s.MeshColors(); !on {
		overlay := renderer.DrawList{Items: append(append([]renderer.DrawItem(nil), list.Items[bodyCount:]...), ground...)}
		decorate := func(l renderer.DrawList) renderer.DrawList {
			return highlightSelection(l, s.Selection().First(), activeBodies(s))
		}
		if m, mats, recs, ok := buildInstancedFrame(s.VisibleInstances(), overlay, cam, s.SurfaceLookup(), s.VisualStyle(), decorate); ok {
			return m, mats, recs
		}
	}
	list.Items = append(list.Items, ground...) // legacy: one flatten of the whole (world-space) list
	return viewport.Flatten(list), nil, nil
}

// renderViewportImage flattens the draw list, renders it into the window's offscreen target
// with the camera's view-projection, and blits the resulting texture back over the
// input-capturing button at (cx,cy) so the panel shows the rendered scene.
func renderViewportImage(win *native.Window, s *app.Session, slot int, cam scene.Camera, list renderer.DrawList, bodyCount, pw, ph int, cx, cy float32) {
	// Fit the shadow frustum to the model (before adding the ground), then drop in the ground
	// plane so object shadows have a surface to land on. Bounds come from the world-space list
	// (its body items are correct for shadow framing even though the instanced path redraws the
	// bodies from local meshes).
	mn, mx, hasGeom := viewport.DrawListBounds(list)
	var ground []renderer.DrawItem
	if hasGeom && wantGround(s) {
		ground = []renderer.DrawItem{groundPlaneItem(mn, mx, renderer.PassSetFor(s.VisualStyle()).Faces)}
	}
	m, mats, recs := frameMeshAndInstances(s, cam, list, bodyCount, ground)
	mvp := renderer.ViewProjection(cam, viewportNear, viewportFar)
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}
	win.SetViewportLighting(viewport.PackLighting(s.SceneLighting()))
	applyEnvironment(win, s.Environment())
	applySkybox(win, s.Environment(), mvp)
	applyShadow(win, s, mn, mx, hasGeom)
	win.RenderViewport(slot, pw, ph, mvp[:], eye,
		m.TriVerts, m.TriVCount, m.TriIndices,
		m.OccVerts, m.OccVCount, m.OccIndices,
		m.LineVerts, m.LineVCount, m.LineIndices,
		m.HidVerts, m.HidVCount, m.HidIndices,
		m.TopTriVerts, m.TopTriVCount, m.TopTriIndices,
		m.TopLineVerts, m.TopLineVCount, m.TopLineIndices,
		m.TriBiasFirst, s.ActiveSectionClip(), // section-plane clip (M12-F04)
		mats, recs) // instanced draw (ADR-0038); nil mats/recs ⇒ legacy one-identity-instance path
	if tex := win.ViewportTexture(slot); tex != 0 {
		native.SetCursorPos(cx, cy) // draw the image back over the invisible button
		native.Image(tex, float32(pw), float32(ph))
	}
}

// sketchOverlays appends the sketch-environment overlays (grid, entities, dimensions, snap and
// tool previews) to list, returning the augmented list plus the sketch plane and dimension
// views the caller needs for the on-screen dimension labels.
func sketchOverlays(s *app.Session, cam scene.Camera, list renderer.DrawList) (renderer.DrawList, sketch.Plane, []app.DimensionView) {
	plane := s.ActiveSketch().Plane()
	if g := s.Grid(); g.Visible {
		list.Items = append(gridOverlay(plane, g.SpacingModel(), g.MajorEvery), list.Items...)
	}
	list.Items = append(list.Items, sketchOverlay(s.ActiveSketch(), s.IsSelectedEntity, hoverCandidate(s))...)
	dims := s.SketchDimensions()
	list.Items = append(list.Items, dimensionLines(plane, dims)...)
	if item, ok := pointsOverlay(plane, s.ActiveSketch(), pointMarkerPixels*cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, item)
	}
	if item, ok := toolPreview(s); ok {
		list.Items = append(list.Items, item)
	}
	if item, ok := inferenceGlyphs(s, plane, glyphPixels*cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, item)
	}
	if item, ok := snapMarker(s, plane, cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, item)
	}
	return list, plane, dims
}

// modelOverlays appends the 3D-model overlays (work planes, part sketches, selected edges, and
// the extrude / active-tool previews) to list.
func modelOverlays(s *app.Session, cam scene.Camera, hovered *feature.WorkPlane, list renderer.DrawList) renderer.DrawList {
	wg, hasWG := s.ActiveWorkGeometry() // a part OR an assembly's origin frame + datums (#769 parity)
	hidden := s.EditScopeHides          // hide datums created after the node being edited (issue #132)
	vis := s.ObjectVisibility()         // View ▸ Object visibility (M05-F12): hidden kinds neither draw nor pick
	if hasWG && vis.WorkPlanes {
		list.Items = append(list.Items, planesOverlay(wg.WorkPlanes(), s.SelectedWorkPlane(), hovered, hidden)...)
	}
	if hasWG && vis.WorkAxes {
		list.Items = append(list.Items, axesOverlay(wg.WorkAxes(), selectedWorkAxis(s), hidden)...)
	}
	if vis.Sketches {
		list.Items = append(list.Items, partSketchOverlays(s)...)
		list.Items = append(list.Items, partSketchPoints(s, pointMarkerPixels*cam.WorldPerPixel())...)
		list.Items = append(list.Items, sketch3DOverlays(s, pointMarkerPixels*cam.WorldPerPixel())...)
	}
	list.Items = append(list.Items, selectedEdgeOverlay(s)...)
	list.Items = append(list.Items, threadOverlay(s)...)
	list.Items = append(list.Items, toolHoverHighlight(s)...)
	list.Items = append(list.Items, toolSelectedHighlight(s)...)
	list.Items = append(list.Items, revolveCenterlineHighlight(s)...)
	list.Items = append(list.Items, activeToolPreviewItems(s)...)
	return list
}

// gizmoOverlays appends the triad and manipulator-handle geometry (M05-F13).
func gizmoOverlays(s *app.Session, cam scene.Camera, list renderer.DrawList) renderer.DrawList {
	list = triadOverlay(s, cam, list)
	return manipulatorOverlay(s, cam, list)
}

// isPlacingTool reports whether the active tool consumes plane-point clicks (a sketch
// geometry tool), so the viewport should route left-clicks to it instead of orbiting.
func isPlacingTool(s *app.Session) bool {
	ti := s.ActiveTool()
	if ti == nil {
		return false
	}
	_, ok := ti.Tool().(app.PlaneClickTool)
	return ok
}

// handleViewportClick feeds a left-click on the viewport image to the session in
// viewport-local pixels (the camera was sized to the panel) — placing sketch geometry,
// selecting a sketch entity, or picking a face/plane. Shift extends the selection.
func handleViewportClick(s *app.Session) {
	if !native.IsItemClicked(native.MouseLeft) {
		return
	}
	x, y := viewportCursor()
	e := app.PointerEvent{X: x, Y: y, Button: app.LeftButton}
	if native.KeyShift() {
		e.Mods |= app.ShiftMod
	}
	if native.KeyCtrl() {
		e.Mods |= app.CtrlMod // Ctrl+click adds a region to a multi-region extrude
	}
	s.Pointer(e)
}

// hoveredPlane returns the origin work plane under the cursor (the front-most hit), or
// nil — used to highlight the plane the user is about to pick. Only meaningful while
// the viewport is hovered and not editing a sketch.
func hoveredPlane(s *app.Session) *feature.WorkPlane {
	if !native.IsItemHovered() || s.InSketch() {
		return nil
	}
	x, y := viewportCursor()
	sel, ok := s.PickAt(x, y, app.NewSelectionFilter())
	if !ok {
		return nil
	}
	if wp, isPlane := sel.(app.WorkPlaneHandle); isPlane {
		return wp.Plane
	}
	return nil
}

// viewportCursor returns the cursor position in viewport-local pixels (relative to the
// viewport image's top-left), valid right after the viewport's InvisibleButton.
func viewportCursor() (float64, float64) {
	mx, my := native.MousePos()
	ox, oy := native.ItemRectMin()
	return float64(mx - ox), float64(my - oy)
}

// hoverCandidate returns the sketch entity under the cursor that the active constraint/
// dimension tool would accept (highlighted green to show what is selectable), or nil.
func hoverCandidate(s *app.Session) sketch.Entity {
	if !native.IsItemHovered() || s.ActiveTool() == nil {
		return nil
	}
	cx, cy := viewportCursor()
	ent, ok := s.HoverCandidate(cx, cy)
	if !ok {
		return nil
	}
	return ent
}

// snapMarker returns the snap glyph (square/triangle/cross) under the cursor when a
// sketch tool is active and the viewport is hovered, so the user sees the otherwise-1px
// snap point. worldPerPixel sizes the glyph screen-constant.
func snapMarker(s *app.Session, plane sketch.Plane, worldPerPixel float64) (renderer.DrawItem, bool) {
	if s.ActiveTool() == nil || !native.IsItemHovered() {
		return renderer.DrawItem{}, false
	}
	cx, cy := viewportCursor()
	r, ok := s.SnapAt(cx, cy)
	if !ok {
		return renderer.DrawItem{}, false
	}
	return snapGlyph(plane, r, snapGlyphPixels*worldPerPixel)
}

// toolPreview returns the active geometry tool's rubber-band preview at the cursor (the
// provisional shape from the placed clicks through the current mouse position).
func toolPreview(s *app.Session) (renderer.DrawItem, bool) {
	if !native.IsItemHovered() || s.ActiveTool() == nil {
		return renderer.DrawItem{}, false
	}
	cx, cy := viewportCursor()
	cur, ok := s.CursorSketchPoint(cx, cy)
	if !ok {
		return renderer.DrawItem{}, false
	}
	pts, closed := s.ActiveToolPreview(cur)
	if len(pts) == 0 {
		return renderer.DrawItem{}, false
	}
	acc := &segAccum{}
	acc.polyline(s.ActiveSketch().Plane(), pts, closed)
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: previewColor}, true
}

const (
	viewportNear = 0.1
	viewportFar  = 5000.0
)

// readNavInput snapshots this frame's viewport pointer state from the native layer for
// the pure ApplyNavigation mapping (see navigate.go). It must be called right after the
// viewport's InvisibleButton, so IsItemActive/Hovered refer to it. While a sketch tool
// is placing geometry, the left button is withheld so a click does not also orbit.
func readNavInput(placing bool) NavInput {
	dx, dy := native.MouseDelta()
	return NavInput{
		Hovered: native.IsItemHovered(),
		Active:  native.IsItemActive(),
		Wheel:   native.MouseWheel(),
		DX:      dx,
		DY:      dy,
		Middle:  native.MouseDown(native.MouseMiddle),
		Left:    !placing && native.MouseDown(native.MouseLeft),
		Shift:   native.KeyShift(),
	}
}
