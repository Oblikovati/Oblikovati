//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"strconv"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/clientgraphics"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/viewport"
	"oblikovati.org/kernel/topo"
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
	// The viewport draws icon textures (the Navigation Bar, #913 N25), so the icon cache must be bound
	// to THIS window. DrawChrome binds it lazily, but the in-window viewportFrame test path calls us
	// directly — without that, a stale cache from a destroyed window would feed ImageButton invalid
	// textures and crash Vulkan at EndFrame. Rebind when absent or bound to a different window.
	if icons == nil || icons.win != win {
		icons = newIconCache(win)
	}
	win.SetViewportNormalDebug(normalDebugOn || s.NormalDebug()) // Tools ▸ Normal Debug, or viewport.setNormalDebug
	if native.Begin("Viewport") {
		drawDocumentTabs(s)
		// A drawing document shows a 2D sheet canvas, not the 3D viewport (M14-F01).
		if dc, err := app.ActiveDrawing(s); err == nil {
			drawSheetCanvas(s, dc)
			native.End()
			return
		}
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
	viewportHovered := native.IsItemHovered() // captured before later items shift IsItemHovered
	handleViewportRightClick(s)
	bx, by := native.ItemRectMin()

	cam := s.Camera()
	cam.Width, cam.Height = pw, ph
	// ViewCube: when the cursor is over it, it owns the click (no orbit/pick) and a click
	// snaps / homes / opens the menu. Single view is already the active view (no activate).
	p := placeViewCube(bx, by, pw, ph, s.CubeSize(), s.CubeCorner())
	hit := viewCubeHover(s, p, cam)
	viewCubeClick(s, hit, pw, ph, nil)

	t0 := frameClock()
	cam, hovered := updateViewportCamera(s, pw, ph, hit.overCube)
	sketchPlane, dims, gfxLabels, gfxImages := buildAndRenderScene(win, s, cam, hovered, pw, ph, cx, cy, t0)
	drawViewportOverlays(s, cam, sketchPlane, dims, gfxLabels, gfxImages, cx, cy, ph)
	drawViewportRubberBands(s, bx, by, cx, cy, pw, ph, viewportHovered)
	if s.ShowViewCube() {
		drawViewCube(cam, s.CubeOrientation(), p, hit.region, hit.homeHit, s.ShowCompass(), s.InactiveOpacity(), hit.arrow)
	}
}

// drawViewportRubberBands paints the on-image interaction overlays drawn after the scene: the
// box-select and Zoom Window rubber bands, the Free-Orbit ring, the navigation bar and
// SteeringWheels menu, and the 2D-sketch dynamic-input HUD (#790).
func drawViewportRubberBands(s *app.Session, bx, by, cx, cy float32, pw, ph int, viewportHovered bool) {
	drawBoxSelectRect(s, bx, by)                // the rubber-band selection rectangle, on top of the image
	drawZoomWindowRect(s, bx, by)               // the Zoom Window rubber band (#913 N16), if armed
	drawOrbitRing(bx, by, pw, ph)               // the Free-Orbit ring while F4 is held (#913 N5–N8)
	drawNavigationBar(s, bx, by, pw, ph)        // the floating nav-tool strip at the right edge (#913 N25)
	drawSteeringWheel(s, cx, cy)                // the on-cursor radial nav menu (#913 N26), if active
	handleSketchHUD(s, bx, by, viewportHovered) // the 2D-sketch dynamic-input HUD (#790)
}

