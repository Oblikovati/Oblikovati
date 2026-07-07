# Point Cloud Point Size And Intensity Chart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render explicit point-cloud Default mode in neutral grey, add a session point-size slider below density, and show an intensity histogram between the ramp swatches.

**Architecture:** Keep state and testable ribbon data in `app`; keep `head/ui` as the renderer of that model. Reuse the existing retained GPU point pipeline, existing ImGui drawing primitives, and existing point-cloud display-mode contract without adding new wire/API methods.

**Tech Stack:** Go, existing `app.Session` state, `renderer` draw-item helpers, Dear ImGui wrappers in `head/internal/native`, retained Vulkan point upload path.

---

### Task 1: Session Point Size And Grey Default Color

**Files:**
- Modify: `app/session.go`
- Modify: `app/point_cloud_gpu.go`
- Modify: `renderer/point_markers.go`
- Modify: `../Oblikovati.API/types/point_cloud_display_mode.go`
- Test: `app/point_cloud_gpu_test.go`
- Test: `app/point_cloud_render_test.go`

- [ ] **Step 1: Add failing point-size tests**

Add these tests to `app/point_cloud_gpu_test.go`:

```go
func TestPointCloudPointSizeDefaultAndClamp(t *testing.T) {
	s := NewSession()
	if got := s.PointCloudPointSize(); got != 1 {
		t.Fatalf("default point size = %g, want 1", got)
	}
	s.SetPointCloudPointSize(-4)
	if got := s.PointCloudPointSize(); got != 1 {
		t.Errorf("negative point size = %g, want clamp to 1", got)
	}
	s.SetPointCloudPointSize(12)
	if got := s.PointCloudPointSize(); got != 10 {
		t.Errorf("oversize point size = %g, want clamp to 10", got)
	}
	s.SetPointCloudPointSize(4.5)
	if got := s.PointCloudPointSize(); got != 4.5 {
		t.Errorf("point size = %g, want 4.5", got)
	}
}
```

- [ ] **Step 2: Update existing color expectations to fail for cyan**

In `app/point_cloud_gpu_test.go`, change `TestPointCloudGPUVerticesInterleave` to expect the shared grey default:

```go
	grey := renderer.PointCloudColor
	if verts[3] != grey[0] || verts[4] != grey[1] || verts[5] != grey[2] || verts[6] != grey[3] {
		t.Errorf("first color = %v, want default grey %v", verts[3:7], grey)
	}
```

Add `oblikovati.org/renderer` to that test file's imports.

Add this assertion to `app/point_cloud_render_test.go` after the RGB/intensity checks:

```go
	pc.SetDisplayMode(types.PointCloudDisplayModeDefault)
	items = s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 || items[0].Colors[0] != renderer.PointCloudColor {
		t.Fatalf("default-mode color = %+v, want grey %v", items, renderer.PointCloudColor)
	}
```

- [ ] **Step 3: Run the focused tests and verify failure**

Run:

```bash
go test ./app -run 'TestPointCloudPointSizeDefaultAndClamp|TestPointCloudGPUVerticesInterleave|TestPointCloudItemsColorsByDisplayMode'
```

Expected: FAIL because `PointCloudPointSize` is undefined and `renderer.PointCloudColor` is still cyan.

- [ ] **Step 4: Add session state and clamping**

In `app/session.go`, add the field near `pointCloudRenderDensity`:

```go
	pointCloudPointSize       float32                         // session viewport point size for point clouds, in pixels
```

Initialize it in `newSession`:

```go
		pointCloudPointSize:       1,
```

Add these methods near `PointCloudRenderDensity`:

```go
// PointCloudPointSize returns the viewport point size for attached scan points, in pixels.
func (s *Session) PointCloudPointSize() float32 { return s.pointCloudPointSize }

// SetPointCloudPointSize sets the viewport point size for attached scan points. Values outside
// 1..10 are clamped because UI drags and future API callers share this setter.
func (s *Session) SetPointCloudPointSize(size float32) {
	if size < 1 {
		size = 1
	}
	if size > 10 {
		size = 10
	}
	s.pointCloudPointSize = size
}
```

