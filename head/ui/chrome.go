//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Package ui composes the Inventor-style chrome — menu bar, ribbon, model browser,
// and the viewport panel — from the live application Session each frame. There is no
// retained widget tree: every frame reads app.BuildRibbon / app.BuildBrowser and the
// current tool/selection, and Dear ImGui draws that (ADR-0004/0009). All layout lives
// here in Go; the native package only exposes ImGui verbs.
package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/head/viewport"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/renderer"
)

// DrawChrome renders one frame of chrome for the session. Call it between
// Window.BeginFrame and Window.EndFrame. It returns the id of a command the user
// activated this frame (via ribbon or menu), or "" — the caller executes it, so this
// function stays free of side effects on the model. The window is needed to render the
// 3D viewport into its offscreen target.
func DrawChrome(win *native.Window, s *app.Session) string {
	if icons == nil {
		icons = newIconCache(win) // lazily bind the icon cache to this window
	}
	applyThemeIfChanged(win, s) // restyle ImGui + overlays when the theme changed (live preview)
	handleKeyboard(s)
	activated := drawMenuBar(s)
	dockID := native.DockSpaceOverMain()
	if !dockLaidOut {
		native.DockDefaultLayout(dockID, "Ribbon", "Model", "Viewport", "Status")
		dockLaidOut = true
	}
	followActiveDocument(s)
	if id := drawRibbon(s); id != "" {
		activated = id
	}
	drawBrowser(s)
	drawViewportPanel(win, s)
	drawDimensionPopup(s)
	drawExtrudeDialog(s)
	drawRevolveDialog(s)
	drawCoilDialog(s)
	drawLoftDialog(s)
	drawSweepDialog(s)
	drawHoleDialog(s)
	drawChamferDialog(s)
	drawShellDialog(s)
	drawOffsetPlaneDialog(s)
	drawFeatureEditDialog(s)
	drawStatusBar(s)
	drawPreferencesWindow(s)
	drawMaterialsWindow(s)
	drawFileDialog(s)
	return activated
}

// handleKeyboard routes global shortcuts to the session. Esc cancels the active tool at
// any point (or clears the selection when idle) — Inventor's universal cancel.
func handleKeyboard(s *app.Session) {
	if native.EscapePressed() {
		_ = s.PressKey(app.KeyEvent{Key: "Escape"})
	}
}

// dockLaidOut guards the one-time default panel arrangement (the dockspace persists
// across frames; the layout is only built once so the user can rearrange afterwards).
var dockLaidOut bool

// drawStatusBar renders Inventor's status bar: the active command's step prompt and the
// selection count, with OK/Cancel for the running tool. OK/Cancel act on the session
// directly — a tool's lifecycle is a UI concern — mirroring the Edit-menu Cancel above.
func drawStatusBar(s *app.Session) {
	if native.Begin("Status") {
		sb := app.BuildStatus(s)
		native.Text(sb.Prompt)
		if sb.ToolActive {
			native.SameLine()
			native.BeginDisabled(!sb.CanCommit)
			if native.Button("OK") {
				_ = s.OK() // a failed commit keeps the tool open (Inventor behavior)
			}
			native.EndDisabled()
			native.SameLine()
			if native.Button("Cancel") {
				s.CancelTool()
			}
		}
		native.SameLine()
		native.Text(selectionText(sb.SelectionCount))
	}
	native.End()
}

// selectionText renders the selection count as a short status (e.g. "1 selected").
func selectionText(n int) string {
	return strconv.Itoa(n) + " selected"
}

// drawMenuBar renders the top menu bar. Only the few items that map to real session
// verbs are wired; the rest grow with the feature set.
func drawMenuBar(s *app.Session) string {
	if !native.BeginMainMenuBar() {
		return ""
	}
	if native.BeginMenu("File") {
		if native.MenuItem("New Part") {
			createNewPart(s)
		}
		if native.MenuItem("Open") {
			fileModal.openFor(dialogOpen)
		}
		if native.MenuItem("Save") {
			saveActive(s)
		}
		if native.MenuItem("Save As") {
			fileModal.openFor(dialogSaveAs)
		}
		native.EndMenu()
	}
	if native.BeginMenu("Edit") {
		if native.MenuItem("Cancel Tool (Esc)") && s.ActiveTool() != nil {
			s.CancelTool()
		}
		native.EndMenu()
	}
	if native.BeginMenu("Tools") {
		if native.MenuItem("Materials") {
			showMaterials = !showMaterials
		}
		if native.MenuItem("Preferences") {
			showPreferences = !showPreferences
		}
		native.EndMenu()
	}
	native.EndMainMenuBar()
	return ""
}