// buildAndRenderScene assembles the single view's draw list (body instances + overlays), renders it,
// and records the per-phase frame timing (t0 brackets the preceding pick). It returns the sketch
// plane, dimensions and client-graphics labels the caller draws as 2D overlays on top.
func buildAndRenderScene(win *native.Window, s *app.Session, cam scene.Camera, hovered *feature.WorkPlane,
	pw, ph int, cx, cy float32, t0 time.Time,
) (sketch.Plane, []app.DimensionView, []clientgraphics.Label, []clientgraphics.ImageBillboard) {
	t1 := frameClock()
	list, bodyCount, sketchPlane, dims := viewportDrawList(s, cam, hovered)
	list, gfxLabels, gfxImages := clientGraphicsOverlay(s, cam, list)
	list = gizmoOverlays(s, cam, list)
	t2 := frameClock()
	renderViewportImage(win, s, 0, cam, list, bodyCount, pw, ph, cx, cy)
	frameTiming(t0, t1, t2, frameClock())
	return sketchPlane, dims, gfxLabels, gfxImages
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
		drawViewCube(cam, s.CubeOrientation(), p, hit.region, hit.homeHit, s.ShowCompass(), s.InactiveOpacity(), hit.arrow)
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
	nav := readNavInput(s)
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
	list, gfxLabels, gfxImages := clientGraphicsOverlay(s, cam, list)
	renderViewportImage(win, s, i, cam, list, bodyCount, pw, ph, tx, ty)
	drawViewportOverlays(s, cam, sketchPlane, dims, gfxLabels, gfxImages, tx, ty, ph)
	drawActiveTileBorder(bx, by, float32(pw), float32(ph))
}

// drawActiveTileBorder strokes a dark border just inside the active tile's screen rect so
// the user can tell which view is focused (drawn after the image, so it sits on top).
func drawActiveTileBorder(x, y, w, h float32) {
	const thickness = 2
	c := chromeTheme.activeViewportBorderColor // themed (types.TokenViewportActiveBorder)
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
	var list renderer.DrawList
	// The instanced path (ADR-0038) rebuilds the bodies from per-component local meshes, so the
	// world-body draw list is built ONLY for the mesh-color debug mode (its own builder + legacy
	// flatten) — otherwise a 10k-copy assembly would tessellate 10k world bodies here just to throw
	// them away. bodyCount stays 0 in the instanced case (everything appended is an overlay).
	if on, _ := s.MeshColors(); on {
		list = baseDrawList(s, cam)
	}
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
	list := cachedBodyDrawList(s, cam)                            // a per-frame copy of the cached tessellation (see viewport_cache.go)
	list.Items = append(list.Items, pointCloudOverlay(s, cam)...) // attached scans (M17-F06, #645)
	return highlightSelection(list, s.Selection().First(), activeBodies(s))
}

func drawViewportOverlays(s *app.Session, cam scene.Camera, sketchPlane sketch.Plane, dims []app.DimensionView, labels []clientgraphics.Label, images []clientgraphics.ImageBillboard, cx, cy float32, ph int) {
	ox, oy := native.ItemRectMin()
	drawAxisGizmo(cam, ox, oy, ph)
	drawClientGraphicsLabels(cx, cy, cam, labels)
	drawClientGraphicsImages(cx, cy, cam, images)
	drawMiniToolbars(s, cam, ox, oy) // in-canvas mini-toolbars (M05-F07)
	if s.InSketch() && len(dims) > 0 {
		if d := drawDimensionLabels(cx, cy, cam, sketchPlane, dims); d != nil {
			s.BeginEditDimension(d) // double-clicked a dimension's value
		}
	}
	if s.InSketch() {
		drawPlacementFieldBoxes(s, cam, ox, oy) // in-place dimension input (#2014)
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
	cam = ApplyNavigation(cam, readNavInput(s))
	s.SetCamera(cam)
	// The triad/manipulators own the pointer when hovered or mid-drag (M05-F13);
	// picking and tool clicks stand down for the frame.
	if !routeGizmoInput(s, cam) {
		handleViewportSelection(s) // box-select on an empty-space drag, else single-pick click
	}
	return cam, hoveredPlane(s)
}

// Per-frame scratch buffers reused across frames so a static orbit allocates nothing here (#1423).
// Safe because the render loop is single-threaded and each is consumed within the frame that fills it
// (the overlay is flattened into the atlas; sources only feeds the highlight decorator; eye is copied
// into the push constant by RenderViewport before it returns).
var (
	frameOverlayScratch []renderer.DrawItem
	frameSourcesScratch []*topo.Body
	frameEyeScratch     [3]float32
)

// frameOverlayList builds the overlay DrawList (body-tail items + ground) into the reused scratch
// slice, so an orbit doesn't reallocate it each frame (#1423).
func frameOverlayList(items, ground []renderer.DrawItem) renderer.DrawList {
	frameOverlayScratch = append(frameOverlayScratch[:0], items...)
	frameOverlayScratch = append(frameOverlayScratch, ground...)
	return renderer.DrawList{Items: frameOverlayScratch}
}

// frameSources collects the group source bodies into the reused scratch slice (#1423).
func frameSources(groups []app.InstanceGroup) []*topo.Body {
	frameSourcesScratch = frameSourcesScratch[:0]
	for _, g := range groups {
		frameSourcesScratch = append(frameSourcesScratch, g.Source)
	}
	return frameSourcesScratch
}

// frameEye packs the camera eye into the reused scratch array; RenderViewport copies it into the push
// constant synchronously, so handing back the shared slice is safe (#1423).
func frameEye(cam scene.Camera) []float32 {
	frameEyeScratch = [3]float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}
	return frameEyeScratch[:]
}