- [ ] **Step 5: Change the shared fallback point color to grey**

In `renderer/point_markers.go`, update the comment and color:

```go
// PointCloudColor is the neutral default marker color for point-cloud display modes that do not
// use an RGB or intensity channel.
var PointCloudColor = [4]float32{0.72, 0.72, 0.72, 1}
```

In `../Oblikovati.API/types/point_cloud_display_mode.go`, update only the comment:

```go
	// PointCloudDisplayModeDefault keeps a neutral host-defined marker color.
```

- [ ] **Step 6: Run tests and commit**

Run:

```bash
go test ./app -run 'TestPointCloudPointSizeDefaultAndClamp|TestPointCloudGPUVerticesInterleave|TestPointCloudItemsColorsByDisplayMode'
go test ./renderer
go test ../Oblikovati.API/types
```

Expected: PASS.

Commit only the hunks from this task. If the worktree still contains unrelated uncommitted changes in these files, stage interactively or skip the commit and report that isolation is unsafe.

```bash
git diff -- app/session.go app/point_cloud_gpu.go app/point_cloud_gpu_test.go app/point_cloud_render_test.go renderer/point_markers.go
git -C ../Oblikovati.API diff -- types/point_cloud_display_mode.go
```

### Task 2: App Ribbon Model For Point Size And Histogram Data

**Files:**
- Modify: `app/ribbon.go`
- Test: `app/point_cloud_ribbon_test.go`

- [ ] **Step 1: Add failing ribbon model tests**

In `app/point_cloud_ribbon_test.go`, extend the existing Point Cloud Display panel assertions:

```go
	if modePanel.PointSizeSlider == nil {
		t.Fatal("Point Cloud controls are missing the Point Size slider")
	}
	if modePanel.PointSizeSlider.Value != 1 || modePanel.PointSizeSlider.Min != 1 || modePanel.PointSizeSlider.Max != 10 {
		t.Fatalf("Point Size slider = %+v, want 1..10 at 1", modePanel.PointSizeSlider)
	}
```

After intensity mode is selected and `displayPanel.IntensityRamp` is checked, add:

```go
	if displayPanel.IntensityRamp.Histogram != nil {
		t.Fatalf("cloud without intensity data should expose an empty histogram, got %v", displayPanel.IntensityRamp.Histogram)
	}
```

Add this focused unit test to the same file:

```go
func TestPointCloudIntensityHistogramBins(t *testing.T) {
	pc := pointcloud.NewWithSamples("Scan", "s.xyz", "rid", []pointcloud.PointSample{
		{Point: math.P3(0, 0, 0), HasIntensity: true, Intensity: 0},
		{Point: math.P3(1, 0, 0), HasIntensity: true, Intensity: 0},
		{Point: math.P3(2, 0, 0), HasIntensity: true, Intensity: 50},
		{Point: math.P3(3, 0, 0), HasIntensity: true, Intensity: 100},
	})
	got := pointCloudIntensityHistogram(pc, 4)
	want := []float32{1, 0, 0.5, 0.5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("histogram = %v, want %v", got, want)
	}
	pc = pointcloud.NewWithSamples("Flat", "s.xyz", "rid", []pointcloud.PointSample{
		{Point: math.P3(0, 0, 0), HasIntensity: true, Intensity: 3},
		{Point: math.P3(1, 0, 0), HasIntensity: true, Intensity: 3},
	})
	if got := pointCloudIntensityHistogram(pc, 4); got != nil {
		t.Fatalf("flat histogram = %v, want nil", got)
	}
}
```

Add `reflect` and `oblikovati.org/model/pointcloud` to that test file's imports.

Add this panel-level test to the same file:

```go
func TestPointCloudIntensityRampPanelCarriesHistogram(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasIntensity: true, Intensity: 0},
		{Point: math.P3(1, 0, 5), HasIntensity: true, Intensity: 50},
		{Point: math.P3(2, 0, 5), HasIntensity: true, Intensity: 100},
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})

	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel(panelPointCloudDisplay)
	if !ok || panel.IntensityRamp == nil {
		t.Fatal("Point Cloud Display panel should show the intensity ramp in intensity mode")
	}
	if got := len(panel.IntensityRamp.Histogram); got != pointCloudIntensityHistogramBins {
		t.Fatalf("histogram bins = %d, want %d", got, pointCloudIntensityHistogramBins)
	}
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./app -run 'TestPointCloudRibbonButtonsWiredAndEnable|TestPointCloudIntensityHistogramBins|TestPointCloudIntensityRampPanelCarriesHistogram'
```

Expected: FAIL because `PointSizeSlider`, `Histogram`, and `pointCloudIntensityHistogram` do not exist.

- [ ] **Step 3: Extend the ribbon model types**

In `app/ribbon.go`, change `RibbonPanel` to include the second slider:

```go
	PointSizeSlider *RibbonSlider
```

Change `RibbonSlider` to include a display format flag:

```go
	Percent bool
```

Change `RibbonColorRamp` to carry chart data:

```go
	Histogram []float32
```

- [ ] **Step 4: Inject point-size slider and histogram data**

In `attachPointCloudDisplayControlsToTab`, add the point-size slider next to density:

```go
		tab.Panels[pi].Slider = pointCloudRenderDensitySlider(s)
		tab.Panels[pi].PointSizeSlider = pointCloudPointSizeSlider(s)
		tab.Panels[pi].IntensityRamp = pointCloudIntensityRampControls(s)
```

Update `pointCloudRenderDensitySlider`:

```go
		Percent: true,
```

Add:

```go
func pointCloudPointSizeSlider(s *Session) *RibbonSlider {
	return &RibbonSlider{
		ID:      "PointCloud.PointSize",
		Label:   "Point Size",
		Value:   s.PointCloudPointSize(),
		Min:     1,
		Max:     10,
		Tooltip: "Point-cloud point size — draw scan points from 1 to 10 pixels wide.",
	}
}
```

Update `pointCloudIntensityRampControls` to set the histogram:

```go
		Histogram: pointCloudIntensityHistogram(pc, pointCloudIntensityHistogramBins),
```

- [ ] **Step 5: Add histogram generation**

Add these helpers to `app/ribbon.go`:

```go
const pointCloudIntensityHistogramBins = 32

func pointCloudIntensityHistogram(pc *pointcloud.PointCloud, bins int) []float32 {
	min, max, ok := pc.IntensityRange()
	if !ok || max <= min || bins <= 0 {
		return nil
	}
	counts := make([]int, bins)
	peak := 0
	for _, sample := range pc.DisplayedSamples() {
		if !sample.HasIntensity {
			continue
		}
		i := int(((sample.Intensity - min) / (max - min)) * float64(bins))
		if i < 0 {
			i = 0
		}
		if i >= bins {
			i = bins - 1
		}
		counts[i]++
		if counts[i] > peak {
			peak = counts[i]
		}
	}
	if peak == 0 {
		return nil
	}
	out := make([]float32, bins)
	for i, n := range counts {
		out[i] = float32(n) / float32(peak)
	}
	return out
}
```

Add `oblikovati.org/model/pointcloud` to `app/ribbon.go` imports.

- [ ] **Step 6: Run tests and commit**

Run:

```bash
go test ./app -run 'TestPointCloudRibbonButtonsWiredAndEnable|TestPointCloudIntensityHistogramBins|TestPointCloudIntensityRampPanelCarriesHistogram'
```

Expected: PASS.