// createNewPart creates a fresh, realized part document and makes it active (the
// workspace activates a newly added document), so the viewport and document tabs
// switch to it immediately. The auto-generated name (Part1, Part2…) is unique among
// open documents, so Add cannot fail on a name clash.
func createNewPart(s *app.Session) {
	_, _ = compdef.AddPart(s.Workspace(), uniquePartName(s), true)
}

// uniquePartName returns the first "PartN" name not currently open.
func uniquePartName(s *app.Session) string {
	taken := map[string]bool{}
	for _, d := range s.Workspace().Documents() {
		taken[d.DisplayName()] = true
	}
	for n := 1; ; n++ {
		if name := "Part" + strconv.Itoa(n); !taken[name] {
			return name
		}
	}
}

// prevInSketch tracks the sketch-environment state across frames so the ribbon can
// auto-switch to the contextual Sketch tab on entry and back to 3D Model on exit — but
// only on the transition frame, so the user can still pick tabs by hand afterwards.
var prevInSketch bool

// drawRibbon renders the ribbon as Inventor's two-level layout: a tab bar of command
// tabs, each tab holding its panels of command buttons. A disabled command renders a
// disabled button (its predicate is re-evaluated every frame). The contextual Sketch
// tab is auto-selected when entering the sketch environment. Returns the id of a
// clicked command, or "".
func drawRibbon(s *app.Session) string {
	force := contextualTab(s)
	var activated string
	if native.Begin("Ribbon") && native.BeginTabBar("##ribbon-tabs") {
		for _, tab := range app.BuildRibbon(s).Tabs {
			if native.BeginTabItemSelected(tab.Name, tab.Name == force) {
				if id := drawTabPanels(tab.Panels); id != "" {
					activated = id
				}
				native.EndTabItem()
			}
		}
		native.EndTabBar()
	}
	native.End()
	return activated
}

// drawTabPanels lays the tab's panels out horizontally — each panel is a layout group
// (button row + title) and panels sit SameLine with a vertical divider between them, so
// no panel is pushed off-screen the way a vertical stack hid the Sketch tab's Exit panel.
func drawTabPanels(panels []app.RibbonPanel) string {
	var activated string
	for i, panel := range panels {
		if i > 0 {
			native.SameLine()
			native.SeparatorVertical()
			native.SameLine()
		}
		if id := drawPanel(panel); id != "" {
			activated = id
		}
	}
	return activated
}

// contextualTab returns the ribbon tab to force-select this frame when the sketch
// environment was just entered ("Sketch") or left ("3D Model"), else "".
func contextualTab(s *app.Session) string {
	cur := s.InSketch()
	if cur == prevInSketch {
		return ""
	}
	prevInSketch = cur
	if cur {
		return "Sketch"
	}
	return "3D Model"
}

// ribbonMaxRows caps how many button rows a panel uses (Inventor stacks small buttons a
// few rows deep); the column count grows to fit, keeping each panel narrow so panels sit
// side-by-side without running off the ribbon.
const ribbonMaxRows = 3

// panelCols returns how many columns to wrap a panel's n buttons into, bounded so the
// panel is at most ribbonMaxRows tall.
func panelCols(n int) int {
	if n <= ribbonMaxRows {
		return 1
	}
	return (n + ribbonMaxRows - 1) / ribbonMaxRows
}

// drawPanel renders one ribbon panel as a self-contained layout group: a compact grid of
// command buttons with the panel title beneath them (Inventor's panel layout), so the
// whole panel is one narrow, horizontally-placeable unit. The title uses plain Text (not
// SeparatorText, which would stretch the group to the full window width and hide every
// panel to its right). Returns the id of a clicked command, or "".
func drawPanel(panel app.RibbonPanel) string {
	var activated string
	cols := panelCols(len(panel.Buttons))
	native.BeginGroup()
	for i, btn := range panel.Buttons {
		if i > 0 && i%cols != 0 {
			native.SameLine() // continue the current row; a new row starts at each multiple of cols
		}
		if id := drawRibbonButton(btn); id != "" {
			activated = id
		}
	}
	native.Text(panel.Name) // panel title under its buttons
	native.EndGroup()
	return activated
}