// frameMeshAndInstances builds the geometry the viewport draws: the instanced path (ADR-0038)
// rebuilds the bodies from per-component LOCAL meshes (deduped, one per unique component) plus the
// overlay/ground tail as one identity instance, returning the merged mesh + per-instance matrices +
// draw records. It falls back to a single legacy flatten of the whole list (nil mats/recs) when
// instancing does not apply — mesh-color debug mode (its own builder) or no keyable geometry.
func frameMeshAndInstances(s *app.Session, cam scene.Camera, list renderer.DrawList, bodyCount int, ground []renderer.DrawItem, groups, culled []app.InstanceGroup) (viewport.Mesh, []float32, []int32, uint64) {
	if bodyCount < 0 || bodyCount > len(list.Items) {
		bodyCount = len(list.Items)
	}
	// M16-F07: apply the active document's display-settings edge color before the (instanced or
	// world) body meshes are built; it is baked into the edge line items, so it also keys the
	// source-mesh cache (instancedSourceKey) and the body cache (bodyGeometryKey).
	renderer.SetEdgeColor(displayEdgeColor(s))
	if on, _ := s.MeshColors(); !on {
		// frameOverlayList/frameSources reuse scratch buffers (consumed within this frame), so a static
		// orbit rebuilds neither's backing array (#1423). Sources highlight against the group SOURCE
		// bodies (already in hand), NOT activeBodies(s) — the latter re-derives every occurrence each
		// frame (TransformBody), an O(occurrences) cost that defeats instancing on a big assembly.
		overlay := frameOverlayList(list.Items[bodyCount:], ground)
		sources := frameSources(groups)
		decorate := func(l renderer.DrawList) renderer.DrawList {
			return highlightSelection(l, s.Selection().First(), sources)
		}
		placedMesh, placedMeshKey, _ := cachedPlacedMesh(s) // retained placed-mesh lane (#1773)
		if m, mats, recs, key, ok := buildInstancedFrame(groups, culled, overlay, placedMesh, placedMeshKey, cam, s.SurfaceLookup(), s.VisualStyle(), decorate, instancedSourceKey(s)); ok {
			return m, mats, recs, geomUploadKey(key)
		}
	}
	// Legacy flatten: a fresh world-space mesh with no stable atlas key, so geomKey 0 ⇒ the native
	// renderer always re-uploads it (correct, just unoptimised — the instanced path is the hot one).
	// Placed mesh references have no instanced source, so add them to the world list here too so they
	// still show in mesh-color debug mode (#1775). Flatten routes the opaque tris correctly. This
	// re-flattens the mesh per frame, acceptable in this debug/fallback path (not the hot instanced one).
	list.Items = append(list.Items, ground...)
	list.Items = append(list.Items, s.MeshDrawItems()...)
	return viewport.Flatten(list), nil, nil, 0
}