Commit only this task's hunks, or skip commit if unrelated edits in the same files cannot be safely staged.

### Task 3: Head Ribbon Rendering For Two Sliders And Separated Ramp

**Files:**
- Modify: `head/ui/chrome_ribbon.go`
- Test: `head/ui/point_cloud_ribbon_shot_test.go`

- [ ] **Step 1: Update the UI screenshot setup to exercise intensity controls**

In `head/ui/point_cloud_ribbon_shot_test.go`, create samples with intensity and select intensity mode before rendering:

```go
	pc, _ := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasIntensity: true, Intensity: 0},
		{Point: math.P3(2, 0, 5), HasIntensity: true, Intensity: 50},
		{Point: math.P3(0, 2, 5), HasIntensity: true, Intensity: 100},
	})
	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
```

Add imports for `oblikovati.org/api/types` and `oblikovati.org/model/pointcloud`.

Set the capture window large enough for the taller panel:

```go
		native.SetNextWindowSize(460, 210)
```

- [ ] **Step 2: Run the cgo screenshot test and verify current UI is incomplete**

Run:

```bash
cd head && go test ./ui -run TestInWindowPointCloudPanelButtons
```

Expected: PASS or SKIP depending on local GPU/window availability, but the saved screenshot still shows adjacent swatches and no point-size slider.

- [ ] **Step 3: Route slider edits by slider id and format**

In `head/ui/chrome_ribbon.go`, update `drawSelectorPanel`:

```go
	if panel.Slider != nil {
		drawRibbonSlider(s, panel.Slider, width)
	}
	if panel.PointSizeSlider != nil {
		drawRibbonSlider(s, panel.PointSizeSlider, width)
	}
```

Replace `drawRibbonSlider` with:

```go
func drawRibbonSlider(s *app.Session, slider *app.RibbonSlider, width float32) {
	value := slider.Value
	native.SetNextItemWidth(width)
	var changed bool
	if slider.Percent {
		changed = native.SliderPercent("##"+slider.ID, &value, slider.Min, slider.Max)
	} else {
		changed = native.SliderFloat("##"+slider.ID, &value, slider.Min, slider.Max)
	}
	if changed {
		applyRibbonSlider(s, slider.ID, value)
	}
	if slider.Tooltip != "" {
		native.SetItemTooltip(slider.Tooltip)
	}
}

func applyRibbonSlider(s *app.Session, id string, value float32) {
	switch id {
	case "PointCloud.RenderDensity":
		s.SetPointCloudRenderDensity(value)
	case "PointCloud.PointSize":
		s.SetPointCloudPointSize(value)
	}
}
```

- [ ] **Step 4: Draw the intensity swatches separated by a chart**

Replace `drawIntensityRampControls` with:

```go
func drawIntensityRampControls(s *app.Session, ramp *app.RibbonColorRamp) {
	low, high := ramp.Low.Value, ramp.High.Value
	if native.ColorSwatch("##"+ramp.Low.ID, &low) {
		s.SetPointCloudIntensityRamp(low, high)
	}
	if ramp.Low.Tooltip != "" {
		native.SetItemTooltip(ramp.Low.Tooltip)
	}
	native.SameLine()
	drawIntensityHistogramChart(ramp.Histogram, low, high, intensityChartWidth, intensityChartHeight)
	native.SameLine()
	if native.ColorSwatch("##"+ramp.High.ID, &high) {
		s.SetPointCloudIntensityRamp(low, high)
	}
	if ramp.High.Tooltip != "" {
		native.SetItemTooltip(ramp.High.Tooltip)
	}
}
```

Add the chart constants and helpers:

```go
const (
	intensityChartWidth  = 92
	intensityChartHeight = 24
)

func drawIntensityHistogramChart(hist []float32, low, high [4]float32, width, height float32) {
	x, y := native.GetCursorScreenPos()
	bottom := y + height
	native.DrawRectFilled(x, y, x+width, bottom, [4]float32{0.08, 0.08, 0.08, 0.35})
	if len(hist) > 0 {
		drawIntensityHistogramArea(hist, low, high, x, y, width, height)
	}
	native.DrawLine(x, bottom-1, x+width, bottom-1, [4]float32{0.95, 0.95, 0.95, 0.55}, 1)
	native.Dummy(width, height)
}

func drawIntensityHistogramArea(hist []float32, low, high [4]float32, x, y, width, height float32) {
	if len(hist) == 1 {
		top := y + height*(1-clampChart01(hist[0]))
		native.DrawRectFilled(x, top, x+width, y+height, low)
		return
	}
	for i := 0; i < len(hist)-1; i++ {
		x0 := x + width*float32(i)/float32(len(hist)-1)
		x1 := x + width*float32(i+1)/float32(len(hist)-1)
		y0 := y + height*(1-clampChart01(hist[i]))
		y1 := y + height*(1-clampChart01(hist[i+1]))
		c := lerpChartColor(low, high, float32(i)/float32(len(hist)-1))
		native.DrawQuadFilled(x0, y+height, x1, y+height, x1, y1, x0, y0, c)
	}
}

func clampChart01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpChartColor(a, b [4]float32, t float32) [4]float32 {
	return [4]float32{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
		0.8,
	}
}
```

- [ ] **Step 5: Increase ribbon band height without changing small-button packing**

In `head/ui/chrome_ribbon.go`, keep `ribbonMaxRows = 3` for button packing and add:

```go
const ribbonControlRows = 4
```

Update `ribbonGridHeight` so the band seats combo, density, point-size, and chart rows:

```go
func ribbonGridHeight(m native.StyleMetrics) float32 {
	buttonRow := float32(scaledIconPx(smallIconPx)) + 2*m.FramePadY
	buttonGrid := ribbonRowsHeight(ribbonMaxRows, buttonRow, m.ItemSpacingY)
	controlGrid := ribbonRowsHeight(ribbonControlRows, native.FrameHeight(), m.ItemSpacingY)
	return maxF32(buttonGrid, controlGrid)
}

func ribbonRowsHeight(rows int, row, spacing float32) float32 {
	if rows <= 0 {
		return 0
	}
	return float32(rows)*row + float32(rows-1)*spacing
}
```

`maxF32` already exists in the `head/ui` package and can be reused from `chrome_ribbon.go`.

- [ ] **Step 6: Run head UI test and commit**

Run:

```bash
cd head && go test ./ui -run TestInWindowPointCloudPanelButtons
```

Expected: PASS or SKIP on systems without a usable window/GPU. If it runs, inspect `head/ui/test-output/point-cloud-panel.png` and verify the point-size slider appears under density and the swatches are separated by the chart.

Commit only this task's hunks, or skip commit if unrelated edits in the same files cannot be safely staged.

### Task 4: Use Session Point Size In The Viewport Upload Path

**Files:**
- Modify: `head/ui/chrome_viewport.go`
- Modify: `head/ui/point_cloud_gpu.go`
- Test: `head/internal/native/viewport_point_upload_test.go`

- [ ] **Step 1: Preserve retained upload behavior while varying size**

Add this check to `TestViewportPointUploadRetained` after the static orbit assertion:

```go
	w.UploadPoints(pts, nPts, keyA, 7.0)
	if got := w.ViewportPointUploads(); got != base {
		t.Errorf("point-size-only change re-uploaded points: uploads went %d to %d, want no buffer transfer", base, got)
	}
```

- [ ] **Step 2: Run native test**

Run:

```bash
cd head && go test ./internal/native -run TestViewportPointUploadRetained
```

Expected: PASS or SKIP. This documents the native behavior already present: size changes update `pointSizePx` without transferring vertices when the key is resident.

- [ ] **Step 3: Remove fixed point size from upload call**

In `head/ui/chrome_viewport.go`, change:

```go
	uploadPointClouds(win, s, pointCloudMarkerPixels)
```

to:

```go
	uploadPointClouds(win, s)
```

In `head/ui/point_cloud_gpu.go`, add a native-window guard to the package state:

```go
var (
	pointUploadKey    uint64
	pointUploadValid  bool
	pointUploadWindow *native.Window
)
```

Then change the function signature and upload call:

```go
func uploadPointClouds(win *native.Window, s *app.Session) {
	key := s.PointCloudDisplayKey()
	sizePx := s.PointCloudPointSize()
	if pointUploadWindow != win {
		pointUploadKey, pointUploadValid = 0, false
		pointUploadWindow = win
	}
	if pointUploadValid && key == pointUploadKey {
		win.UploadPoints(nil, 0, key, sizePx)
		return
	}
	verts, count := s.PointCloudGPUVertices()
	win.UploadPoints(verts, count, key, sizePx)
	pointUploadKey, pointUploadValid = key, true
}
```

- [ ] **Step 4: Rename the remaining overlay constant**

In `head/ui/chrome_viewport.go`, rename `pointCloudMarkerPixels` to `pointCloudHighlightPixels` for the selected-point cross only:

```go
	if hi, ok := s.SelectedCloudPointHighlight(pointCloudHighlightPixels * cam.WorldPerPixel()); ok {
```

Update the comment:

```go
// pointCloudHighlightPixels is the half-extent of the selected-point highlight cross.
const pointCloudHighlightPixels = 3.0
```

- [ ] **Step 5: Run head tests and commit**

Run:

```bash
cd head && go test ./ui -run TestInWindowPointCloudPanelButtons
cd head && go test ./internal/native -run TestViewportPointUploadRetained
```

Expected: PASS or SKIP if no window/GPU is available.

Commit only this task's hunks, or skip commit if unrelated edits in the same files cannot be safely staged.

### Task 5: Final Verification

**Files:**
- No source edits.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w app/session.go app/point_cloud_gpu.go app/point_cloud_gpu_test.go app/point_cloud_render_test.go app/point_cloud_ribbon_test.go app/ribbon.go renderer/point_markers.go head/ui/chrome_ribbon.go head/ui/chrome_viewport.go head/ui/point_cloud_gpu.go head/ui/point_cloud_ribbon_shot_test.go head/internal/native/viewport_point_upload_test.go ../Oblikovati.API/types/point_cloud_display_mode.go
```

Expected: command exits 0.

- [ ] **Step 2: Run focused app and renderer tests**

Run:

```bash
go test ./renderer ./app -run 'Test.*PointCloud.*|TestSurfacesMeshTabPanelOrder'
```

Expected: PASS.

- [ ] **Step 3: Run API type tests**

Run:

```bash
go test ../Oblikovati.API/types
```

Expected: PASS.

- [ ] **Step 4: Run cgo/head focused tests**

Run:

```bash
cd head && go test ./ui -run TestInWindowPointCloudPanelButtons
cd head && go test ./internal/native -run TestViewportPointUploadRetained
```

Expected: PASS on a machine with a usable window/GPU, otherwise SKIP with the skip reason reported.

- [ ] **Step 5: Review diff before handoff**

Run:

```bash
git diff -- app/session.go app/point_cloud_gpu.go app/point_cloud_gpu_test.go app/point_cloud_render_test.go app/point_cloud_ribbon_test.go app/ribbon.go renderer/point_markers.go head/ui/chrome_ribbon.go head/ui/chrome_viewport.go head/ui/point_cloud_gpu.go head/ui/point_cloud_ribbon_shot_test.go head/internal/native/viewport_point_upload_test.go
git -C ../Oblikovati.API diff -- types/point_cloud_display_mode.go
```

Expected: diff contains only the grey default color, point-size session/control/upload path, intensity histogram model/rendering, and the API comment correction.