// drawRibbonButton renders one command in its configured style (text, small icon, or
// large icon), greyed when its predicate is false, with the command tooltip on hover.
// It returns the command id when clicked this frame, else "".
func drawRibbonButton(btn app.RibbonButton) string {
	native.BeginDisabled(!btn.Enabled)
	clicked := drawButtonControl(btn)
	native.EndDisabled()
	if clicked {
		return btn.Command.ID()
	}
	return ""
}

// drawButtonControl draws the command's clickable control and its tooltip, returning
// whether it was clicked. An icon-style command falls back to a labeled text button
// when its glyph texture is unavailable (missing asset or upload failure), so a missing
// icon never hides the command.
func drawButtonControl(btn app.RibbonButton) bool {
	if px, ok := iconSizeFor(btn.Command.ButtonStyle()); ok {
		if tex, ok := icons.texture(btn.Command.Icon(), px); ok {
			return drawIconButton(btn, tex, float32(px))
		}
	}
	clicked := native.Button(btn.Command.DisplayName())
	native.SetItemTooltip(btn.Command.Tooltip())
	return clicked
}

// iconSizeFor returns the rasterization size for an icon button style, or false for a
// text-only command.
func iconSizeFor(s app.ButtonStyle) (int, bool) {
	switch s {
	case app.SmallIconButton:
		return smallIconPx, true
	case app.LargeIconButton:
		return largeIconPx, true
	default:
		return 0, false
	}
}

// drawIconButton renders an icon button: small ones are icon-only (dense tool grids),
// large ones place the name as a caption beneath the icon (Inventor's large button).
// The icon is the click target either way.
func drawIconButton(btn app.RibbonButton, tex uint64, px float32) bool {
	if btn.Command.ButtonStyle() != app.LargeIconButton {
		clicked := native.ImageButton(btn.Command.ID(), tex, px, px, iconTint)
		native.SetItemTooltip(btn.Command.Tooltip())
		return clicked
	}
	native.BeginGroup()
	clicked := native.ImageButton(btn.Command.ID(), tex, px, px, iconTint)
	native.SetItemTooltip(btn.Command.Tooltip())
	native.Text(btn.Command.DisplayName())
	native.EndGroup()
	return clicked
}

// prevFramedDoc tracks which document the camera was last framed to, so switching
// the active document (New Part, a tab, an add-in) reframes the view to the new one.
var prevFramedDoc uint64

// followActiveDocument reframes the viewport when the active document changes, so a
// switch actually shows the new document. Without this the camera stays on the
// previous document's view — and since an empty new part has no bounds to fit, it
// would look like an empty viewport.
func followActiveDocument(s *app.Session) {
	active := s.ActiveDocument()
	var cur uint64
	if active != nil {
		cur = uint64(active.ID())
	}
	if cur == prevFramedDoc {
		return
	}
	prevFramedDoc = cur
	if active == nil || s.CameraAnimating() {
		return
	}
	// Frame the geometry if there is any; otherwise center on the model origin so the
	// part's coordinate planes are visible rather than a void (Home keeps the current
	// target on an empty model, so point it at the origin first).
	if len(activeBodies(s)) == 0 {
		cam := s.Camera()
		cam.Target = math.P3(0, 0, 0)
		s.SetCamera(cam)
	}
	s.HomeView()
}

// prevActiveDoc tracks the active document id across frames so a programmatic switch
// (e.g. an add-in activating a document) force-selects its tab once, without fighting
// the user's own tab clicks on subsequent frames.
var prevActiveDoc uint64