// frameBounds is the model bounds for shadow + ground framing: the instance groups' transformed
// range boxes (no tessellation), unioned with the overlay tail's extent. In mesh-color mode the
// bodies are in the world-space list, so it scans that instead.
func frameBounds(s *app.Session, list renderer.DrawList, groups []app.InstanceGroup) (mn, mx [3]float32, ok bool) {
	if on, _ := s.MeshColors(); on {
		return viewport.DrawListBounds(list)
	}
	mn, mx, ok = instancedBounds(groups)
	if omn, omx, ook := viewport.DrawListBounds(list); ook { // include overlay (work-plane/sketch) extent
		if !ok {
			return omn, omx, true
		}
		for i := 0; i < 3; i++ {
			mn[i] = minF32(mn[i], omn[i])
			mx[i] = maxF32(mx[i], omx[i])
		}
	}
	return mn, mx, ok
}

func minF32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// renderViewportImage flattens the draw list, renders it into the window's offscreen target
// with the camera's view-projection, and blits the resulting texture back over the
// input-capturing button at (cx,cy) so the panel shows the rendered scene.
func renderViewportImage(win *native.Window, s *app.Session, slot int, cam scene.Camera, list renderer.DrawList, bodyCount, pw, ph int, cx, cy float32) {
	// Fit the shadow frustum to the model (before adding the ground), then drop in the ground
	// plane so object shadows have a surface to land on. With instancing, the bounds come from the
	// instance groups' transformed range boxes (no tessellation) rather than scanning a world-body
	// mesh, so framing a 10k-copy assembly is O(instances).
	groups := s.VisibleInstances()
	mn, mx, hasGeom := frameBounds(s, list, groups) // framing/shadow bounds cover the whole model
	var ground []renderer.DrawItem
	if hasGeom && wantGround(s) {
		ground = []renderer.DrawItem{groundPlaneItem(mn, mx, renderer.PassSetFor(s.VisualStyle()).Faces, displayGroundColor(s))}
	}
	tb := frameClock()
	// Draw only the instances inside the view frustum (M34-F1) — off-screen placements never reach
	// the GPU upload. The bounds above still use the full set so shadows/framing don't shift on orbit.
	// allGroups builds the retained vertex atlas (stable on orbit); culled drives the per-frame draws.
	m, mats, recs, geomKey := frameMeshAndInstances(s, cam, list, bodyCount, ground, groups, s.CulledInstances(cam))
	frameStats.buildNs = time.Since(tb).Nanoseconds()
	mvp := renderer.ViewProjection(cam, viewportNear, viewportFarPlane(s, cam, mn, mx, hasGeom))
	eye := frameEye(cam) // reused scratch; RenderViewport copies it into the push constant synchronously (#1423)
	win.SetViewportLighting(viewport.PackLighting(s.SceneLighting()))
	applyEnvironment(win, app.RenderEnvironment(s.Environment())) // app value -> renderer at the wall (B10 #1621)
	applySkybox(win, app.RenderEnvironment(s.Environment()), mvp)
	applyShadow(win, s, mn, mx, hasGeom)
	uploadPointClouds(win, s) // retained GL-points buffer, skipped on orbit (#645)
	tg := frameClock()
	win.RenderViewport(slot, pw, ph, mvp[:], eye,
		m.TriVerts, m.TriVCount, m.TriIndices,
		m.OccVerts, m.OccVCount, m.OccIndices,
		m.LineVerts, m.LineVCount, m.LineIndices,
		m.HidVerts, m.HidVCount, m.HidIndices,
		m.TopTriVerts, m.TopTriVCount, m.TopTriIndices,
		m.TopLineVerts, m.TopLineVCount, m.TopLineIndices,
		m.TriBiasFirst, m.TopTriSolidFirst, s.ActiveSectionClip(), // section-plane clip (M12-F04); #1489 solid-glyph split
		mats, recs, geomKey) // instanced draw (ADR-0038); geomKey gates the geometry re-upload (#1422)
	frameStats.gpuNs = time.Since(tg).Nanoseconds()
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
	// Depth layering (the sketch entities are coplanar with the grid, so depth-testing them
	// z-fights): the grid stays depth-tested at the bottom; every sketch overlay is marked OnTop
	// so it draws in the depth-disabled lane in submission order — entities first, then dimensions,
	// then interaction feedback. So grid < entities < dimensions, as intended (#909).
	if g := s.Grid(); g.Visible {
		list.Items = append(gridOverlay(plane, g.SpacingModel(), g.MajorEvery), list.Items...)
	}
	list.Items = append(list.Items, onTop(sketchOverlay(s.ActiveSketch(), s.IsSelectedEntity, hoverCandidate(s)))...)
	list.Items = append(list.Items, onTop(projectedCurveOverlay(s.ActiveSketch()))...)
	dims := s.SketchDimensions()
	list.Items = append(list.Items, onTop(dimensionLines(plane, dims))...)
	if item, ok := pointsOverlay(plane, s.ActiveSketch(), pointMarkerPixels*cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, onTopItem(item))
	}
	if items, ok := toolPreview(s); ok {
		list.Items = append(list.Items, onTop(items)...)
	}
	if item, ok := inferenceGlyphs(s, plane, glyphPixels*cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, onTopItem(item))
	}
	if item, ok := snapMarker(s, plane, cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, onTopItem(item))
	}
	return list, plane, dims
}

