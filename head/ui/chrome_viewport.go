//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/head/viewport"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/renderer"
	"github.com/Oblikovati/oblikovati/scene"
)

// drawViewportPanel renders the active part's geometry through the Vulkan viewport and
// shows the result as an image filling the panel. An invisible drag button under the
// image captures mouse navigation (orbit/pan/zoom) and updates the session camera, so
// the scene — rebuilt from the model each frame (renderer.BuildDrawList) and projected
// with the navigated camera — always reflects current model and view state (ADR-0004).
func drawViewportPanel(win *native.Window, s *app.Session) {
	if native.Begin("Viewport") {
		drawDocumentTabs(s)
		w, h := native.ContentRegionAvail()
		pw, ph := clampDim(w), clampDim(h)

		// Reserve the region with an input-capturing button, then read navigation from it.
		cx, cy := native.GetCursorPos()
		native.InvisibleButton("##viewport-nav", float32(pw), float32(ph))

		cam, hovered := updateViewportCamera(s, pw, ph)

		bodies := activeBodies(s)
		list := renderer.BuildDrawListStyled(bodies, cam, ops.DefaultQuality(), s.SurfaceLookup(), s.VisualStyle())
		// Paint the selected body cyan so a browser/viewport pick reads in the 3D view
		// (a no-op in sketch mode, where the selection is sketch entities, not bodies).
		list = highlightSelection(list, s.Selection().First(), bodies)
		var dims []app.DimensionView
		var sketchPlane sketch.Plane
		if s.InSketch() {
			list, sketchPlane, dims = sketchOverlays(s, cam, list)
		} else {
			list = modelOverlays(s, cam, hovered, list)
		}
		renderViewportImage(win, s, cam, list, pw, ph, cx, cy)
		if s.InSketch() && len(dims) > 0 {
			if d := drawDimensionLabels(cx, cy, cam, sketchPlane, dims); d != nil {
				s.BeginEditDimension(d) // double-clicked a dimension's value
			}
		}
	}
	native.End()
}

// updateViewportCamera sizes the camera to the panel and either advances the active camera
// tween (ignoring user input, e.g. while entering/exiting a sketch) or applies this frame's
// navigation and resolves click/hover so the picker hit-tests against the current view. It
// returns the camera to render with and the work plane under the cursor (nil while animating).
func updateViewportCamera(s *app.Session, pw, ph int) (scene.Camera, *feature.WorkPlane) {
	cam := s.Camera()
	cam.Width, cam.Height = pw, ph
	if s.CameraAnimating() {
		s.SetCamera(cam)
		s.TickCameraAnimation(float64(native.DeltaTime()))
		cam = s.Camera()
		cam.Width, cam.Height = pw, ph
		return cam, nil
	}
	cam = ApplyNavigation(cam, readNavInput(isPlacingTool(s)))
	s.SetCamera(cam)
	handleViewportClick(s)
	return cam, hoveredPlane(s)
}

// renderViewportImage flattens the draw list, renders it into the window's offscreen target
// with the camera's view-projection, and blits the resulting texture back over the
// input-capturing button at (cx,cy) so the panel shows the rendered scene.
func renderViewportImage(win *native.Window, s *app.Session, cam scene.Camera, list renderer.DrawList, pw, ph int, cx, cy float32) {
	m := viewport.Flatten(list)
	mvp := renderer.ViewProjection(cam, viewportNear, viewportFar)
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}
	win.SetViewportLighting(viewport.PackLighting(s.SceneLighting()))
	applyEnvironment(win, s.Environment())
	applySkybox(win, s.Environment(), mvp)
	applyShadow(win, s, m)
	win.RenderViewport(pw, ph, mvp[:], eye,
		m.TriVerts, m.TriVCount, m.TriIndices,
		m.OccVerts, m.OccVCount, m.OccIndices,
		m.LineVerts, m.LineVCount, m.LineIndices,
		m.HidVerts, m.HidVCount, m.HidIndices)
	if tex := win.ViewportTexture(); tex != 0 {
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
	if item, ok := snapMarker(s, plane, cam.WorldPerPixel()); ok {
		list.Items = append(list.Items, item)
	}
	return list, plane, dims
}

// modelOverlays appends the 3D-model overlays (work planes, part sketches, selected edges, and
// the extrude / active-tool previews) to list.
func modelOverlays(s *app.Session, cam scene.Camera, hovered *feature.WorkPlane, list renderer.DrawList) renderer.DrawList {
	list.Items = append(list.Items, planesOverlay(activePart(s), s.SelectedWorkPlane(), hovered)...)
	list.Items = append(list.Items, partSketchOverlays(s)...)
	list.Items = append(list.Items, partSketchPoints(s, pointMarkerPixels*cam.WorldPerPixel())...)
	list.Items = append(list.Items, selectedEdgeOverlay(s)...)
	list.Items = append(list.Items, extrudeHoverHighlight(s)...)
	list.Items = append(list.Items, extrudeProfileHighlight(s)...)
	list.Items = append(list.Items, activeToolPreviewItems(s)...)
	return list
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