// drawDocumentTabs renders one tab per open document at the top of the viewport. The
// active document's tab is shown selected; clicking another tab activates that
// document, and when the active document changes elsewhere (an add-in, the menu) its
// tab is selected so the strip follows. The tabs read the workspace each frame, so
// opening/closing/activating documents is reflected automatically.
func drawDocumentTabs(s *app.Session) {
	docs := s.Workspace().Documents()
	if len(docs) == 0 {
		return
	}
	active := s.ActiveDocument()
	var cur uint64
	if active != nil {
		cur = uint64(active.ID())
	}
	// Force-select the active tab only on the frame the active document changed out
	// from under the UI (a New Part, an add-in's activate_document); otherwise leave
	// selection to the user's clicks.
	force := cur != prevActiveDoc
	prevActiveDoc = cur

	if native.BeginTabBar("##doc-tabs") {
		for _, d := range docs {
			selected := force && active != nil && d.ID() == active.ID()
			open := native.BeginTabItemSelected(d.DisplayName(), selected)
			// Only treat a tab as a user click on non-force frames. On a force frame we
			// are asserting the active tab via SetSelected, which ImGui applies on the
			// NEXT frame — so this frame BeginTabItem still reports the OLD visible tab.
			// Acting on that would call SetActiveDocument for the old document and flip
			// the active doc back, oscillating every frame.
			if open && !force && (active == nil || d.ID() != active.ID()) {
				_ = s.Workspace().SetActiveDocument(d)
			}
			if open {
				native.EndTabItem()
			}
		}
		native.EndTabBar()
	}
}

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

		// During a camera transition (entering/exiting a sketch) advance the tween and
		// ignore user input; otherwise apply navigation then resolve clicks/hover so the
		// picker hit-tests against the current view.
		placing := isPlacingTool(s)
		var hovered *feature.WorkPlane
		cam := s.Camera()
		cam.Width, cam.Height = pw, ph
		if s.CameraAnimating() {
			s.SetCamera(cam)
			s.TickCameraAnimation(float64(native.DeltaTime()))
			cam = s.Camera()
			cam.Width, cam.Height = pw, ph
		} else {
			cam = ApplyNavigation(cam, readNavInput(placing))
			s.SetCamera(cam)
			handleViewportClick(s)
			hovered = hoveredPlane(s)
		}

		bodies := activeBodies(s)
		list := renderer.BuildDrawList(bodies, cam, ops.DefaultQuality(), s.SurfaceLookup())
		// Paint the selected body cyan so a browser/viewport pick reads in the 3D view
		// (a no-op in sketch mode, where the selection is sketch entities, not bodies).
		list = highlightSelection(list, s.Selection().First(), bodies)
		var dims []app.DimensionView
		var sketchPlane sketch.Plane
		if s.InSketch() {
			sketchPlane = s.ActiveSketch().Plane()
			if g := s.Grid(); g.Visible {
				list.Items = append(gridOverlay(sketchPlane, g.SpacingModel(), g.MajorEvery), list.Items...)
			}
			list.Items = append(list.Items, sketchOverlay(s.ActiveSketch(), s.IsSelectedEntity, hoverCandidate(s))...)
			dims = s.SketchDimensions()
			list.Items = append(list.Items, dimensionLines(sketchPlane, dims)...)
			if item, ok := pointsOverlay(sketchPlane, s.ActiveSketch(), pointMarkerPixels*cam.WorldPerPixel()); ok {
				list.Items = append(list.Items, item)
			}
			if item, ok := toolPreview(s); ok {
				list.Items = append(list.Items, item)
			}
			if item, ok := snapMarker(s, sketchPlane, cam.WorldPerPixel()); ok {
				list.Items = append(list.Items, item)
			}
		} else {
			list.Items = append(list.Items, planesOverlay(activePart(s), s.SelectedWorkPlane(), hovered)...)
			list.Items = append(list.Items, partSketchOverlays(s)...)
			list.Items = append(list.Items, extrudeHoverHighlight(s)...)
			list.Items = append(list.Items, extrudeProfileHighlight(s)...)
			list.Items = append(list.Items, activeToolPreviewItems(s)...)
		}
		m := viewport.Flatten(list)
		mvp := renderer.ViewProjection(cam, viewportNear, viewportFar)
		win.RenderViewport(pw, ph, mvp[:],
			m.TriVerts, m.TriVCount, m.TriIndices,
			m.LineVerts, m.LineVCount, m.LineIndices)
		if tex := win.ViewportTexture(); tex != 0 {
			native.SetCursorPos(cx, cy) // draw the image back over the invisible button
			native.Image(tex, float32(pw), float32(ph))
		}
		if s.InSketch() && len(dims) > 0 {
			if d := drawDimensionLabels(cx, cy, cam, sketchPlane, dims); d != nil {
				s.BeginEditDimension(d) // double-clicked a dimension's value
			}
		}
	}
	native.End()
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

// clampDim keeps the offscreen target at least 1px (ImGui can report a zero/negative
// content region on the first frame or when the panel is collapsed).
func clampDim(v float32) int {
	if v < 1 {
		return 1
	}
	return int(v)
}

// activeBodies returns the surface bodies of the active part, or nil.
func activeBodies(s *app.Session) []*topo.Body {
	part := activePart(s)
	if part == nil {
		return nil
	}
	return part.SurfaceBodies().All()
}

// activePart returns the active document's part component definition, or nil.
func activePart(s *app.Session) *compdef.PartComponentDefinition {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	part, _ := d.Content().(*compdef.PartComponentDefinition)
	return part
}