// onTop marks every item to draw in the depth-disabled lane (over the grid), preserving the
// caller's submission order so later groups layer above earlier ones.
func onTop(items []renderer.DrawItem) []renderer.DrawItem {
	for i := range items {
		items[i].OnTop = true
	}
	return items
}

// onTopItem is onTop for a single item.
func onTopItem(item renderer.DrawItem) renderer.DrawItem {
	item.OnTop = true
	return item
}

// modelOverlays appends the 3D-model overlays (work planes, part sketches, selected edges, and
// the extrude / active-tool previews) to list.
func modelOverlays(s *app.Session, cam scene.Camera, hovered *feature.WorkPlane, list renderer.DrawList) renderer.DrawList {
	wg, hasWG := s.ActiveWorkGeometry() // a part OR an assembly's origin frame + datums (#769 parity)
	hidden := s.EditScopeHides          // hide datums created after the node being edited (issue #132)
	vis := s.ObjectVisibility()         // View ▸ Object visibility (M05-F12): hidden kinds neither draw nor pick
	if hasWG && vis.WorkPlanes {
		list.Items = append(list.Items, planesOverlay(wg.WorkPlanes(), s.SelectedWorkPlane(), hovered, hidden, s.RevealSketchHostDatums())...)
	}
	if hasWG && vis.WorkAxes {
		list.Items = append(list.Items, axesOverlay(wg.WorkAxes(), selectedWorkAxis(s), hidden)...)
	}
	if vis.Sketches {
		list.Items = append(list.Items, cachedPartSketchOverlays(s)...)
		list.Items = append(list.Items, partSketchPoints(s, pointMarkerPixels*cam.WorldPerPixel())...)
		list.Items = append(list.Items, sketch3DOverlays(s, pointMarkerPixels*cam.WorldPerPixel())...)
	}
	list.Items = append(list.Items, selectedEdgeOverlay(s)...)
	list.Items = append(list.Items, selectedMeshFacetOverlay(s)...) // picked placed-mesh facet (#1776)
	list.Items = append(list.Items, threadOverlay(s)...)
	list.Items = append(list.Items, toolHoverHighlight(s)...)
	list.Items = append(list.Items, toolSelectedHighlight(s)...)
	list.Items = append(list.Items, highlightSetItems(s)...)
	list.Items = append(list.Items, revolveCenterlineHighlight(s)...)
	list.Items = append(list.Items, activeToolPreviewItems(s)...)
	list.Items = append(list.Items, pointCloudOverlay(s, cam)...)     // attached scans (M17-F06, #645)
	list.Items = append(list.Items, s.SurfaceInterrogationItems()...) // reflection/highlight/isophote (M36-F12)
	list.Items = append(list.Items, s.DeviationItems()...)            // deviation heatmap (M36-F14)
	return list
}

// pointCloudOverlay builds only the highlight cross for a snapped/selected scan point (#645). The
// cloud's points themselves are no longer overlay markers — they render through the retained native
// GL-points buffer (uploadPointClouds / renderer point pipeline), so a large scan neither rebuilds
// marker geometry every frame nor rides (and invalidates) the whole-model geometry-upload cache.
// Only the single selected point stays a screen-sized cross, drawn on top so the snap reads clearly.
func pointCloudOverlay(s *app.Session, cam scene.Camera) []renderer.DrawItem {
	if hi, ok := s.SelectedCloudPointHighlight(pointCloudHighlightPixels * cam.WorldPerPixel()); ok {
		return []renderer.DrawItem{hi}
	}
	return nil
}

// pointCloudHighlightPixels is the on-screen half-extent of the selected scan-point highlight cross.
const pointCloudHighlightPixels = 3.0

// gizmoOverlays appends the triad and manipulator-handle geometry (M05-F13).
func gizmoOverlays(s *app.Session, cam scene.Camera, list renderer.DrawList) renderer.DrawList {
	list = triadOverlay(s, cam, list)
	return manipulatorOverlay(s, cam, list)
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

// toolPreview returns the active geometry tool's rubber-band preview at the cursor: the
// provisional shape from the placed clicks through the current mouse position, drawn solid for
// real geometry, dashed for construction geometry, and with a dotted witness line under each
// in-place dimension box (#2014).
func toolPreview(s *app.Session) ([]renderer.DrawItem, bool) {
	if !native.IsItemHovered() || s.ActiveTool() == nil {
		return nil, false
	}
	cx, cy := viewportCursor()
	cur, ok := s.CursorSketchPoint(cx, cy)
	if !ok {
		return nil, false
	}
	items := placementOverlayItems(s, s.ActiveSketch().Plane(), cur)
	return items, len(items) > 0
}

const (
	viewportNear = 0.1
	viewportFar  = 5000.0
)

// viewportClipFar returns the far clip distance for this frame. It is the fixed viewportFar
// for ordinary scenes, but extends to enclose the model once the camera pulls back far enough
// that the geometry would fall beyond it. Large imported drawings (DWG/DXF) span tens of
// thousands of units, so a fixed 5000 far plane clipped the whole sketch to nothing on
// zoom-out — the geometry was simply behind the far plane (the near plane stays fixed, so
// nothing closer is ever clipped and small scenes are unchanged). The bound comes from the
// camera's distance to the scene's bounding sphere, with a margin so the far face isn't
// exactly on the plane.
// viewportFarPlane is the per-frame far clip distance: it unions the framed geometry with the
// finished 2D- and 3D-sketch overlay extents (farBounds + cachedSketch3DBounds) and computes
// the adaptive far from that (viewportClipFar). Split so the bounding-sphere math is unit-tested
// in isolation from the package-level overlay caches.
func viewportFarPlane(s *app.Session, cam scene.Camera, mn, mx [3]float32, hasGeom bool) float64 {
	lo, hi, ok := farBounds(mn, mx, hasGeom)
	if smn, smx, sok := cachedSketch3DBounds(s); sok {
		lo, hi = unionBounds(lo, hi, ok, smn, smx)
		ok = true
	}
	if cmn, cmx, cok := pointCloudFarBounds(s); cok { // #1789: enclose a large/distant scan
		lo, hi = unionBounds(lo, hi, ok, cmn, cmx)
		ok = true
	}
	return viewportClipFar(cam, lo, hi, ok)
}

// farBounds unions the framed (instanced/triangle) bounds with the finished-sketch overlay
// extent so the adaptive far plane encloses sketch-only drawings. frameBounds/DrawListBounds
// see only non-OnTop triangles, so a DWG/DXF import — all OnTop line work — would otherwise
// report no bounds and fall back to the fixed far plane, clipping the sketch on zoom-out.
func farBounds(mn, mx [3]float32, hasGeom bool) (lo, hi [3]float32, ok bool) {
	lo, hi, ok = mn, mx, hasGeom
	if smn, smx, sok := cachedSketchOverlayBounds(); sok {
		lo, hi = unionBounds(lo, hi, ok, smn, smx)
		ok = true
	}
	return lo, hi, ok
}

// unionBounds widens an (optional) box by another box. When the first box is absent (!has) the
// second becomes the result; otherwise the two are merged component-wise. The result always has
// bounds, so the caller sets ok = true.
func unionBounds(lo, hi [3]float32, has bool, omn, omx [3]float32) ([3]float32, [3]float32) {
	if !has {
		return omn, omx
	}
	for i := 0; i < 3; i++ {
		lo[i], hi[i] = minF32(lo[i], omn[i]), maxF32(hi[i], omx[i])
	}
	return lo, hi
}

func viewportClipFar(cam scene.Camera, mn, mx [3]float32, hasGeom bool) float64 {
	if !hasGeom {
		return viewportFar
	}
	cx := (float64(mn[0]) + float64(mx[0])) / 2
	cy := (float64(mn[1]) + float64(mx[1])) / 2
	cz := (float64(mn[2]) + float64(mx[2])) / 2
	rx, ry, rz := float64(mx[0])-cx, float64(mx[1])-cy, float64(mx[2])-cz
	radius := math.Sqrt(rx*rx + ry*ry + rz*rz)
	dx, dy, dz := cam.Eye.X-cx, cam.Eye.Y-cy, cam.Eye.Z-cz
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if far := (dist + radius) * 1.1; far > viewportFar {
		return far
	}
	return viewportFar
}

// readNavInput snapshots this frame's viewport pointer state from the native layer for
// the pure ApplyNavigation mapping (see navigate.go). It must be called right after the
// viewport's InvisibleButton, so IsItemActive/Hovered refer to it. Navigation is driven by
// the middle button only (pan / Shift+orbit); the left button belongs to selection and
// box-select, handled separately, so it is not read here.
func readNavInput(s *app.Session) NavInput {
	dx, dy := native.MouseDelta()
	modal := heldNavMode()
	cx, cy := viewportCursor()
	in := NavInput{
		Hovered: native.IsItemHovered(),
		Active:  native.IsItemActive(),
		Wheel:   native.MouseWheel(),
		DX:      dx,
		DY:      dy,
		CursorX: float32(cx),
		CursorY: float32(cy),
		Middle:  native.MouseDown(native.MouseMiddle),
		Shift:   native.KeyShift(),
		Modal:   modal,
		Left:    modal != NavNone && native.MouseDown(native.MouseLeft),
	}
	in.OrbitZone = latchOrbitZone(in, cx, cy)
	in.Constrained = s.ConstrainedOrbitActive() && native.MouseDown(native.MouseLeft)
	return in
}

// orbitLatch holds the Free-Orbit ring zone for the duration of one F4 orbit drag.
var orbitLatch struct {
	active bool
	zone   OrbitZone
}

// latchOrbitZone classifies the Free-Orbit ring zone once at the start of an F4 orbit drag (Modal ==
// NavOrbit + left held) and holds it until the drag ends, so the whole drag uses the zone the cursor
// started in — Inventor's ring behaviour (#913 N5–N8). Non-F4 orbits (Shift+middle) keep OrbitFree.
func latchOrbitZone(in NavInput, cx, cy float64) OrbitZone {
	if !in.Active || in.Modal != NavOrbit || !in.Left {
		orbitLatch.active = false
		return OrbitFree
	}
	if !orbitLatch.active {
		w, h := viewportPixelSize()
		orbitLatch.zone = classifyOrbitZone(cx, cy, w/2, h/2, orbitRingRadius(w, h))
		orbitLatch.active = true
	}
	return orbitLatch.zone
}

// viewportPixelSize returns the viewport image's pixel size (ItemRectMax − ItemRectMin), valid right
// after the viewport's InvisibleButton.
func viewportPixelSize() (float64, float64) {
	x0, y0 := native.ItemRectMin()
	x1, y1 := native.ItemRectMax()
	return float64(x1 - x0), float64(y1 - y0)
}

// heldNavMode reports which hold-to-navigate function key is down — F2 pan, F3 zoom, F4 orbit
// (#911) — so a left-drag drives that gesture. F4 wins if several are held.
func heldNavMode() NavMode {
	switch {
	case native.FKeyDown(4):
		return NavOrbit
	case native.FKeyDown(3):
		return NavZoom
	case native.FKeyDown(2):
		return NavPan
	default:
		return NavNone
	}
}
